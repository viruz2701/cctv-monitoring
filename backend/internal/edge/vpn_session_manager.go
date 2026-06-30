// ═══════════════════════════════════════════════════════════════════════════
// Package edge — VPN Session Manager (EDGE-08)
//
// VPNSessionManager управляет жизненным циклом WireGuard VPN сессий
// для удалённого доступа инженеров к edge-агентам.
//
// Flow создания сессии:
//   1. Validate RBAC (admin/support only)
//   2. Generate WireGuard keypair
//   3. Add peer to WG server (AllowedIPs = LAN of agent)
//   4. Set timeout (1-2 hours, configurable)
//   5. Send MQTT command to agent: start_vpn_session
//   6. Save session to DB
//   7. Start goroutine for auto-close on timeout
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation, session management
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.12.4: Audit trail
//   - OWASP ASVS V2.1: Session management
//   - OWASP ASVS V3.3: Privilege escalation prevention
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ═══ Типы данных ═══════════════════════════════════════════════════════

// VPNSession представляет VPN сессию для удалённого доступа.
//
// SELFSERV-01: PrivateKey хранится только в памяти (не в БД),
// передаётся клиенту при скачивании конфига.
type VPNSession struct {
	ID         uuid.UUID   `json:"id"`
	AgentID    string      `json:"agent_id"`
	EngineerID uuid.UUID   `json:"engineer_id"`
	StartedAt  time.Time   `json:"started_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
	AllowedIPs []net.IPNet `json:"allowed_ips"`
	PublicKey  string      `json:"public_key"`
	// PrivateKey — приватный ключ клиента (только в памяти, не в БД).
	// SELFSERV-01: Передаётся инженеру при скачивании wg-quick конфига.
	PrivateKey       string     `json:"-"`
	Status           string     `json:"status"` // active, expired, revoked
	BytesTransferred int64      `json:"bytes_transferred"`
	CreatedAt        time.Time  `json:"created_at"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
}

// CreateSessionRequest — запрос на создание VPN сессии.
type CreateSessionRequest struct {
	AgentID    string      `json:"agent_id"`
	EngineerID uuid.UUID   `json:"engineer_id"`
	AllowedIPs []net.IPNet `json:"allowed_ips"`
	Duration   Duration    `json:"duration"` // время жизни сессии
}

// Duration — duration для JSON (человекочитаемый формат).
type Duration struct {
	time.Duration
}

// UnmarshalJSON реализует json.Unmarshaler для Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

