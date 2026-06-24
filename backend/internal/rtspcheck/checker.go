// Package rtspcheck — RTSP Health Checker (CCTV-2.2.2).
//
// Проверяет RTSP endpoint через telemetry (без видеопотоков):
//   - TCP connect to RTSP port (554)
//   - RTSP OPTIONS request (проверка что сервер отвечает)
//   - RTSP DESCRIBE (получение информации о stream)
//   - Response time measurement
//   - Stream health detection (frozen stream через мониторинг)
//
// Compliance:
//   - CCTV Core IP
//   - Только telemetry — без загрузки видео
package rtspcheck

import (
	"bufio"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// RTSPStatus — статус RTSP endpoint.
type RTSPStatus string

const (
	StatusOnline   RTSPStatus = "ONLINE"
	StatusOffline  RTSPStatus = "OFFLINE"
	StatusDegraded RTSPStatus = "DEGRADED"
	StatusTimeout  RTSPStatus = "TIMEOUT"
)

// StreamHealth — здоровье видеопотока.
type StreamHealth string

const (
	StreamHealthy   StreamHealth = "healthy"
	StreamFrozen    StreamHealth = "frozen"    // stream не меняется
	StreamNoSignal  StreamHealth = "no_signal" // black frame
	StreamUnknown   StreamHealth = "unknown"
)

// CheckResult — результат проверки RTSP endpoint.
type CheckResult struct {
	URL          string        `json:"url"`
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Status       RTSPStatus    `json:"status"`
	StatusCode   int           `json:"status_code"` // RTSP status code
	ResponseTime time.Duration `json:"response_time_ms"`
	PublicIP     string        `json:"public_ip,omitempty"` // публичный IP камеры
	Server       string        `json:"server,omitempty"`    // Server header
	Streams      int           `json:"streams"`             // количество stream'ов
	Transport    string        `json:"transport,omitempty"` // supported transport
	StreamHealth StreamHealth  `json:"stream_health"`
	Error        string        `json:"error,omitempty"`
	CheckedAt    time.Time     `json:"checked_at"`
}

// Checker — RTSP health checker.
type Checker struct {
	mu         sync.Mutex
	logger     *slog.Logger
	timeout    time.Duration
	userAgent  string
	lastSeqMap map[string]string // url → last response body hash (for frozen detection)
}

// NewChecker создаёт RTSP Checker.
func NewChecker(logger *slog.Logger) *Checker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{
		logger:     logger.With("component", "rtsp-check"),
		timeout:    5 * time.Second,
		userAgent:  "CCTV-Health-Monitor/1.0",
		lastSeqMap: make(map[string]string),
	}
}

// SetTimeout устанавливает таймаут подключения.
func (c *Checker) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Check выполняет полную проверку RTSP endpoint.
//
// Алгоритм:
//  1. TCP connect к host:port
//  2. RTSP OPTIONS → проверка что сервер отвечает
//  3. RTSP DESCRIBE → получение информации о stream
//  4. Проверка frozen stream (сравнение с предыдущим ответом)
//  5. Измерение времени отклика
func (c *Checker) Check(url string) *CheckResult {
	result := &CheckResult{
		URL:       url,
		CheckedAt: time.Now().UTC(),
		Status:    StatusOffline,
	}

	// Парсинг URL
	host, port, err := parseRTSPURL(url)
	if err != nil {
		result.Error = fmt.Sprintf("invalid RTSP URL: %v", err)
		return result
	}
	result.Host = host
	result.Port = port

	// 1. TCP connect
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		result.Error = fmt.Sprintf("TCP connect failed: %v", err)
		result.Status = StatusOffline
		result.ResponseTime = time.Since(start)
		c.logger.Warn("rtsp tcp connect failed", "url", url, "error", err)
		return result
	}
	defer conn.Close()

	// Устанавливаем таймаут на чтение/запись
	conn.SetDeadline(time.Now().Add(c.timeout))

	// 2. RTSP OPTIONS
	optionsResp, err := c.sendRequest(conn, "OPTIONS", url, "")
	if err != nil {
		result.Error = fmt.Sprintf("OPTIONS failed: %v", err)
		result.Status = StatusDegraded
		result.ResponseTime = time.Since(start)
		return result
	}

	result.StatusCode = optionsResp.statusCode
	result.Server = optionsResp.headers.Get("Server")
	result.PublicIP = conn.RemoteAddr().String()
	result.Transport = optionsResp.headers.Get("Public")

	// 3. RTSP DESCRIBE
	describeResp, err := c.sendRequest(conn, "DESCRIBE", url, "Accept: application/sdp\r\n")
	if err != nil {
		result.Status = StatusDegraded
		result.ResponseTime = time.Since(start)
		return result
	}

	// Парсим SDP для подсчёта stream'ов
	result.Streams = countStreams(describeResp.body)

	// 4. Frozen stream detection
	result.StreamHealth = c.detectStreamHealth(url, describeResp.body)

	// Response time
	result.ResponseTime = time.Since(start)
	result.Status = StatusOnline

	return result
}

// CheckMultiple выполняет проверку нескольких RTSP endpoint'ов.
func (c *Checker) CheckMultiple(urls []string) []*CheckResult {
	results := make([]*CheckResult, 0, len(urls))
	for _, url := range urls {
		result := c.Check(url)
		results = append(results, result)
	}
	return results
}

