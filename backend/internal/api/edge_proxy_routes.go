// ═══════════════════════════════════════════════════════════════════════════
// Package api — Edge Proxy Routes (PROXY-01, PROXY-02)
//
// Маршруты для Zero-Touch Proxy:
//   - HTTP прокси к камере через WireGuard VPN
//   - WebSocket SSH терминал
//
// Все маршруты защищены JWT аутентификацией (монтируются внутри protected group).
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - OWASP ASVS V3.3: Privilege escalation prevention
//   - ISO 27001 A.12.4: Audit trail
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
// ═══════════════════════════════════════════════════════════════════════════

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/edge"
	"gb-telemetry-collector/internal/trace"
)

// upgrader — WebSocket upgrader с дефолтными настройками.
var proxyWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Проверка Origin делается в middleware
	CheckOrigin: func(r *http.Request) bool { return true },
}

// mountEdgeProxyRoutes регистрирует маршруты Edge Proxy.
//
// Все эндпоинты требуют JWT (монтируются внутри protected group).
func (s *Server) mountEdgeProxyRoutes(r chi.Router) {
	// PROXY-01: HTTP прокси к устройству через VPN
	// GET /api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/{path*}
	// Пример: GET /api/v1/edge/proxy/agent-01/192.168.1.100:80/index.html
	r.Get("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/*", s.handleEdgeHTTPProxy)
	r.Get("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}", s.handleEdgeHTTPProxy)
	r.Post("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/*", s.handleEdgeHTTPProxy)
	r.Post("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}", s.handleEdgeHTTPProxy)
	r.Put("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/*", s.handleEdgeHTTPProxy)
	r.Put("/api/v1/edge/proxy/{agent_id}/{device_ip}:{port}", s.handleEdgeHTTPProxy)

	// PROXY-02: WebSocket SSH терминал
	// WSS /api/v1/edge/ssh/{agent_id}/{device_ip}/{port}
	// Пример: WSS /api/v1/edge/ssh/agent-01/192.168.1.100/22
	r.Get("/api/v1/edge/ssh/{agent_id}/{device_ip}/{port}", s.handleEdgeSSHProxy)
}

// ═══ PROXY-01: HTTP Proxy Handler ═════════════════════════════════════

// handleEdgeHTTPProxy проксирует HTTP-запрос к устройству через VPN.
//
// Endpoint: GET/POST/PUT /api/v1/edge/proxy/{agent_id}/{device_ip}:{port}/{path*}
//
// Compliance:
//   - OWASP ASVS V5.1: Input validation
//   - OWASP ASVS V3.3: Access control
//   - ISO 27001 A.12.4: Audit trail
func (s *Server) handleEdgeHTTPProxy(w http.ResponseWriter, r *http.Request) {
	if s.httpProxy == nil {
		RespondError(w, r, NewBadRequestError("Edge HTTP proxy is not configured"))
		return
	}

	agentID := chi.URLParam(r, "agent_id")
	deviceIPPort := chi.URLParam(r, "device_ip:port")
	path := chi.URLParam(r, "*")

	// Парсим device_ip:port
	parts := strings.SplitN(deviceIPPort, ":", 2)
	if len(parts) != 2 {
		RespondError(w, r, NewValidationError("invalid device_ip:port format, expected ip:port"))
		return
	}
	deviceIP := parts[0]
	port := parts[1]

	// Извлекаем engineer_id из JWT контекста
	engineerID, err := s.getEngineerIDFromContext(r)
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	// TraceID для audit
	traceID := trace.FromContext(r.Context())

	// Проксируем запрос
	err = s.httpProxy.ProxyRequest(r.Context(), w, r, agentID, deviceIP, port, path, engineerID, traceID)
	if err != nil {
		RespondError(w, r, NewInternalError("proxy request failed", err))
		return
	}
}

// ═══ PROXY-02: SSH Proxy Handler ═════════════════════════════════════

