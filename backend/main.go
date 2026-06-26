package main

import (
	"context"
	"gb-telemetry-collector/internal/api"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/cron"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/events"
	"gb-telemetry-collector/internal/featureflag"
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
	"strings"
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

// extractPort извлекает номер порта из ошибки вида "listen udp 0.0.0.0:515: bind: permission denied"
func extractPort(errStr string) string {
	parts := strings.Split(errStr, ":")
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[len(parts)-3])
	}
	return "unknown"
}

// retryStart запускает сервис с экспоненциальной задержкой при ошибках порта.
// Не блокирует startup — возвращает функцию отмены.
// Используется для: logServer, sipHandler, protocolManager.
func retryStart(ctx context.Context, name string, logger *slog.Logger, fn func(context.Context) error) func() {
	stopCh := make(chan struct{})
	go func() {
		backoff := 1 * time.Second
		maxBackoff := 60 * time.Second
		attempt := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			default:
			}
			attempt++
			err := fn(ctx)
			if err == nil {
				logger.Info("Service started successfully", "service", name)
				return
			}
			errStr := err.Error()
			// Retry только на "address already in use" (порт освободится)
			// "permission denied" (порт <1024) не решается ожиданием — не retry
			if strings.Contains(errStr, "bind: permission denied") {
				logger.Warn("Port requires elevated privileges, service unavailable",
					"service", name, "port", extractPort(errStr), "error", err,
				)
				return
			}
			if !strings.Contains(errStr, "address already in use") &&
				!strings.Contains(errStr, "cannot assign requested address") {
				logger.Error("Service failed with non-retryable error", "service", name, "error", err)
				return
			}
			logger.Warn("Port busy, will retry",
				"service", name,
				"attempt", attempt,
				"backoff", backoff,
			)
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}()
	return func() { close(stopCh) }
}

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

	// --- NATS Connection (ARCH-01: нужен до State Manager) ---
	// В production (NATSRequired=true) — обязателен, startup фейлится при недоступности.
	// В dev режиме — опционально, работаем без событийной шины.
	var natsConn *nats.Conn
	if cfg.NATSURL != "" {
		nc, err := nats.Connect(cfg.NATSURL)
		if err != nil {
			if cfg.NATSRequired {
				logger.Error("NATS connection required but unavailable", "url", cfg.NATSURL, "error", err)
				os.Exit(1)
			}
			logger.Debug("NATS not available, continuing without", "error", err)
		} else {
			natsConn = nc
			logger.Info("NATS connected", "url", cfg.NATSURL)
		}
	} else if cfg.NATSRequired {
		logger.Error("NATS connection required but nats_url is not configured")
		os.Exit(1)
	}

	// --- NATS JetStream KV State Manager (ARCH-01) ---
	// Если NATS доступен — используем распределённое состояние для horizontal scaling.
	// Если UseNATSKV=true и NATSRequired=true — JetStream обязателен, без fallback.
	// В dev режиме — InMemoryStateManager (dev mode).
	var (
		stateManager     state.DeviceStateManager
		jetStreamManager *state.JetStreamStateManager // для graceful shutdown
	)
	if natsConn != nil && cfg.UseNATSKV {
		js, err := natsConn.JetStream()
		if err != nil {
			if cfg.NATSRequired {
				logger.Error("NATS JetStream required but not available", "error", err)
				os.Exit(1)
			}
			logger.Warn("NATS JetStream not available, falling back to InMemoryStateManager", "error", err)
			stateManager = state.NewInMemoryStateManager()
		} else {
			jsMgr, err := state.NewJetStreamStateManager(js, logger)
			if err != nil {
				if cfg.NATSRequired {
					logger.Error("JetStream KV state manager creation failed", "error", err)
					os.Exit(1)
				}
				logger.Warn("JetStream KV state manager failed, falling back to InMemoryStateManager", "error", err)
				stateManager = state.NewInMemoryStateManager()
			} else {
				stateManager = jsMgr
				jetStreamManager = jsMgr
				logger.Info("ARCH-01: Using JetStream KV distributed state manager")
			}
		}
	} else {
		stateManager = state.NewInMemoryStateManager()
		if natsConn == nil {
			logger.Debug("NATS not connected, using InMemoryStateManager")
		}
	}

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

	// Goroutine retry: если порт 515 занят — ждём и перезапускаем
	stopLogServer := retryStart(ctx, "syslog", logger, func(ctx context.Context) error {
		return logServer.Start(ctx)
	})
	defer stopLogServer()

	// --- SIP-сервер ---
	sipHandler := sip.NewSIPHandler(stateWrapper, logger, cfg.GB28181)
	// Goroutine retry: если порт 5060 занят — ждём и перезапускаем
	stopSIP := retryStart(ctx, "gb28181/sip", logger, func(ctx context.Context) error {
		return sipHandler.Start(ctx)
	})
	defer stopSIP()

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

	// Goroutine retry для protocolManager (dahua, hisilicon, tvt, ftp, snmp)
	// StartAll не возвращает ошибку (логирует внутри), но retry нужен на случай
	// если порты освободятся позже
	stopProtocols := retryStart(ctx, "protocols", logger, func(ctx context.Context) error {
		return protocolManager.StartAll(ctx)
	})
	defer stopProtocols()

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

	// --- Event Store (DM-1.2.2: NATS JetStream + S3 Cold Storage) ---
	var eventStore *events.EventStore
	if cfg.EventStore.Enabled {
		esCfg := events.DefaultEventStoreConfig()
		esCfg.NATSURL = cfg.EventStore.NATSURL
		esCfg.NATSCreds = cfg.EventStore.NATSCreds
		esCfg.NATSUseTLS = cfg.EventStore.NATSTLS

		// Cold storage (опционально)
		if cfg.EventStore.S3Endpoint != "" {
			esCfg.S3Endpoint = cfg.EventStore.S3Endpoint
			esCfg.S3Region = cfg.EventStore.S3Region
			esCfg.S3Bucket = cfg.EventStore.S3Bucket
			esCfg.S3AccessKey = cfg.EventStore.S3AccessKey
			esCfg.S3SecretKey = cfg.EventStore.S3SecretKey
			esCfg.S3UseTLS = cfg.EventStore.S3UseTLS
		}

		if cfg.EventStore.HotRetentionHours > 0 {
			esCfg.HotRetention = time.Duration(cfg.EventStore.HotRetentionHours) * time.Hour
		}
		if cfg.EventStore.ColdRetentionHours > 0 {
			esCfg.ColdRetention = time.Duration(cfg.EventStore.ColdRetentionHours) * time.Hour
		}

		esCfg.Logger = logger

		var err error
		eventStore, err = events.NewEventStore(esCfg)
		if err != nil {
			logger.Warn("Event Store initialization failed, continuing without", "error", err)
		} else {
			logger.Info("Event Store initialized",
				"cold_storage", cfg.EventStore.S3Endpoint != "",
				"hot_retention", esCfg.HotRetention,
				"cold_retention", esCfg.ColdRetention,
			)

			// Публикуем событие запуска системы
			systemEvent := eventStore.NewRecord(
				events.SourceSystem,
				"system.startup",
				"cctv-backend",
				map[string]interface{}{
					"version":    "1.0.0",
					"hostname":   getEnv("HOSTNAME", "unknown"),
					"go_version": "1.25",
				},
			)
			_ = eventStore.Store(ctx, systemEvent)
		}
	}

	// --- Feature Flag Manager (F-0.2.4) ---
	ffManager, err := featureflag.NewManager(database, logger)
	if err != nil {
		logger.Error("Failed to create feature flag manager", "error", err)
		os.Exit(1)
	}

	// --- API-сервер ---
	apiServer := api.NewServer(cfg.APIAddr, stateWrapper, logger, database, cfg.ImagesDir, cfg, sipHandler, syncEng)
	if natsConn != nil {
		apiServer.SetNATSConn(natsConn, cfg.NATSRequired)
	}
	apiServer.SetFeatureFlagsManager(ffManager)
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

	// --- Materialized View Auto-Refresh (каждые 60 минут) ---
	// P3-2.1: REFRESH MATERIALIZED VIEW CONCURRENTLY
	mvTicker := time.NewTicker(60 * time.Minute)
	defer mvTicker.Stop()
	go func() {
		// Запуск сразу при старте
		if err := maintenanceCron.RefreshMaterializedViews(ctx); err != nil {
			logger.Error("initial MV refresh failed", "error", err)
		}
		for {
			select {
			case <-mvTicker.C:
				if err := maintenanceCron.RefreshMaterializedViews(ctx); err != nil {
					logger.Error("MV refresh failed", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	logger.Info("Materialized view auto-refresh started (every 60 minutes)")

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

	// 8. Event Store shutdown (DM-1.2.2)
	if eventStore != nil {
		logger.Info("Shutting down Event Store...")
		// Публикуем событие остановки
		shutdownEvent := eventStore.NewRecord(
			events.SourceSystem,
			"system.shutdown",
			"cctv-backend",
			map[string]interface{}{
				"reason": "graceful_shutdown",
				"signal": "SIGTERM",
			},
		)
		_ = eventStore.StoreSync(shutdownCtx, shutdownEvent)

		if err := eventStore.Close(); err != nil {
			logger.Error("Event Store close failed", "error", err)
		} else {
			logger.Info("Event Store shut down")
		}
	}

	// 9. Feature Flag manager stop
	if ffManager != nil {
		logger.Info("Shutting down Feature Flag manager...")
		ffManager.Stop()
	}

	// 10. JetStream KV State Manager stop (ARCH-01)
	if jetStreamManager != nil {
		logger.Info("Shutting down JetStream KV state manager...")
		jetStreamManager.Stop()
		logger.Info("JetStream KV state manager stopped")
	}

	// 11. NATS drain (если подключен)
	if natsConn != nil {
		logger.Info("Draining NATS connection...")
		if err := natsConn.Drain(); err != nil {
			logger.Error("NATS drain failed", "error", err)
		} else {
			logger.Info("NATS connection drained")
		}
	}

	// 12. DBWriter drain
	logger.Info("Draining DB writer...")
	close(dbWriter.ch)
	logger.Info("DB writer drained")

	// 13. Database connection close
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
