// Package oauth2 — метрики для OAuth2 токенов.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.2: OAuth2 for External Adapters
//
// Метрики:
//   - oauth2_token_refresh_total  — количество успешных refresh токена
//   - oauth2_token_expired_total  — количество истекших токенов
//   - oauth2_token_refresh_errors — количество ошибок при refresh
//
// Используются атомарные счётчики (sync/atomic) для lock-free concurrent access.
// ═══════════════════════════════════════════════════════════════════════════
package oauth2

import (
	"sync/atomic"
)

// ────────────────────────────────────────────────────────────────────────────
// Metrics — атомарные счётчики для мониторинга OAuth2 токенов.
// ────────────────────────────────────────────────────────────────────────────

// Metrics содержит счётчики событий, связанных с OAuth2 токенами.
// Все поля — atomic для concurrent-safe доступа без блокировок.
type Metrics struct {
	tokenRefreshes atomic.Int64 // token_refresh_count
	tokenExpired   atomic.Int64 // token_expired_count
	refreshErrors  atomic.Int64 // token_refresh_error_count
	saves          atomic.Int64 // token_save_count
	loads          atomic.Int64 // token_load_count
}

// NewMetrics создаёт новый экземпляр Metrics.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// ── Инкременты ────────────────────────────────────────────────────────────

// IncTokenRefresh увеличивает счётчик успешных refresh.
func (m *Metrics) IncTokenRefresh() {
	m.tokenRefreshes.Add(1)
}

// IncTokenExpired увеличивает счётчик истекших токенов.
func (m *Metrics) IncTokenExpired() {
	m.tokenExpired.Add(1)
}

// IncRefreshError увеличивает счётчик ошибок refresh.
func (m *Metrics) IncRefreshError() {
	m.refreshErrors.Add(1)
}

// IncSave увеличивает счётчик сохранений токена.
func (m *Metrics) IncSave() {
	m.saves.Add(1)
}

// IncLoad увеличивает счётчик загрузок токена.
func (m *Metrics) IncLoad() {
	m.loads.Add(1)
}

// ── Геттеры ───────────────────────────────────────────────────────────────

// TokenRefreshCount возвращает количество успешных refresh.
func (m *Metrics) TokenRefreshCount() int64 {
	return m.tokenRefreshes.Load()
}

// TokenExpiredCount возвращает количество истекших токенов.
func (m *Metrics) TokenExpiredCount() int64 {
	return m.tokenExpired.Load()
}

// RefreshErrorCount возвращает количество ошибок refresh.
func (m *Metrics) RefreshErrorCount() int64 {
	return m.refreshErrors.Load()
}

// SaveCount возвращает количество сохранений.
func (m *Metrics) SaveCount() int64 {
	return m.saves.Load()
}

// LoadCount возвращает количество загрузок.
func (m *Metrics) LoadCount() int64 {
	return m.loads.Load()
}

// ── Snapshot ──────────────────────────────────────────────────────────────

// MetricsSnapshot — снимок метрик для экспорта.
type MetricsSnapshot struct {
	TokenRefreshCount int64 `json:"token_refresh_count"`
	TokenExpiredCount int64 `json:"token_expired_count"`
	RefreshErrorCount int64 `json:"token_refresh_error_count"`
	SaveCount         int64 `json:"token_save_count"`
	LoadCount         int64 `json:"token_load_count"`
}

// Snapshot возвращает текущие значения всех счётчиков.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		TokenRefreshCount: m.TokenRefreshCount(),
		TokenExpiredCount: m.TokenExpiredCount(),
		RefreshErrorCount: m.RefreshErrorCount(),
		SaveCount:         m.SaveCount(),
		LoadCount:         m.LoadCount(),
	}
}
