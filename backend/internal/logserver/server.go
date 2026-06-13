package logserver

import (
    "context"
    "encoding/json"
    "net"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "log/slog"

    "gb-telemetry-collector/internal/models"
)

// Config – конфигурация лог-сервера
type Config struct {
    SyslogEnabled bool          `yaml:"syslog_enabled"`
    SyslogPort    int           `yaml:"syslog_port"`
    SyslogProto   string        `yaml:"syslog_proto"`
    HTTPEnabled   bool          `yaml:"http_enabled"`
    HTTPPort      int           `yaml:"http_port"`
    FTPEnabled    bool          `yaml:"ftp_enabled"`
    FTPInterval   time.Duration `yaml:"ftp_interval"`
    FTPAddress    string        `yaml:"ftp_address"`
    FTPUser       string        `yaml:"ftp_user"`
    FTPPassword   string        `yaml:"ftp_password"`
    FTPRemotePath string        `yaml:"ftp_remote_path"`
    ParserRules   []LogParserRule `yaml:"parser_rules"`
}

// LogParserRule – правило парсинга
type LogParserRule struct {
    Name    string `yaml:"name"`
    Regex   string `yaml:"regex"`
    Level   string `yaml:"level"`
    Enabled bool   `yaml:"enabled"`
}

// LogParser – парсер логов (упрощённый)
type LogParser struct {
    rules []LogParserRule
}

func NewLogParser(rules []LogParserRule) *LogParser {
    return &LogParser{rules: rules}
}

func (p *LogParser) Parse(raw string) (level string, eventCode int, message string) {
    // TODO: реализовать парсинг по правилам
    return "INFO", 0, raw
}

// FTPPoller – заглушка
type FTPPoller struct {
    config   *Config
    logger   *slog.Logger
    stopCh   chan struct{}
    callback func(filename string, content []byte)
}

func NewFTPPoller(cfg *Config, logger *slog.Logger, callback func(string, []byte)) *FTPPoller {
    return &FTPPoller{
        config:   cfg,
        logger:   logger,
        stopCh:   make(chan struct{}),
        callback: callback,
    }
}

func (f *FTPPoller) Start() {
    // заглушка
}

func (f *FTPPoller) Stop() {
    close(f.stopCh)
}

// LogServer – основной сервер логов
type LogServer struct {
    config      *Config
    logger      *slog.Logger
    parser      *LogParser
    dbSaver     func(*models.ParsedLog) error
    httpServer  *http.Server
    syslogConn  *net.UDPConn
    ftpPoller   *FTPPoller
}

func NewLogServer(cfg *Config, logger *slog.Logger, saver func(*models.ParsedLog) error) *LogServer {
    return &LogServer{
        config:  cfg,
        logger:  logger,
        parser:  NewLogParser(cfg.ParserRules),
        dbSaver: saver,
    }
}

func (s *LogServer) Start(ctx context.Context) error {
    if s.config.SyslogEnabled {
        if err := s.startSyslog(); err != nil {
            return err
        }
    }
    if s.config.HTTPEnabled {
        s.startHTTP()
    }
    if s.config.FTPEnabled {
        s.ftpPoller = NewFTPPoller(s.config, s.logger, s.handleFTPFile)
        s.ftpPoller.Start()
    }
    return nil
}

func (s *LogServer) startSyslog() error {
    addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: s.config.SyslogPort}
    conn, err := net.ListenUDP(s.config.SyslogProto, addr)
    if err != nil {
        return err
    }
    s.syslogConn = conn

    go func() {
        buf := make([]byte, 65536)
        for {
            n, remote, err := conn.ReadFromUDP(buf)
            if err != nil {
                s.logger.Error("Syslog read error", "error", err)
                continue
            }
            go s.handleSyslogMessage(buf[:n], remote)
        }
    }()
    s.logger.Info("Syslog server started", "proto", s.config.SyslogProto, "port", s.config.SyslogPort)
    return nil
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

func (s *LogServer) handleSyslogMessage(data []byte, remote *net.UDPAddr) {
    raw := string(data)
    deviceID := remote.IP.String()
    level, code, message := s.parser.Parse(raw)

    parsedLog := &models.ParsedLog{
        Time:      time.Now(),
        DeviceID:  deviceID,
        LogLevel:  level,
        EventCode: code,
        Message:   message,
        Source:    "syslog",
        Raw:       raw,
    }
    if err := s.dbSaver(parsedLog); err != nil {
        s.logger.Error("Failed to save parsed log", "error", err)
    }
}

func (s *LogServer) handleHTTPLog(w http.ResponseWriter, r *http.Request) {
    var req struct {
        DeviceID string `json:"device_id"`
        Log      string `json:"log"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    raw := req.Log
    level, code, message := s.parser.Parse(raw)
    parsedLog := &models.ParsedLog{
        Time:      time.Now(),
        DeviceID:  req.DeviceID,
        LogLevel:  level,
        EventCode: code,
        Message:   message,
        Source:    "http",
        Raw:       raw,
    }
    if err := s.dbSaver(parsedLog); err != nil {
        s.logger.Error("Failed to save HTTP log", "error", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *LogServer) handleFTPFile(filename string, content []byte) {
    raw := string(content)
    deviceID := strings.TrimSuffix(filename, ".log")
    level, code, message := s.parser.Parse(raw)
    parsedLog := &models.ParsedLog{
        Time:      time.Now(),
        DeviceID:  deviceID,
        LogLevel:  level,
        EventCode: code,
        Message:   message,
        Source:    "ftp",
        Raw:       raw,
    }
    if err := s.dbSaver(parsedLog); err != nil {
        s.logger.Error("Failed to save FTP log", "error", err)
    }
}

func (s *LogServer) Stop() {
    if s.syslogConn != nil {
        s.syslogConn.Close()
    }
    if s.httpServer != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        s.httpServer.Shutdown(ctx)
    }
    if s.ftpPoller != nil {
        s.ftpPoller.Stop()
    }
}
func DefaultConfig() *Config {
    return &Config{
        SyslogEnabled: true,
        SyslogPort:    514,
        SyslogProto:   "udp",
        HTTPEnabled:   true,
        HTTPPort:      8082,
        FTPEnabled:    false,
        ParserRules:   []LogParserRule{},
    }
}
