package logserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/respond"
	"gb-telemetry-collector/internal/state"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config – конфигурация лог-сервера
type Config struct {
	SyslogEnabled bool `yaml:"syslog_enabled" json:"syslog_enabled"`
	SyslogUDPPort int  `yaml:"syslog_udp_port" json:"syslog_udp_port"`
	SyslogTCPPort int  `yaml:"syslog_tcp_port" json:"syslog_tcp_port"`
	HTTPEnabled   bool `yaml:"http_enabled" json:"http_enabled"`
	HTTPPort      int  `yaml:"http_port" json:"http_port"`
}

// LogServer – основной сервер логов
type LogServer struct {
	config     *Config
	logger     *slog.Logger
	stateMgr   state.DeviceStateManager // <-- КРИТИЧЕСКИ ВАЖНО: связь со стейтом
	dbSaver    func(*models.ParsedLog) error
	httpServer *http.Server
	syslogUDP  *net.UDPConn
	syslogTCP  net.Listener
	mu         sync.Mutex
}

func NewLogServer(cfg *Config, logger *slog.Logger, stateMgr state.DeviceStateManager, saver func(*models.ParsedLog) error) *LogServer {
	return &LogServer{
		config:   cfg,
		logger:   logger,
		stateMgr: stateMgr,
		dbSaver:  saver,
	}
}

func (s *LogServer) Start(ctx context.Context) error {
	if s.config.SyslogEnabled {
		if err := s.startSyslogUDP(); err != nil {
			s.logger.Error("Failed to start Syslog UDP", "error", err)
		}
		if err := s.startSyslogTCP(); err != nil {
			s.logger.Error("Failed to start Syslog TCP", "error", err)
		}
	}
	if s.config.HTTPEnabled {
		s.startHTTP()
	}
	return nil
}

func (s *LogServer) startSyslogUDP() error {
	addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: s.config.SyslogUDPPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.syslogUDP = conn
	go func() {
		buf := make([]byte, 65536)
		for {
			n, remote, err := conn.ReadFromUDP(buf)
			if err != nil {
				return // Закрыли сокет
			}
			go s.processLogMessage(buf[:n], remote.IP.String(), "syslog_udp")
		}
	}()
	s.logger.Info("Syslog UDP server started", "port", s.config.SyslogUDPPort)
	return nil
}

func (s *LogServer) startSyslogTCP() error {
	if s.config.SyslogTCPPort == 0 {
		return nil
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.config.SyslogTCPPort))
	if err != nil {
		return err
	}
	s.syslogTCP = listener
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go s.handleTCPConnection(conn)
		}
	}()
	s.logger.Info("Syslog TCP server started", "port", s.config.SyslogTCPPort)
	return nil
}

func (s *LogServer) handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	remoteIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
	for scanner.Scan() {
		go s.processLogMessage(scanner.Bytes(), remoteIP, "syslog_tcp")
	}
}

func (s *LogServer) startHTTP() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/v1/logs/raw", s.handleHTTPLog)

	s.httpServer = &http.Server{
		Addr:    ":" + strconv.Itoa(s.config.HTTPPort),
		Handler: r,
	}
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP log server failed", "error", err)
		}
	}()
	s.logger.Info("HTTP log server started", "port", s.config.HTTPPort)
}

