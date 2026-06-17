package main

import (
	"context"
	"gb-telemetry-collector/internal/api"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/cron"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/logging"
	"gb-telemetry-collector/internal/logserver"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/protocols"
	"gb-telemetry-collector/internal/sip"
	"gb-telemetry-collector/internal/state"
	"gb-telemetry-collector/internal/telegram"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

func main() {
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
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "gb_user"),
		Password: getEnv("DB_PASSWORD", "gb_password"),
		DBName:   getEnv("DB_NAME", "gb_telemetry"),
		SSLMode:  "disable",
	}
	database, err := db.New(dbCfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Создание пользователя admin@example.com
	_, err = database.GetUserByUsername("admin@example.com")
	if err != nil {
		hashed, _ := auth.HashPassword("admin123")
		database.CreateUser("admin@example.com", hashed, "admin", "admin@example.com", nil)
		logger.Info("Created admin user: admin@example.com / admin123")
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
	defer logServer.Stop()

	// --- SIP-сервер ---
	sipHandler := sip.NewSIPHandler(stateWrapper, logger, cfg.GB28181)
	if err := sipHandler.Start(ctx); err != nil {
		logger.Error("Failed to start SIP server", "error", err)
		os.Exit(1)
	}
	defer sipHandler.Stop()

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
	defer protocolManager.StopAll()

	// --- API-сервер ---
	apiServer := api.NewServer(cfg.APIAddr, stateWrapper, logger, database, cfg.ImagesDir, cfg, sipHandler)
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

	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("API server failed", "error", err)
			cancel()
		}
	}()
	defer apiServer.Stop()

	if telegramBot != nil {
		defer telegramBot.Stop()
	}

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
	defer reaper.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
	cancel()
	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	sipHandler.Stop()
	apiServer.Stop()
	reaper.Stop()
	close(dbWriter.ch)
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