// ═══════════════════════════════════════════════════════════════════════
// RTSP protocol helpers
// ═══════════════════════════════════════════════════════════════════════

type rtspResponse struct {
	statusCode int
	statusText string
	headers    textproto.MIMEHeader
	body       string
}

// sendRequest отправляет RTSP запрос и парсит ответ.
func (c *Checker) sendRequest(conn net.Conn, method, url, extraHeaders string) (*rtspResponse, error) {
	// Сборка запроса
	req := fmt.Sprintf("%s %s RTSP/1.0\r\n"+
		"CSeq: 1\r\n"+
		"User-Agent: %s\r\n"+
		"%s"+
		"\r\n", method, url, c.userAgent, extraHeaders)

	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, fmt.Errorf("write %s request: %w", method, err)
	}

	// Чтение ответа
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	// Status line: RTSP/1.0 200 OK
	statusLine, err := tp.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("read %s status: %w", method, err)
	}

	resp := &rtspResponse{}
	if parts := strings.SplitN(statusLine, " ", 3); len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &resp.statusCode)
		if len(parts) >= 3 {
			resp.statusText = parts[2]
		}
	}

	// Headers
	resp.headers, err = tp.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("read %s headers: %w", method, err)
	}

	// Content-Length
	contentLength := 0
	if cl := resp.headers.Get("Content-Length"); cl != "" {
		fmt.Sscanf(cl, "%d", &contentLength)
	}

	// Body
	if contentLength > 0 && contentLength < 65536 {
		body := make([]byte, contentLength)
		if _, err := reader.Read(body); err == nil {
			resp.body = string(body)
		}
	}

	return resp, nil
}

// countStreams подсчитывает количество медиа-потоков в SDP.
func countStreams(sdp string) int {
	count := 0
	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "m=") {
			count++
		}
	}
	return count
}

// detectStreamHealth определяет здоровье потока.
func (c *Checker) detectStreamHealth(url string, body string) StreamHealth {
	if body == "" {
		return StreamUnknown
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Хеш тела ответа для сравнения (используем длину + первые 1000 байт)
	hashKey := body
	if len(hashKey) > 1000 {
		hashKey = hashKey[:1000]
	}
	hash := fmt.Sprintf("%x", hashKey)

	if prevHash, ok := c.lastSeqMap[url]; ok {
		if hash == prevHash {
			c.logger.Warn("rtsp frozen stream detected", "url", url)
			return StreamFrozen
		}
	}

	c.lastSeqMap[url] = hash
	return StreamHealthy
}

// ═══════════════════════════════════════════════════════════════════════
// URL parsing
// ═══════════════════════════════════════════════════════════════════════

func parseRTSPURL(rawURL string) (host string, port int, err error) {
	// rtsp://user:pass@host:port/path
	// rtsp://host:port/path
	s := rawURL

	// Remove scheme
	if strings.HasPrefix(s, "rtsp://") {
		s = s[7:]
	} else if strings.HasPrefix(s, "rtsps://") {
		s = s[8:]
	}

	// Remove credentials
	if atIdx := strings.LastIndex(s, "@"); atIdx >= 0 {
		s = s[atIdx+1:]
	}

	// Split host:port/path
	slashIdx := strings.Index(s, "/")
	hostPort := s
	if slashIdx >= 0 {
		hostPort = s[:slashIdx]
	}

	// Split host:port
	if colonIdx := strings.LastIndex(hostPort, ":"); colonIdx >= 0 {
		host = hostPort[:colonIdx]
		fmt.Sscanf(hostPort[colonIdx+1:], "%d", &port)
	} else {
		host = hostPort
		port = 554 // default RTSP port
	}

	if host == "" {
		return "", 0, fmt.Errorf("empty host in RTSP URL")
	}

	return host, port, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// Summary возвращает краткую сводку проверки.
func (r *CheckResult) Summary() string {
	ms := r.ResponseTime.Milliseconds()
	parts := []string{
		fmt.Sprintf("RTSP %s (%dms)", r.Status, ms),
	}

	if r.StatusCode > 0 {
		parts = append(parts, fmt.Sprintf("code=%d", r.StatusCode))
	}
	if r.Streams > 0 {
		parts = append(parts, fmt.Sprintf("streams=%d", r.Streams))
	}
	if r.Server != "" {
		parts = append(parts, fmt.Sprintf("server=%s", r.Server))
	}
	if r.StreamHealth != "" && r.StreamHealth != StreamHealthy {
		parts = append(parts, fmt.Sprintf("health=%s", r.StreamHealth))
	}
	if r.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", r.Error))
	}

	return strings.Join(parts, " | ")
}

// HealthScore возвращает числовую оценку здоровья (0-100).
func (r *CheckResult) HealthScore() float64 {
	score := 100.0

	switch r.Status {
	case StatusOffline:
		return 0
	case StatusTimeout:
		score -= 50
	case StatusDegraded:
		score -= 30
	}

	// Response time penalty
	ms := r.ResponseTime.Milliseconds()
	if ms > 1000 {
		score -= 20
	} else if ms > 500 {
		score -= 10
	} else if ms > 200 {
		score -= 5
	}

	// Stream health penalty
	if r.StreamHealth == StreamFrozen {
		score -= 40
	} else if r.StreamHealth == StreamNoSignal {
		score -= 30
	}

	return math.Max(0, score)
}
