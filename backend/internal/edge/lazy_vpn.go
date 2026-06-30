// ═══════════════════════════════════════════════════════════════════════════
// Package edge — Lazy VPN Session (PROXY-03)
//
// LazyVPNSession предоставляет отложенное создание VPN сессий:
// - Создаёт VPN-сессию только при первом обращении (GetOrCreateSession)
// - Переиспользует активные сессии для одного engineer_id + agent_id
// - Продлевает TTL при каждом обращении (скользящее окно)
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation, session management
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - Приказ ОАЦ №66 п. 7.18.2: Контроль удалённого доступа
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultLazyVPNSessionTTL — TTL lazy VPN сессии по умолчанию.
const DefaultLazyVPNSessionTTL = 1 * time.Hour

// LazyVPNSession предоставляет ленивое создание VPN сессий.
//
// PROXY-03: Сессия создаётся только при первом обращении инженера
// к устройству через Edge Proxy. Повторные запросы переиспользуют
// активную сессию и продлевают её TTL.
type LazyVPNSession struct {
	manager *VPNSessionManager
	logger  *slog.Logger

	// Локальный кэш активных сессий (engineer_id+agent_id → session)
	mu       sync.RWMutex
	sessions map[string]*cachedSession
}

// cachedSession — сессия в кэше LazyVPNSession.
type cachedSession struct {
	session   *VPNSession
	expiresAt time.Time
}

// sessionKey формирует ключ для кэша: engineer_id + agent_id.
func sessionKey(engineerID uuid.UUID, agentID string) string {
	return engineerID.String() + ":" + agentID
}

// NewLazyVPNSession создаёт новый LazyVPNSession.
func NewLazyVPNSession(manager *VPNSessionManager, logger *slog.Logger) *LazyVPNSession {
	return &LazyVPNSession{
		manager:  manager,
		logger:   logger.With("component", "lazy-vpn-session"),
		sessions: make(map[string]*cachedSession),
	}
}

// GetOrCreateSession возвращает активную VPN сессию для engineer_id + agent_id.
//
// Flow:
//  1. Ищем активную сессию в кэше
//  2. Если есть и не истекла — возвращаем
//  3. Если есть в БД (active, не истекла) — загружаем в кэш и возвращаем
//  4. Если нет — создаём новую через VPNSessionManager
//  5. Продлеваем TTL при каждом обращении
//
// Compliance:
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - Приказ ОАЦ №66 п. 7.18.2: Контроль удалённого доступа
func (l *LazyVPNSession) GetOrCreateSession(
	ctx context.Context,
	agentID string,
	engineerID uuid.UUID,
	allowedIPs []net.IPNet,
) (*VPNSession, error) {
	key := sessionKey(engineerID, agentID)
	logger := l.logger.With("agent_id", agentID, "engineer_id", engineerID)

	// 1. Проверяем кэш
	l.mu.RLock()
	if cached, ok := l.sessions[key]; ok {
		if time.Now().Before(cached.expiresAt) && cached.session.Status == "active" {
			l.mu.RUnlock()
			logger.Debug("reusing cached vpn session", "session_id", cached.session.ID)
			// Продлеваем TTL через горутину
			go l.extendSessionTTL(ctx, cached.session)
			return cached.session, nil
		}
	}
	l.mu.RUnlock()

	// 2. Ищем активную сессию в БД
	sessions, err := l.manager.GetSessions(ctx, SessionFilter{
		AgentID:    agentID,
		EngineerID: engineerID,
		Status:     "active",
		Limit:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("lazy-vpn: failed to query sessions: %w", err)
	}

	if len(sessions) > 0 {
		s := &sessions[0]
		if time.Now().Before(s.ExpiresAt) {
			logger.Debug("found active session in db", "session_id", s.ID)
			l.cacheSession(key, s)
			go l.extendSessionTTL(ctx, s)
			return s, nil
		}
	}

	// 3. Создаём новую сессию
	logger.Info("creating new lazy vpn session")

	createReq := CreateSessionRequest{
		AgentID:    agentID,
		EngineerID: engineerID,
		AllowedIPs: allowedIPs,
		Duration: Duration{
			Duration: DefaultLazyVPNSessionTTL,
		},
	}

	session, err := l.manager.CreateSession(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("lazy-vpn: failed to create session: %w", err)
	}

	l.cacheSession(key, session)
	logger.Info("lazy vpn session created", "session_id", session.ID)
	return session, nil
}

// cacheSession добавляет сессию в кэш.
func (l *LazyVPNSession) cacheSession(key string, session *VPNSession) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sessions[key] = &cachedSession{
		session:   session,
		expiresAt: session.ExpiresAt,
	}
}

// extendSessionTTL продлевает TTL сессии при обращении (скользящее окно).
//
// Compliance:
//   - ISO 27001 A.12.4: Audit trail — продление логируется
//   - IEC 62443-3-3 SR 2.1: Session timeout management
func (l *LazyVPNSession) extendSessionTTL(ctx context.Context, session *VPNSession) {
	if session.Status != "active" {
		return
	}

	newExpiry := time.Now().Add(DefaultLazyVPNSessionTTL)
	if newExpiry.After(session.ExpiresAt) {
		l.logger.Debug("extending session ttl",
			"session_id", session.ID,
			"old_expiry", session.ExpiresAt,
			"new_expiry", newExpiry,
		)
		session.ExpiresAt = newExpiry

		// Обновляем в кэше
		key := sessionKey(session.EngineerID, session.AgentID)
		l.mu.Lock()
		if cached, ok := l.sessions[key]; ok && cached.session.ID == session.ID {
			cached.expiresAt = newExpiry
		}
		l.mu.Unlock()
	}
}

// InvalidateCache удаляет сессию из кэша (при revoke).
func (l *LazyVPNSession) InvalidateCache(engineerID uuid.UUID, agentID string) {
	key := sessionKey(engineerID, agentID)
	l.mu.Lock()
	delete(l.sessions, key)
	l.mu.Unlock()
	l.logger.Debug("invalidated session cache", "key", key)
}
