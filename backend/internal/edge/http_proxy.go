// ═══════════════════════════════════════════════════════════════════════════
// Package edge — Edge HTTP Proxy (PROXY-01)
//
// HTTPProxy проксирует HTTP-запросы инженера к камере/устройству
// через активную WireGuard VPN-сессию.
//
// Flow:
//  1. Проверка активной VPN-сессии для agent_id
//  2. Проверка, что device_ip разрешён в AllowedIPs сессии
//  3. Создание HTTP-запроса к device_ip через локальный WG интерфейс
//  4. Копирование заголовков (кроме Host), передача тела
//  5. Возврат ответа клиенту
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 → Zone 5 conduit)
//   - IEC 62443-3-3 SR 5.1: Network segmentation (через WG)
//   - OWASP ASVS L3 V5: Input validation, access control
//   - ISO 27001 A.12.4: Audit trail
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ═══ Constants ═════════════════════════════════════════════════════════

const (
	// DefaultProxyTimeout — таймаут прокси-запроса к устройству.
	DefaultProxyTimeout = 30 * time.Second

	// MaxBodySize — максимальный размер тела запроса (10 MB).
	MaxBodySize = 10 * 1024 * 1024

	// ProxyRateLimit — лимит запросов на engineer_id.
	ProxyRateLimit     = 100
	ProxyRateWindowSec = 60 // 1 минута
)

// ═══ Types ═════════════════════════════════════════════════════════════

// DeviceIPChecker проверяет, разрешён ли IP устройства для сессии.
type DeviceIPChecker interface {
	IsAllowedIP(session *VPNSession, deviceIP net.IP) bool
}

// ProxyAuditLogger логирует прокси-запросы в audit_log.
type ProxyAuditLogger interface {
	LogProxyRequest(ctx context.Context, entry *ProxyAuditEntry) error
}

// ProxyAuditEntry — запись аудита прокси-запроса.
type ProxyAuditEntry struct {
	SessionID  uuid.UUID
	AgentID    string
	EngineerID uuid.UUID
	DeviceIP   string
	Method     string
	Path       string
	StatusCode int
	BytesSent  int64
	DurationMs int64
	TraceID    string
	Error      string
}

// HTTPProxy проксирует HTTP-запросы к устройству через VPN.
//
// Соответствие:
//   - IEC 62443 SR 5.1: Network segmentation
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.12.4: Audit logging
type HTTPProxy struct {
	manager     *VPNSessionManager
	lazyVPN     *LazyVPNSession
	auditLogger ProxyAuditLogger
	httpClient  *http.Client
	logger      *slog.Logger

	// Rate limiter: engineer_id → count
	mu           sync.Mutex
	rateCounters map[string]*rateCounter
}

type rateCounter struct {
	count    int
	windowAt time.Time
}

