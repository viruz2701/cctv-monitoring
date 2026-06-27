// ═══════════════════════════════════════════════════════════════════════
// Package edge — Edge Agent SL-4 Security (P3-NICE.3)
//
// Соответствие:
//   - IEC 62443 SL-4 (Zone 5: Edge)
//   - Приказ ОАЦ №66 п. 7.18 (защита конечных узлов)
//   - СТБ 34.101.30 (belt/bign/bash криптография)
//   - ISO 27001 A.13.2 (сетевая безопасность)
// ═══════════════════════════════════════════════════════════════════════

package edge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
)

// ═══ Constants ═════════════════════════════════════════════════════════

const (
	// Default mTLS settings
	defaultHandshakeTimeout = 10 * time.Second
	defaultKeepAlivePeriod  = 30 * time.Second
	defaultReconnectDelay   = 5 * time.Second
	maxReconnectAttempts    = 10

	// Agent version for identification
	AgentVersion = "1.0.0"

	// Heartbeat interval for health monitoring
	heartbeatInterval = 15 * time.Second
)

// ═══ Types ═════════════════════════════════════════════════════════════

// AgentStatus represents the current state of the edge agent.
type AgentStatus string

const (
	StatusDisconnected AgentStatus = "disconnected"
	StatusConnecting   AgentStatus = "connecting"
	StatusConnected    AgentStatus = "connected"
	StatusError        AgentStatus = "error"
)

// AgentConfig holds the configuration for the edge agent.
// Все секретные значения загружаются из env vars, НЕ из конфиг-файла.
type AgentConfig struct {
	// Identity (Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация)
	DeviceID   string // device-id (из env: EDGE_DEVICE_ID)
	InstanceID string // unique instance identifier

	// mTLS settings (Приказ ОАЦ №66 п. 7.18.2 — mTLS 1.3)
	ServerAddr    string // backend server address (env: EDGE_SERVER_ADDR)
	CertFile      string // client certificate path (env: EDGE_CERT_FILE)
	KeyFile       string // client key path (env: EDGE_KEY_FILE)
	CAFile        string // CA certificate path (env: EDGE_CA_FILE)
	TLSServerName string // TLS SNI (env: EDGE_TLS_SERVER_NAME)

	// Tamper detection (Приказ ОАЦ №66 п. 7.18.3)
	IntegrityCheckInterval time.Duration // how often to verify integrity (env: EDGE_INTEGRITY_INTERVAL)
	ExpectedHash           string        // expected binary hash (env: EDGE_EXPECTED_HASH)

	// Operational
	Logger *slog.Logger
}

// Agent represents a secure edge agent with mTLS connectivity.
// Реализует требования SL-4 по защите конечных узлов.
type Agent struct {
	config AgentConfig
	status AgentStatus
	mu     sync.RWMutex

	conn   net.Conn
	tlsCfg *tls.Config
	ctx    context.Context
	cancel context.CancelFunc

	// Channels
	stopCh    chan struct{}
	doneCh    chan struct{}
	heartbeat *time.Ticker

	// Security (P3-NICE.3)
	integrityChecker *IntegrityChecker
	eventLog         []SecurityEvent
	eventMu          sync.RWMutex

	logger *slog.Logger
}

// ═══ Construction ═════════════════════════════════════════════════════

// NewAgent creates a new Edge Agent with the given configuration.
// Загружает TLS сертификаты и настраивает mTLS 1.3.
func NewAgent(cfg AgentConfig) (*Agent, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "edge-agent", "device_id", cfg.DeviceID)

	// Validate required config
	if cfg.ServerAddr == "" {
		return nil, fmt.Errorf("edge: server address is required (set EDGE_SERVER_ADDR)")
	}
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("edge: client cert and key are required (set EDGE_CERT_FILE, EDGE_KEY_FILE)")
	}
	if cfg.CAFile == "" {
		return nil, fmt.Errorf("edge: CA file is required (set EDGE_CA_FILE)")
	}

	// Загружаем сертификаты для mTLS
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("edge: failed to load client cert: %w", err)
	}

	// Загружаем CA для верификации сервера
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("edge: failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("edge: failed to parse CA certificate")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		// Только TLS 1.3 (IEC 62443 SL-4)
		MinVersion: tls.VersionTLS13,
		// Требуем сертификат клиента (mTLS)
		ClientAuth: tls.RequireAndVerifyClientCert,
		// SNI для правильной маршрутизации
		ServerName: cfg.TLSServerName,
		// Безопасные cipher suites для TLS 1.3
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Настройка integrity checker
	var integrityChecker *IntegrityChecker
	if cfg.IntegrityCheckInterval > 0 && cfg.ExpectedHash != "" {
		integrityChecker = NewIntegrityChecker(cfg.ExpectedHash, cfg.IntegrityCheckInterval, logger)
	}

	if cfg.IntegrityCheckInterval == 0 {
		cfg.IntegrityCheckInterval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger.Info("edge agent initialized",
		"server", cfg.ServerAddr,
		"tls_version", "1.3",
		"integrity_check", integrityChecker != nil,
	)

	return &Agent{
		config:           cfg,
		status:           StatusDisconnected,
		tlsCfg:           tlsCfg,
		ctx:              ctx,
		cancel:           cancel,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
		integrityChecker: integrityChecker,
		logger:           logger,
	}, nil
}

// ═══ Lifecycle ═══════════════════════════════════════════════════════

// Start establishes the mTLS connection and begins the agent loop.
// Блокирующий вызов — запускайте в горутине.
func (a *Agent) Start() error {
	a.setStatus(StatusConnecting)
	a.logger.Info("starting edge agent")

	// Запускаем integrity checking если настроен
	if a.integrityChecker != nil {
		go a.integrityChecker.Start(a.stopCh)
	}

	// Основной цикл подключения с реконнектом
	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		select {
		case <-a.stopCh:
			a.setStatus(StatusDisconnected)
			close(a.doneCh)
			return nil
		default:
		}

		err := a.connect()
		if err != nil {
			a.logger.Warn("connection attempt failed", "attempt", attempt+1, "error", err)
			a.logSecurityEvent(SecurityEvent{
				Type:      EventConnectionFailure,
				Severity:  SevWarning,
				Message:   fmt.Sprintf("Connection attempt %d failed: %v", attempt+1, err),
				Timestamp: time.Now(),
			})

			if attempt < maxReconnectAttempts-1 {
				time.Sleep(defaultReconnectDelay)
			}
			continue
		}

		// Connected — запускаем heartbeat и держим соединение
		a.setStatus(StatusConnected)
		a.logger.Info("edge agent connected")
		a.logSecurityEvent(SecurityEvent{
			Type:      EventConnectionEstablished,
			Severity:  SevInfo,
			Message:   "mTLS connection established",
			Timestamp: time.Now(),
		})

		a.handleConnection()
		return nil
	}

	a.setStatus(StatusError)
	close(a.doneCh)
	return fmt.Errorf("edge: max reconnect attempts (%d) reached", maxReconnectAttempts)
}