// SessionFilter — фильтр для списка сессий.
type SessionFilter struct {
	AgentID    string    `json:"agent_id,omitempty"`
	EngineerID uuid.UUID `json:"engineer_id,omitempty"`
	Status     string    `json:"status,omitempty"` // active, expired, revoked, all
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// MQTTPublisher — интерфейс для отправки MQTT команд агентам.
type MQTTPublisher interface {
	PublishCommand(ctx context.Context, agentID string, command string, payload interface{}) error
}

// ═══ VPNSessionManager ═════════════════════════════════════════════════

// VPNSessionManager управляет WireGuard VPN сессиями.
type VPNSessionManager struct {
	pool           *pgxpool.Pool
	wgServer       *WireGuardServer
	mqttPub        MQTTPublisher
	activeSessions map[string]*VPNSession
	mu             sync.RWMutex
	logger         *slog.Logger
	cleanupStop    chan struct{}
	hmacKey        []byte   // SELFSERV-01: ключ для HMAC подписи конфигов
	dns            []string // SELFSERV-01: DNS сервера для клиента
	serverEndpoint string   // SELFSERV-01: публичный endpoint для клиентов
}

// NewVPNSessionManager создаёт новый менеджер VPN сессий.
func NewVPNSessionManager(
	pool *pgxpool.Pool,
	wgServer *WireGuardServer,
	mqttPub MQTTPublisher,
	logger *slog.Logger,
) *VPNSessionManager {
	return &VPNSessionManager{
		pool:           pool,
		wgServer:       wgServer,
		mqttPub:        mqttPub,
		activeSessions: make(map[string]*VPNSession),
		logger:         logger.With("component", "vpn-session-manager"),
		cleanupStop:    make(chan struct{}),
	}
}

// SetHMACKey устанавливает ключ для HMAC подписи WireGuard конфигов.
// SELFSERV-01: Опционально, для обеспечения целостности конфигов.
func (m *VPNSessionManager) SetHMACKey(key []byte) {
	m.hmacKey = key
}

// SetDNS устанавливает DNS сервера для WireGuard клиентов.
// SELFSERV-01: Опционально, для указания DNS в конфиге клиента.
func (m *VPNSessionManager) SetDNS(dns []string) {
	m.dns = dns
}

// SetServerEndpoint устанавливает публичный endpoint WireGuard сервера.
// SELFSERV-01: Используется в конфиге клиента (host:port).
func (m *VPNSessionManager) SetServerEndpoint(endpoint string) {
	m.serverEndpoint = endpoint
}

// GetServerPublicKey возвращает публичный ключ WireGuard сервера.
func (m *VPNSessionManager) GetServerPublicKey() string {
	if m.wgServer == nil {
		return ""
	}
	return m.wgServer.GetPublicKey()
}

// GetServerEndpoint возвращает endpoint для подключения к WG серверу.
func (m *VPNSessionManager) GetServerEndpoint() string {
	if m.serverEndpoint != "" {
		return m.serverEndpoint
	}
	if m.wgServer == nil {
		return ""
	}
	return fmt.Sprintf("vpn.example.com:%d", m.wgServer.GetListenPort())
}

// GetDNS возвращает DNS сервера для клиента.
func (m *VPNSessionManager) GetDNS() []string {
	return m.dns
}

// GetHMACKey возвращает ключ для HMAC подписи конфигов.
func (m *VPNSessionManager) GetHMACKey() []byte {
	return m.hmacKey
}

// CreateSession создаёт новую VPN сессию.
//
// Flow:
//  1. Validate RBAC (admin/support only — вызывается до этого метода)
//  2. Generate WireGuard keypair
//  3. Add peer to WG server (AllowedIPs = LAN of agent)
//  4. Set timeout (1-2 hours, configurable)
//  5. Send MQTT command to agent: start_vpn_session
//  6. Save session to DB
//  7. Start goroutine for auto-close on timeout
//
// Compliance:
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - Приказ ОАЦ №66 п. 7.18.2: Контроль удалённого доступа
//   - ISO 27001 A.12.4: Audit trail
//   - OWASP ASVS V2.1.1: Session management
func (m *VPNSessionManager) CreateSession(ctx context.Context, req CreateSessionRequest) (*VPNSession, error) {
	logger := m.logger.With("agent_id", req.AgentID, "engineer_id", req.EngineerID)
	logger.Info("creating vpn session")

	// 1. Валидация длительности
	if req.Duration.Duration <= 0 {
		req.Duration.Duration = 1 * time.Hour // дефолт 1 час
	}
	if req.Duration.Duration > 2*time.Hour {
		return nil, fmt.Errorf("vpn: max session duration is 2 hours")
	}

	// 2. Генерация ключей WireGuard
	// SELFSERV-01: PrivateKey сохраняется в памяти для self-service скачивания конфига.
	privKey, pubKey, err := m.wgServer.GenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("vpn: failed to generate keypair: %w", err)
	}

	// 3. Добавление пира в WG сервер
	if err := m.wgServer.AddPeer(ctx, pubKey, req.AllowedIPs); err != nil {
		return nil, fmt.Errorf("vpn: failed to add wireguard peer: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(req.Duration.Duration)

	session := &VPNSession{
		ID:         uuid.New(),
		AgentID:    req.AgentID,
		EngineerID: req.EngineerID,
		StartedAt:  now,
		ExpiresAt:  expiresAt,
		AllowedIPs: req.AllowedIPs,
		PublicKey:  pubKey,
		PrivateKey: privKey, // SELFSERV-01: для self-service скачивания конфига
		Status:     "active",
		CreatedAt:  now,
	}

	// 4. Сохранение в БД
	if err := m.saveSession(ctx, session); err != nil {
		// Rollback: удаляем пира из WG
		_ = m.wgServer.RemovePeer(ctx, pubKey)
		return nil, fmt.Errorf("vpn: failed to save session: %w", err)
	}

	// 5. Отправка MQTT команды агенту
	if err := m.mqttPub.PublishCommand(ctx, req.AgentID, "start_vpn_session", map[string]interface{}{
		"session_id":   session.ID.String(),
		"duration_sec": int(req.Duration.Duration.Seconds()),
	}); err != nil {
		m.logger.Warn("failed to send MQTT command to agent",
			"agent_id", req.AgentID,
			"error", err,
		)
		// Не фейлим создание из-за MQTT — агент получит команду при реконнекте
	}

	// 6. Регистрация в activeSessions
	m.mu.Lock()
	m.activeSessions[session.ID.String()] = session
	m.mu.Unlock()

	// 7. Auto-close goroutine
	go m.autoCloseSession(session)

	logger.Info("vpn session created",
		"session_id", session.ID,
		"expires_at", expiresAt,
	)

	return session, nil
}

// RevokeSession закрывает VPN сессию досрочно.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.2: Отзыв доступа
//   - ISO 27001 A.12.4: Audit trail
func (m *VPNSessionManager) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	logger := m.logger.With("session_id", sessionID)
	logger.Info("revoking vpn session")

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("vpn: session not found: %w", err)
	}

	if session.Status != "active" {
		return fmt.Errorf("vpn: session is not active, current status: %s", session.Status)
	}

	// Удаляем пира из WG
	if err := m.wgServer.RemovePeer(ctx, session.PublicKey); err != nil {
		return fmt.Errorf("vpn: failed to remove wireguard peer: %w", err)
	}

	// Отправляем MQTT команду агенту
	if err := m.mqttPub.PublishCommand(ctx, session.AgentID, "stop_vpn_session", map[string]interface{}{
		"session_id": sessionID.String(),
	}); err != nil {
		m.logger.Warn("failed to send stop command to agent",
			"agent_id", session.AgentID,
			"error", err,
		)
	}

	// Обновляем статус в БД
	now := time.Now()
	if err := m.updateSessionStatus(ctx, sessionID, "revoked", &now); err != nil {
		return fmt.Errorf("vpn: failed to update session status: %w", err)
	}

	// Удаляем из activeSessions
	m.mu.Lock()
	delete(m.activeSessions, sessionID.String())
	m.mu.Unlock()

	logger.Info("vpn session revoked")
	return nil
}