// handleEdgeSSHProxy обрабатывает WebSocket соединение для SSH терминала.
//
// Endpoint: WSS /api/v1/edge/ssh/{agent_id}/{device_ip}/{port}
//
// Flow:
//  1. WebSocket upgrade
//  2. Получение credentials от клиента
//  3. SSH подключение через VPN
//  4. Двусторонний прокси
//
// Compliance:
//   - OWASP ASVS V2.1: Authentication (JWT + SSH credentials)
//   - OWASP ASVS V5.1: Input validation
//   - IEC 62443-3-3 SR 5.1: Network segmentation
func (s *Server) handleEdgeSSHProxy(w http.ResponseWriter, r *http.Request) {
	if s.sshProxy == nil {
		RespondError(w, r, NewBadRequestError("Edge SSH proxy is not configured"))
		return
	}

	agentID := chi.URLParam(r, "agent_id")
	deviceIP := chi.URLParam(r, "device_ip")
	portStr := chi.URLParam(r, "port")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		RespondError(w, r, NewValidationError("invalid port"))
		return
	}

	// Извлекаем engineer_id из JWT контекста
	engineerID, err := s.getEngineerIDFromContext(r)
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	// Upgrade to WebSocket
	conn, err := proxyWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to upgrade to websocket", err))
		return
	}

	// Ожидаем первое сообщение с SSH credentials
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseProtocolError, "failed to read credentials"))
		conn.Close()
		return
	}

	var authReq struct {
		Type     string `json:"type"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(msgBytes, &authReq); err != nil || authReq.Type != "auth" {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseProtocolError, "invalid auth message"))
		conn.Close()
		return
	}

	if authReq.Username == "" {
		authReq.Username = "root"
	}

	// Создаём каналы для двусторонней связи
	send := func(msg edge.WSMessage) error {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal ws message: %w", err)
		}
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	recv := make(chan edge.WSMessage, 100)

	// Горутина для чтения из WebSocket
	go func() {
		defer close(recv)
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					s.logger.Warn("websocket read error", "error", err)
				}
				return
			}

			var msg edge.WSMessage
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			select {
			case recv <- msg:
			default:
				// Переполнение буфера — игнорируем
			}
		}
	}()

	// Отправляем подтверждение
	conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"connected"}`))

	// Запускаем SSH сессию
	err = s.sshProxy.HandleSSHSession(
		r.Context(),
		agentID, deviceIP, port,
		engineerID,
		authReq.Username, authReq.Password,
		send, recv,
	)
	if err != nil {
		s.logger.Warn("ssh session ended with error", "error", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, err.Error()))
	}
	conn.Close()
}

// ═══ Helpers ═══════════════════════════════════════════════════════════

// getEngineerIDFromContext извлекает engineer_id из JWT контекста.
func (s *Server) getEngineerIDFromContext(r *http.Request) (uuid.UUID, error) {
	// Извлекаем user_id из контекста (устанавливается AuthMiddleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		return uuid.Nil, fmt.Errorf("user_id not in context")
	}
	return uuid.Parse(userID)
}

// ═══ Audit Logger Implementation ═════════════════════════════════════

// LogProxyRequest логирует прокси-запрос в audit_log (ISO 27001 A.12.4).
//
// Compliance:
//   - ISO 27001 A.12.4.1: Event logging
//   - ISO 27001 A.12.4.3: Log protection
func (s *Server) LogProxyRequest(ctx context.Context, entry *edge.ProxyAuditEntry) error {
	if s.auditChainStore == nil {
		return nil
	}

	oldVal := map[string]interface{}{
		"session_id": entry.SessionID.String(),
		"agent_id":   entry.AgentID,
		"device_ip":  entry.DeviceIP,
	}

	newVal := map[string]interface{}{
		"status_code": entry.StatusCode,
		"bytes_sent":  entry.BytesSent,
		"duration_ms": entry.DurationMs,
		"method":      entry.Method,
		"path":        entry.Path,
		"error":       entry.Error,
	}

	action := "edge_proxy_request"
	if entry.Error != "" {
		action = "edge_proxy_request_error"
	}

	return s.auditChainStore.InsertWithChain(ctx, &audit.AuditEntry{
		UserID:     entry.EngineerID.String(),
		Action:     action,
		EntityType: "edge_session",
		EntityID:   entry.SessionID.String(),
		OldValue:   oldVal,
		NewValue:   newVal,
		IPAddress:  "",
		UserAgent:  "",
		TraceID:    entry.TraceID,
	})
}

// ═══ Server Fields (добавляются в Server struct) ═══════════════════

// httpProxy — HTTP прокси для устройств через VPN.
// Устанавливается через SetHTTPProxy.
var _ = (*Server).SetHTTPProxy

// SetHTTPProxy устанавливает HTTP прокси.
func (s *Server) SetHTTPProxy(proxy *edge.HTTPProxy) {
	s.httpProxy = proxy
}

// sshProxy — SSH прокси для устройств через VPN.
// Устанавливается через SetSSHProxy.
var _ = (*Server).SetSSHProxy

// SetSSHProxy устанавливает SSH прокси.
func (s *Server) SetSSHProxy(proxy *edge.SSHProxy) {
	s.sshProxy = proxy
}