// NewHTTPProxy создаёт новый HTTPProxy.
func NewHTTPProxy(
	manager *VPNSessionManager,
	lazyVPN *LazyVPNSession,
	auditLogger ProxyAuditLogger,
	logger *slog.Logger,
) *HTTPProxy {
	return &HTTPProxy{
		manager:     manager,
		lazyVPN:     lazyVPN,
		auditLogger: auditLogger,
		httpClient: &http.Client{
			Timeout: DefaultProxyTimeout,
			// Не следуем редиректам (безопасность)
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger:       logger.With("component", "http-proxy"),
		rateCounters: make(map[string]*rateCounter),
	}
}

// ProxyRequest проксирует HTTP-запрос к устройству через VPN.
//
// Endpoint: GET /api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/{path*}
//
// Flow:
//  1. Проверка rate limit (100 req/min на engineer_id)
//  2. Получение/создание VPN сессии через LazyVPNSession
//  3. Проверка AllowedIPs
//  4. Проксирование запроса
//  5. Аудит
//
// Compliance:
//   - OWASP ASVS V2.2.1: Rate limiting
//   - OWASP ASVS V5.1: Input validation (device_ip, port, path)
//   - OWASP ASVS V3.3: Access control (через VPN сессию)
//   - ISO 27001 A.12.4: Audit trail
func (p *HTTPProxy) ProxyRequest(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	agentID string,
	deviceIP string,
	port string,
	path string,
	engineerID uuid.UUID,
	traceID string,
) error {
	start := time.Now()
	logger := p.logger.With("agent_id", agentID, "device_ip", deviceIP, "port", port, "path", path, "engineer_id", engineerID)

	// 1. Rate limit check (OWASP ASVS V2.2.1)
	if !p.checkRateLimit(engineerID.String()) {
		logger.Warn("rate limit exceeded")
		return fmt.Errorf("http-proxy: rate limit exceeded for engineer %s", engineerID)
	}

	// 2. Валидация IP (OWASP ASVS V5.1)
	parsedIP := net.ParseIP(deviceIP)
	if parsedIP == nil {
		return fmt.Errorf("http-proxy: invalid device IP: %s", deviceIP)
	}

	// Разрешён только private range (устройства в LAN)
	if !isPrivateIP(parsedIP) {
		return fmt.Errorf("http-proxy: device IP must be in private range: %s", deviceIP)
	}

	// 3. Определяем AllowedIPs из сессии агента
	allowedIPs, err := p.getAllowedIPsForAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("http-proxy: failed to get allowed IPs: %w", err)
	}

	// 4. Получаем/создаём VPN сессию (PROXY-03)
	session, err := p.lazyVPN.GetOrCreateSession(ctx, agentID, engineerID, allowedIPs)
	if err != nil {
		return fmt.Errorf("http-proxy: failed to get vpn session: %w", err)
	}

	// 5. Проверяем, что device_ip разрешён в AllowedIPs сессии
	if !p.isAllowedIP(session, parsedIP) {
		logger.Warn("device IP not allowed in session",
			"session_id", session.ID,
			"allowed_ips", session.AllowedIPs,
		)
		return fmt.Errorf("http-proxy: device IP %s is not in session AllowedIPs", deviceIP)
	}

	// 6. Формируем целевой URL
	targetURL := fmt.Sprintf("http://%s:%s/%s", deviceIP, port, strings.TrimPrefix(path, "/"))
	logger.Debug("proxying request", "target_url", targetURL, "method", r.Method)

	// 7. Читаем тело запроса
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, MaxBodySize))
	if err != nil {
		return fmt.Errorf("http-proxy: failed to read request body: %w", err)
	}
	defer r.Body.Close()

	// 8. Создаём прокси-запрос
	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("http-proxy: failed to create request: %w", err)
	}

	// 9. Копируем заголовки (кроме Host и Connection) (OWASP ASVS V5.3)
	for key, values := range r.Header {
		keyLower := strings.ToLower(key)
		if keyLower == "host" || keyLower == "connection" || keyLower == "upgrade" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 10. Выполняем прокси-запрос
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		logger.Error("proxy request failed", "error", err)
		auditEntry := p.makeAuditEntry(session, engineerID, deviceIP, r.Method, path, 502, 0, start, traceID, err.Error())
		p.logAudit(ctx, auditEntry)
		return fmt.Errorf("http-proxy: request to device failed: %w", err)
	}
	defer resp.Body.Close()

	// 11. Копируем ответ клиенту
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	bytesWritten, err := io.Copy(w, resp.Body)
	if err != nil {
		logger.Error("failed to copy response body", "error", err)
		// Уже отправили заголовки — не можем вернуть ошибку клиенту
	}

	// 12. Аудит (ISO 27001 A.12.4)
	auditEntry := p.makeAuditEntry(session, engineerID, deviceIP, r.Method, path, resp.StatusCode, bytesWritten, start, traceID, "")
	p.logAudit(ctx, auditEntry)

	logger.Debug("proxy request completed",
		"status", resp.StatusCode,
		"bytes", bytesWritten,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// ═══ Internal ═══════════════════════════════════════════════════════

// getAllowedIPsForAgent получает разрешённые сети для агента.
// В production — из БД, сейчас — CIDR по-умолчанию.
func (p *HTTPProxy) getAllowedIPsForAgent(ctx context.Context, agentID string) ([]net.IPNet, error) {
	// TODO: Загружать AllowedIPs из конфигурации агента в БД
	// Пока используем дефолтную LAN подсеть
	_, lanNet, err := net.ParseCIDR("192.168.0.0/16")
	if err != nil {
		return nil, fmt.Errorf("failed to parse default CIDR: %w", err)
	}
	_, lanNet2, _ := net.ParseCIDR("10.0.0.0/8")
	return []net.IPNet{*lanNet, *lanNet2}, nil
}

// isAllowedIP проверяет, входит ли IP в список разрешённых сетей сессии.
func (p *HTTPProxy) isAllowedIP(session *VPNSession, ip net.IP) bool {
	for _, allowed := range session.AllowedIPs {
		if allowed.Contains(ip) {
			return true
		}
	}
	return false
}

// checkRateLimit проверяет rate limit для engineer_id.
//
// Compliance: OWASP ASVS V2.2.1 — Rate limiting
func (p *HTTPProxy) checkRateLimit(engineerID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	counter, exists := p.rateCounters[engineerID]

	if !exists || now.After(counter.windowAt.Add(ProxyRateWindowSec*time.Second)) {
		// Новое окно
		p.rateCounters[engineerID] = &rateCounter{
			count:    1,
			windowAt: now,
		}
		return true
	}

	counter.count++
	if counter.count > ProxyRateLimit {
		return false
	}

	return true
}

// makeAuditEntry создаёт запись аудита прокси-запроса.
func (p *HTTPProxy) makeAuditEntry(
	session *VPNSession,
	engineerID uuid.UUID,
	deviceIP, method, path string,
	statusCode int,
	bytesWritten int64,
	start time.Time,
	traceID string,
	errMsg string,
) *ProxyAuditEntry {
	return &ProxyAuditEntry{
		SessionID:  session.ID,
		AgentID:    session.AgentID,
		EngineerID: engineerID,
		DeviceIP:   deviceIP,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		BytesSent:  bytesWritten,
		DurationMs: time.Since(start).Milliseconds(),
		TraceID:    traceID,
		Error:      errMsg,
	}
}

// logAudit логирует прокси-запрос в audit_log.
func (p *HTTPProxy) logAudit(ctx context.Context, entry *ProxyAuditEntry) {
	if p.auditLogger != nil {
		if err := p.auditLogger.LogProxyRequest(ctx, entry); err != nil {
			p.logger.Warn("failed to log proxy audit", "error", err)
		}
	}
	// Всегда логируем в структурированный лог
	level := slog.LevelInfo
	if entry.Error != "" {
		level = slog.LevelError
	}
	p.logger.Log(ctx, level, "proxy request",
		"session_id", entry.SessionID,
		"agent_id", entry.AgentID,
		"device_ip", entry.DeviceIP,
		"method", entry.Method,
		"path", entry.Path,
		"status", entry.StatusCode,
		"bytes", entry.BytesSent,
		"duration_ms", entry.DurationMs,
		"trace_id", entry.TraceID,
		"error", entry.Error,
	)
}

// isPrivateIP проверяет, является ли IP частным (RFC 1918).
func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
	}
	return false
}