// GetSessions возвращает список VPN сессий по фильтру.
func (m *VPNSessionManager) GetSessions(ctx context.Context, filter SessionFilter) ([]VPNSession, error) {
	query := `SELECT id, agent_id, engineer_id, started_at, expires_at,
		allowed_ips, public_key, status, bytes_transferred, created_at, closed_at
		FROM vpn_sessions WHERE 1=1`
	args := make([]interface{}, 0)
	argIdx := 1

	if filter.AgentID != "" {
		query += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, filter.AgentID)
		argIdx++
	}
	if filter.EngineerID != uuid.Nil {
		query += fmt.Sprintf(" AND engineer_id = $%d", argIdx)
		args = append(args, filter.EngineerID)
		argIdx++
	}
	if filter.Status != "" && filter.Status != "all" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := m.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("vpn: failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []VPNSession
	for rows.Next() {
		var s VPNSession
		if err := rows.Scan(
			&s.ID, &s.AgentID, &s.EngineerID,
			&s.StartedAt, &s.ExpiresAt,
			&s.AllowedIPs, &s.PublicKey,
			&s.Status, &s.BytesTransferred,
			&s.CreatedAt, &s.ClosedAt,
		); err != nil {
			return nil, fmt.Errorf("vpn: failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = make([]VPNSession, 0)
	}

	return sessions, nil
}

// GetSession возвращает детали VPN сессии по ID.
func (m *VPNSessionManager) GetSession(ctx context.Context, sessionID uuid.UUID) (*VPNSession, error) {
	// Сначала проверяем в памяти
	m.mu.RLock()
	if s, ok := m.activeSessions[sessionID.String()]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()

	// Иначе из БД
	query := `SELECT id, agent_id, engineer_id, started_at, expires_at,
		allowed_ips, public_key, status, bytes_transferred, created_at, closed_at
		FROM vpn_sessions WHERE id = $1`

	var s VPNSession
	err := m.pool.QueryRow(ctx, query, sessionID).Scan(
		&s.ID, &s.AgentID, &s.EngineerID,
		&s.StartedAt, &s.ExpiresAt,
		&s.AllowedIPs, &s.PublicKey,
		&s.Status, &s.BytesTransferred,
		&s.CreatedAt, &s.ClosedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("vpn: session not found")
		}
		return nil, fmt.Errorf("vpn: failed to get session: %w", err)
	}

	return &s, nil
}

// GetSessionConfig возвращает WireGuard конфигурацию для клиента.
func (m *VPNSessionManager) GetSessionConfig(ctx context.Context, sessionID uuid.UUID) (map[string]interface{}, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != "active" {
		return nil, fmt.Errorf("vpn: session is not active")
	}

	// Формируем WG config для клиента
	config := map[string]interface{}{
		"interface_name":       m.wgServer.DeviceName(),
		"server_public_key":    m.wgServer.GetPublicKey(),
		"server_endpoint":      fmt.Sprintf("vpn.example.com:%d", m.wgServer.GetListenPort()),
		"address":              "", // IP для WG интерфейса клиента (из allowed_ips)
		"allowed_ips":          session.AllowedIPs,
		"dns":                  []string{},
		"persistent_keepalive": 25,
	}

	return config, nil
}

// StartAutoCleanup запускает периодическую очистку истёкших сессий.
//
// Compliance: ISO 27001 A.12.4 — своевременное закрытие сессий
func (m *VPNSessionManager) StartAutoCleanup(ctx context.Context, interval time.Duration) {
	m.logger.Info("starting auto cleanup", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.expireSessions(ctx)
		case <-m.cleanupStop:
			m.logger.Info("auto cleanup stopped")
			return
		}
	}
}

