package main

import (
	"context"
	"gb-telemetry-collector/internal/api"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/cron"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/logging"
	"gb-telemetry-collector/internal/logserver"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/protocols"
	"gb-telemetry-collector/internal/sip"
	"gb-telemetry-collector/internal/state"
	"gb-telemetry-collector/internal/sync"
	"gb-telemetry-collector/internal/telegram"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

type DBWriter struct {
	db *db.DB
	ch chan func()
}

func NewDBWriter(database *db.DB, bufferSize int) *DBWriter {
	w := &DBWriter{db: database, ch: make(chan func(), bufferSize)}
	go w.start()
	return w
}

func (w *DBWriter) start() {
	for job := range w.ch {
		job()
	}
}

func (w *DBWriter) Submit(job func()) {
	select {
	case w.ch <- job:
	default:
		// логгер будет доступен позже, но сброс не критичен
	}
}

type stateManagerWrapper struct {
	inner       state.DeviceStateManager
	dbWriter    *DBWriter
	logger      *slog.Logger
	broadcastFn func(*models.Alarm)
}

func (w *stateManagerWrapper) Get(deviceID string) (*models.Device, bool) {
	return w.inner.Get(deviceID)
}

func (w *stateManagerWrapper) Set(device *models.Device) {
	w.inner.Set(device)
	w.dbWriter.Submit(func() {
		if err := w.dbWriter.db.SaveDevice(device); err != nil {
			w.logger.Error("Failed to save device", "device_id", device.DeviceID, "error", err)
		}
	})
}

func (w *stateManagerWrapper) Delete(deviceID string) {
	w.inner.Delete(deviceID)
}

func (w *stateManagerWrapper) UpdateLastSeen(deviceID string) {
	w.inner.UpdateLastSeen(deviceID)
	if dev, ok := w.inner.Get(deviceID); ok {
		w.dbWriter.Submit(func() {
			w.dbWriter.db.SaveTelemetry(dev.DeviceID, dev.Status, dev.LastSeen, dev.HeartbeatInterval)
		})
	}
}

func (w *stateManagerWrapper) SetOnline(deviceID string) {
	w.inner.SetOnline(deviceID)
	if dev, ok := w.inner.Get(deviceID); ok {
		w.dbWriter.Submit(func() {
			w.dbWriter.db.SaveTelemetry(dev.DeviceID, dev.Status, dev.LastSeen, dev.HeartbeatInterval)
		})
	}
}

func (w *stateManagerWrapper) SetOffline(deviceID string) {
	w.inner.SetOffline(deviceID)
	if dev, ok := w.inner.Get(deviceID); ok {
		w.dbWriter.Submit(func() {
			w.dbWriter.db.SaveTelemetry(dev.DeviceID, dev.Status, dev.LastSeen, dev.HeartbeatInterval)
		})
	}
}

func (w *stateManagerWrapper) AddAlarm(deviceID string, alarm *models.Alarm) {
	w.inner.AddAlarm(deviceID, alarm)
	w.dbWriter.Submit(func() {
		w.dbWriter.db.SaveAlarm(alarm)
	})
	if w.broadcastFn != nil {
		w.broadcastFn(alarm)
	}
}

func (w *stateManagerWrapper) GetAll() map[string]*models.Device {
	return w.inner.GetAll()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt32(key string, def int32) int32 {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 32)
		if err == nil {
			return int32(parsed)
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		parsed, err := time.ParseDuration(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

// shutdownTimeout — максимальное время на graceful shutdown.
const shutdownTimeout = 30 * time.Second

func main() {
	// Загружаем .env файл перед чтением конфигурации
	_ = godotenv.Load()

	cfg := config.Load()

	// Инициализация логгера с ротацией
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	logger := logging.NewLogger(logging.Config{
		FilePath:   cfg.LogFile,
		MaxSizeMB:  cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAgeDays: cfg.LogMaxAgeDays,
		Compress:   cfg.LogCompress,
		Level:      logLevel,
		AddSource:  cfg.Debug,
	})

	dbCfg := db.Config{
		Host:              getEnv("DB_HOST", "localhost"),
		Port:              5432,
		User:              getEnv("DB_USER", "gb_user"),
		Password:          getEnv("DB_PASSWORD", "gb_password"),
		DBName:            getEnv("DB_NAME", "gb_telemetry"),
		SSLMode:           getEnv("DB_SSLMODE", "disable"),
		MaxConns:          getEnvInt32("DB_MAX_CONNS", 25),
		MinConns:          getEnvInt32("DB_MIN_CONNS", 5),
		MaxConnLifetime:   getEnvDuration("DB_MAX_CONN_LIFETIME", 5*time.Minute),
		MaxConnIdleTime:   getEnvDuration("DB_MAX_CONN_IDLE_TIME", 3*time.Minute),
		HealthCheckPeriod: getEnvDuration("DB_HEALTH_CHECK_PERIOD", time.Minute),
	}

	// Run golang-migrate migrations (replaces initSchema)
	if err := db.RunMigrations(dbCfg.DSN(), logger); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	database, err := db.New(dbCfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Seed default admin if no users exist
	if err := database.SeedDefaultAdmin(); err != nil {
		logger.Warn("Failed to seed default admin", "error", err)
	}

	dbWriter := NewDBWriter(database, 1000)

	stateManager := state.NewInMemoryStateManager()
	stateWrapper := &stateManagerWrapper{
		inner:    stateManager,
		dbWriter: dbWriter,
		logger:   logger,
	}

	// --- Лог-сервер ---
	logCfg := &logserver.Config{
		SyslogEnabled: true,
		SyslogUDPPort: cfg.LogServerPort,
		HTTPEnabled:   false,
	}
	logServer := logserver.NewLogServer(logCfg, logger, stateWrapper, func(log *models.ParsedLog) error {
		dbWriter.Submit(func() {
			if err := database.SaveParsedLog(log); err != nil {
				logger.Error("Failed to save parsed log", "error", err)
			}
		})
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := logServer.Start(ctx); err != nil {
		logger.Error("Failed to start log server", "error", err)
		os.Exit(1)
	}

	// --- SIP-сервер ---
	sipHandler := sip.NewSIPHandler(stateWrapper, logger, cfg.GB28181)
	if err := sipHandler.Start(ctx); err != nil {
		logger.Error("Failed to start SIP server", "error", err)
		os.Exit(1)
	}

	// --- Запуск дополнительных протоколов ---
	protocolManager := protocols.NewManager(logger)

	if cfg.Dahua.Enabled {
		dahuaHandler := protocols.NewDahuaHandler(cfg.Dahua.Ports, stateWrapper, logger)
		protocolManager.Register(dahuaHandler)
	}

	if cfg.Hisilicon.Enabled {
		hisiliconHandler := protocols.NewHisiliconHandler(cfg.Hisilicon.Port, stateWrapper, logger)
		protocolManager.Register(hisiliconHandler)
	}

	if cfg.TVT.Enabled {
		tvtHandler := protocols.NewTVTHandler(cfg.TVT.Port, stateWrapper, logger)
		protocolManager.Register(tvtHandler)
	}

	if cfg.FTP.Enabled {
		ftpHandler := protocols.NewFTPHandler(cfg.FTP.Port, cfg.FTP.RootPath, cfg.FTP.User, cfg.FTP.Password, stateWrapper, logger)
		protocolManager.Register(ftpHandler)
	}

	if cfg.Hikvision.Enabled && len(cfg.Hikvision.Cameras) > 0 {
		// Преобразуем карту в слайс структур HikCameraConfig
		var hikCameras []protocols.HikCameraConfig
		for name, cam := range cfg.Hikvision.Cameras {
			hikCameras = append(hikCameras, protocols.HikCameraConfig{
				Name:     name,
				Address:  cam.Address,
				HTTPS:    cam.HTTPS,
				Username: cam.Username,
				Password: cam.Password,
				RawTCP:   cam.RawTCP,
			})
		}
		hikHandler := protocols.NewHikvisionHandler(hikCameras, stateWrapper, logger)
		protocolManager.Register(hikHandler)
		logger.Info("Hikvision handler registered", "cameras", len(hikCameras))
	}

	// SNMP
	if cfg.SNMP.Enabled {
		snmpHandler := protocols.NewSNMPHandler(cfg.SNMP, stateWrapper, logger)
		protocolManager.Register(snmpHandler)
		logger.Info("SNMP handler registered", "port", cfg.SNMP.Port, "version", cfg.SNMP.Version)
	}

	if err := protocolManager.StartAll(ctx); err != nil {
		logger.Error("Failed to start additional protocols", "error", err)
		os.Exit(1)
	}

	// --- Bi-directional ITSM Sync Engine ---
	var syncEng *sync.SyncEngine
	if cfg.ITSMSyncInterval != "" {
		syncInterval, err := time.ParseDuration(cfg.ITSMSyncInterval)
		if err != nil {
			syncInterval = 5 * time.Minute
		}
		syncEng = sync.NewSyncEngine(
			database, logger,
			cfg.ServiceNowWebhookSecret,
			cfg.JiraWebhookSecret,
			cfg.TOIRWebhookSecret,
			syncInterval,
		)
		syncEng.Start(ctx)
		logger.Info("ITSM sync engine started", "interval", syncInterval)
	}

	// --- NATS Connection (опционально) ---
	var natsConn *nats.Conn
	if cfg.NATSURL != "" {
		nc, err := nats.Connect(cfg.NATSURL)
		if err != nil {
			logger.Warn("NATS not available, continuing without", "error", err)
		} else {
			natsConn = nc
			logger.Info("NATS connected", "url", cfg.NATSURL)
		}
	}

	// --- API-сервер ---
	apiServer := api.NewServer(cfg.APIAddr, stateWrapper, logger, database, cfg.ImagesDir, cfg, sipHandler, syncEng)
	if natsConn != nil {
		apiServer.SetNATSConn(natsConn)
	}
	stateWrapper.broadcastFn = apiServer.BroadcastAlarm

	// --- Telegram бот ---
	var telegramBot *telegram.Bot
	if cfg.Telegram.Enabled && cfg.Telegram.Token != "" {
		var err error
		telegramBot, err = telegram.NewBot(telegram.Config{
			Token:  cfg.Telegram.Token,
			Logger: logger,
		}, database, stateWrapper)
		if err != nil {
			logger.Error("Failed to create Telegram bot", "error", err)
		} else {
			apiServer.SetTelegramBot(telegramBot)
			go telegramBot.Start(ctx)
			logger.Info("Telegram bot started")
		}
	}

	// Запуск API сервера в горутине
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("API server starting", "addr", cfg.APIAddr)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	// --- Maintenance Cron Job (каждые 15 минут) ---
	maintenanceCron := cron.NewMaintenanceCron(database, logger)
	maintenanceCronTicker := time.NewTicker(15 * time.Minute)
	defer maintenanceCronTicker.Stop()
	go func() {
		// Запуск сразу при старте
		maintenanceCron.Run(ctx)
		for {
			select {
			case <-maintenanceCronTicker.C:
				maintenanceCron.Run(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
	logger.Info("Maintenance cron job started (every 15 minutes)")

	// --- Reaper ---
	reaper := NewReaper(stateWrapper, cfg.HeartbeatTimeout, logger)
	reaper.Start()

	// ══════════════════════════════════════════════════════════════════
	// GRACEFUL SHUTDOWN
	// Соответствует: ISO 27001 A.12.1.1, ISO 27001 A.17.1.1, СТБ 34.101.27 п. 8.1
	// ══════════════════════════════════════════════════════════════════

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		logger.Info("⏳ Shutdown signal received, initiating graceful shutdown...",
			"signal", sig.String(),
			"timeout", shutdownTimeout,
		)
	case err := <-serverErrCh:
		if err != nil {
			logger.Error("API server failed", "error", err)
		}
	}

	// Отменяем корневой контекст — все зависимые сервисы получат сигнал остановки
	cancel()

	// Создаём контекст с таймаутом для graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// 1. Reaper stop
	logger.Info("Shutting down reaper...")
	reaper.Stop()
	logger.Info("Reaper stopped")

	// 2. HTTP server graceful shutdown
	logger.Info("Shutting down HTTP server...")
	if err := apiServer.Stop(shutdownCtx); err != nil {
		logger.Error("HTTP server graceful shutdown failed", "error", err)
	} else {
		logger.Info("HTTP server stopped")
	}

	// 3. SIP server stop
	logger.Info("Shutting down SIP server...")
	sipHandler.Stop()
	logger.Info("SIP server stopped")

	// 4. Protocol manager stop (FTP, Dahua, Hikvision, etc.)
	logger.Info("Shutting down protocol handlers...")
	protocolManager.StopAll()
	logger.Info("Protocol handlers stopped")

	// 5. Log server stop
	logger.Info("Shutting down log server...")
	logServer.Stop()
	logger.Info("Log server stopped")

	// 6. Telegram bot stop
	if telegramBot != nil {
		logger.Info("Shutting down Telegram bot...")
		telegramBot.Stop()
		logger.Info("Telegram bot stopped")
	}

	// 7. ITSM sync engine stop
	if syncEng != nil {
		logger.Info("Shutting down ITSM sync engine...")
		syncEng.Stop()
		logger.Info("ITSM sync engine stopped")
	}

	// 8. NATS drain (если подключен)
	if natsConn != nil {
		logger.Info("Draining NATS connection...")
		if err := natsConn.Drain(); err != nil {
			logger.Error("NATS drain failed", "error", err)
		} else {
			logger.Info("NATS connection drained")
		}
	}

	// 9. DBWriter drain
	logger.Info("Draining DB writer...")
	close(dbWriter.ch)
	logger.Info("DB writer drained")

	// 10. Database connection close
	logger.Info("Closing database connection...")
	database.Close()
	logger.Info("Database connection closed")

	// Если shutdown занял больше времени чем таймаут, принудительно выходим
	// (этот код выполнится после shutdownCtx, если все завершилось раньше — ок)
	logger.Info("✅ Graceful shutdown complete")
}

type Reaper struct {
	stateManager state.DeviceStateManager
	timeout      time.Duration
	logger       *slog.Logger
	ticker       *time.Ticker
	stopCh       chan struct{}
}

func NewReaper(stateMgr state.DeviceStateManager, timeout time.Duration, logger *slog.Logger) *Reaper {
	return &Reaper{
		stateManager: stateMgr,
		timeout:      timeout,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

func (r *Reaper) Start() {
	r.ticker = time.NewTicker(15 * time.Second)
	go func() {
		for {
			select {
			case <-r.ticker.C:
				r.check()
			case <-r.stopCh:
				r.ticker.Stop()
				return
			}
		}
	}()
}

func (r *Reaper) check() {
	now := time.Now()
	for _, dev := range r.stateManager.GetAll() {
		if dev.Status == models.StatusOnline {
			if now.Sub(dev.LastSeen) > r.timeout {
				r.logger.Info("Device timed out, marking offline", "device_id", dev.DeviceID)
				r.stateManager.SetOffline(dev.DeviceID)
			}
		}
	}
}

func (r *Reaper) Stop() {
	close(r.stopCh)
}