// Stop gracefully shuts down the edge agent.
func (a *Agent) Stop() {
	a.logger.Info("stopping edge agent")
	a.cancel()
	close(a.stopCh)

	if a.conn != nil {
		a.conn.Close()
	}

	<-a.doneCh
	a.setStatus(StatusDisconnected)
	a.logger.Info("edge agent stopped")
}

// ═══ Connection management ═══════════════════════════════════════════

func (a *Agent) connect() error {
	dialer := &tls.Dialer{
		Config: a.tlsCfg,
	}

	conn, err := dialer.DialContext(a.ctx, "tcp", a.config.ServerAddr)
	if err != nil {
		return fmt.Errorf("mTLS dial failed: %w", err)
	}

	a.conn = conn
	return nil
}

func (a *Agent) handleConnection() {
	defer func() {
		if a.conn != nil {
			a.conn.Close()
		}
		a.setStatus(StatusDisconnected)
		a.logger.Info("edge agent disconnected")
		a.logSecurityEvent(SecurityEvent{
			Type:      EventConnectionClosed,
			Severity:  SevInfo,
			Message:   "connection closed",
			Timestamp: time.Now(),
		})
	}()

	a.heartbeat = time.NewTicker(heartbeatInterval)
	defer a.heartbeat.Stop()

	for {
		select {
		case <-a.stopCh:
			return

		case <-a.heartbeat.C:
			// Send heartbeat with device info
			msg := fmt.Sprintf("HEARTBEAT %s %s", a.config.DeviceID, AgentVersion)
			if err := a.sendMessage(msg); err != nil {
				a.logger.Warn("heartbeat failed", "error", err)
				return
			}
		}
	}
}

func (a *Agent) sendMessage(msg string) error {
	if a.conn == nil {
		return fmt.Errorf("no connection")
	}

	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := a.conn.Write([]byte(msg + "\n"))
	return err
}

// ═══ Status ═════════════════════════════════════════════════════════

// Status returns the current agent status.
func (a *Agent) Status() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *Agent) setStatus(s AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = s
}

// GetStatusInfo returns detailed status information.
func (a *Agent) GetStatusInfo() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	info := map[string]interface{}{
		"status":          string(a.status),
		"device_id":       a.config.DeviceID,
		"instance_id":     a.config.InstanceID,
		"agent_version":   AgentVersion,
		"server":          a.config.ServerAddr,
		"tls_version":     "1.3",
		"integrity_check": a.integrityChecker != nil,
	}

	if a.integrityChecker != nil {
		info["last_integrity_check"] = a.integrityChecker.LastCheck()
		info["integrity_valid"] = a.integrityChecker.IsValid()
	}

	return info
}

// ═══ Security events ═════════════════════════════════════════════════

func (a *Agent) logSecurityEvent(event SecurityEvent) {
	a.eventMu.Lock()
	defer a.eventMu.Unlock()

	// Keep last 1000 events
	if len(a.eventLog) >= 1000 {
		a.eventLog = a.eventLog[1:]
	}
	a.eventLog = append(a.eventLog, event)

	// Log to structured logger
	level := slog.LevelInfo
	switch event.Severity {
	case SevWarning:
		level = slog.LevelWarn
	case SevCritical:
		level = slog.LevelError
	}
	a.logger.Log(a.ctx, level, event.Message, "event_type", event.Type, "severity", event.Severity)
}

// GetSecurityEvents returns the security event log.
func (a *Agent) GetSecurityEvents() []SecurityEvent {
	a.eventMu.RLock()
	defer a.eventMu.RUnlock()

	result := make([]SecurityEvent, len(a.eventLog))
	copy(result, a.eventLog)
	return result
}