// processLogMessage - единая точка входа для всех протоколов
func (s *LogServer) processLogMessage(data []byte, remoteIP string, source string) {
	raw := string(data)

	// 1. УМНЫЙ ПОИСК УСТРОЙСТВА
	// Сначала ищем по IP в Location (работает, если IP белый или мы знаем маппинг)
	deviceID := fmt.Sprintf("syslog_%s", strings.ReplaceAll(remoteIP, ".", "_"))
	dev, exists := s.stateMgr.Get(deviceID)

	if !exists {
		// Если не нашли по ID, ищем по IP в поле Location среди всех устройств
		for _, d := range s.stateMgr.GetAll() {
			if d.Location == remoteIP {
				deviceID = d.DeviceID
				dev = d
				exists = true
				break
			}
		}
	}

	// 2. ОБНОВЛЕНИЕ СТАТУСА (Камера жива, раз шлет логи)
	if exists {
		s.stateMgr.UpdateLastSeen(deviceID)
		if dev.Status == models.StatusOffline {
			s.stateMgr.SetOnline(deviceID)
		}
	} else {
		// Авто-регистрация неизвестного устройства
		s.stateMgr.Set(&models.Device{
			DeviceID:     deviceID,
			Status:       models.StatusOnline,
			LastSeen:     time.Now(),
			RegisteredAt: time.Now(),
			VendorType:   "syslog_auto",
			Name:         fmt.Sprintf("Syslog Device %s", remoteIP),
			Location:     remoteIP,
		})
	}

	// 3. ПАРСИНГ И СОХРАНЕНИЕ
	level, code, message, vendor := parseSyslogContent(raw)

	parsedLog := &models.ParsedLog{
		Time:      time.Now(),
		DeviceID:  deviceID,
		LogLevel:  level,
		EventCode: code,
		Message:   message,
		Source:    source,
		Raw:       raw,
	}
	if err := s.dbSaver(parsedLog); err != nil {
		s.logger.Error("Failed to save parsed log", "error", err)
	}

	// 4. ГЕНЕРАЦИЯ АЛЕРТА
	if level == "ERROR" || level == "WARN" {
		priority := models.AlarmPriorityMedium
		method := models.AlarmMethodEquipmentFault
		if level == "WARN" {
			priority = models.AlarmPriorityHigh
			method = models.AlarmMethodMotionDetection
		}

		s.stateMgr.AddAlarm(deviceID, &models.Alarm{
			DeviceID:    deviceID,
			Priority:    priority,
			Method:      method,
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("[%s] %s", vendor, message),
		})
	}
}

// parseSyslogContent - эвристический парсер для Hikvision, Dahua и Generic
func parseSyslogContent(raw string) (level string, eventCode int, message string, vendor string) {
	lower := strings.ToLower(raw)
	vendor = "generic"

	// Определение вендора
	if strings.Contains(lower, "hikvision") || strings.Contains(lower, "ds-") {
		vendor = "hikvision"
	} else if strings.Contains(lower, "dahua") || strings.Contains(lower, "dh-") {
		vendor = "dahua"
	}

	// Определение события
	switch {
	case strings.Contains(lower, "hdd") || strings.Contains(lower, "disk") || strings.Contains(lower, "storage"):
		return "ERROR", 6, "Storage/HDD Error detected", vendor
	case strings.Contains(lower, "video loss") || strings.Contains(lower, "signal loss"):
		return "ERROR", 5, "Video Signal Lost", vendor
	case strings.Contains(lower, "motion") || strings.Contains(lower, "tripwire") || strings.Contains(lower, "intrusion"):
		return "WARN", 1, "Motion/Intrusion Detection", vendor
	case strings.Contains(lower, "login") || strings.Contains(lower, "online"):
		return "INFO", 0, "Status update / Login", vendor
	default:
		return "INFO", 0, raw, vendor
	}
}

func (s *LogServer) handleHTTPLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
		Log      string `json:"log"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "Bad request")
		return
	}

	// Для HTTP мы уже знаем DeviceID
	if dev, ok := s.stateMgr.Get(req.DeviceID); ok {
		s.stateMgr.UpdateLastSeen(req.DeviceID)
		if dev.Status == models.StatusOffline {
			s.stateMgr.SetOnline(req.DeviceID)
		}
	}

	level, code, message, _ := parseSyslogContent(req.Log)
	parsedLog := &models.ParsedLog{
		Time:      time.Now(),
		DeviceID:  req.DeviceID,
		LogLevel:  level,
		EventCode: code,
		Message:   message,
		Source:    "http",
		Raw:       req.Log,
	}
	if err := s.dbSaver(parsedLog); err != nil {
		respond.Error(w, http.StatusInternalServerError, "Internal error")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *LogServer) Stop() {
	if s.syslogUDP != nil {
		s.syslogUDP.Close()
	}
	if s.syslogTCP != nil {
		s.syslogTCP.Close()
	}
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

func DefaultConfig() *Config {
	return &Config{
		SyslogEnabled: true,
		SyslogUDPPort: 1514, // 514 требует root, лучше 1514
		SyslogTCPPort: 1514,
		HTTPEnabled:   true,
		HTTPPort:      8083, // 8082 занят p2p-gateway!
	}
}