// StopAutoCleanup останавливает периодическую очистку.
func (m *VPNSessionManager) StopAutoCleanup() {
	close(m.cleanupStop)
}

// ═══ Внутренние методы ═══════════════════════════════════════════════

// saveSession сохраняет сессию в БД.
func (m *VPNSessionManager) saveSession(ctx context.Context, session *VPNSession) error {
	query := `INSERT INTO vpn_sessions
		(id, agent_id, engineer_id, started_at, expires_at, allowed_ips, public_key, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := m.pool.Exec(ctx, query,
		session.ID, session.AgentID, session.EngineerID,
		session.StartedAt, session.ExpiresAt,
		session.AllowedIPs, session.PublicKey,
		session.Status, session.CreatedAt,
	)
	return err
}

// updateSessionStatus обновляет статус сессии в БД.
func (m *VPNSessionManager) updateSessionStatus(ctx context.Context, sessionID uuid.UUID, status string, closedAt *time.Time) error {
	query := `UPDATE vpn_sessions SET status = $1, closed_at = $2 WHERE id = $3`
	_, err := m.pool.Exec(ctx, query, status, closedAt, sessionID)
	return err
}

// expireSessions находит и закрывает истёкшие сессии.
func (m *VPNSessionManager) expireSessions(ctx context.Context) {
	m.logger.Debug("checking expired sessions")

	sessions, err := m.GetSessions(ctx, SessionFilter{Status: "active"})
	if err != nil {
		m.logger.Error("failed to get active sessions", "error", err)
		return
	}

	now := time.Now()
	for _, s := range sessions {
		if now.After(s.ExpiresAt) {
			m.logger.Info("expiring session",
				"session_id", s.ID,
				"expires_at", s.ExpiresAt,
			)

			// Удаляем пира из WG
			if err := m.wgServer.RemovePeer(ctx, s.PublicKey); err != nil {
				m.logger.Error("failed to remove peer for expired session",
					"session_id", s.ID,
					"error", err,
				)
			}

			// Обновляем статус
			if err := m.updateSessionStatus(ctx, s.ID, "expired", &now); err != nil {
				m.logger.Error("failed to update expired session",
					"session_id", s.ID,
					"error", err,
				)
			}

			// Удаляем из activeSessions
			m.mu.Lock()
			delete(m.activeSessions, s.ID.String())
			m.mu.Unlock()
		}
	}
}

// autoCloseSession автоматически закрывает сессию по таймауту.
func (m *VPNSessionManager) autoCloseSession(session *VPNSession) {
	timer := time.NewTimer(time.Until(session.ExpiresAt))
	defer timer.Stop()

	select {
	case <-timer.C:
		ctx := context.Background()
		m.logger.Info("auto-closing session by timeout",
			"session_id", session.ID,
		)

		// Удаляем пира из WG
		if err := m.wgServer.RemovePeer(ctx, session.PublicKey); err != nil {
			m.logger.Error("failed to remove peer on auto-close",
				"session_id", session.ID,
				"error", err,
			)
		}

		// Обновляем статус
		now := time.Now()
		if err := m.updateSessionStatus(ctx, session.ID, "expired", &now); err != nil {
			m.logger.Error("failed to update session on auto-close",
				"session_id", session.ID,
				"error", err,
			)
		}

		// Удаляем из activeSessions
		m.mu.Lock()
		delete(m.activeSessions, session.ID.String())
		m.mu.Unlock()

	case <-m.cleanupStop:
		return
	}
}
