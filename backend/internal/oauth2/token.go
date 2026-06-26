// Package oauth2 — управление OAuth2 токенами для внешних адаптеров.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.2: OAuth2 for External Adapters
//
// TokenManager объединяет:
//   - OAuth2 Client Credentials flow (через golang.org/x/oauth2)
//   - Token auto-refresh с grace-периодом
//   - Зашифрованное хранение токенов в БД
//   - Graceful fallback при ошибках
//   - Метрики (token_refresh_count, token_expired_count)
//
// Compliance:
//   - IEC 62443 SL-3 (Application integrity)
//   - ISO 27001 A.9.4.2 (Secure authentication)
//   - OWASP ASVS V2.1 (Password/credential storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package oauth2

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

// GracePeriod — время до истечения токена, когда начинается превентивный refresh.
const GracePeriod = 30 * time.Second

// ────────────────────────────────────────────────────────────────────────────
// TokenManager — управляет OAuth2 токенами для внешнего адаптера.
// ────────────────────────────────────────────────────────────────────────────

// TokenManagerConfig — конфигурация TokenManager.
type TokenManagerConfig struct {
	// ClientID — OAuth2 Client ID.
	ClientID string

	// ClientSecret — OAuth2 Client Secret.
	ClientSecret string

	// TokenURL — endpoint для получения токена.
	TokenURL string

	// Scopes — запрашиваемые scope.
	Scopes []string

	// AuthStyle — стиль аутентификации (InHeader или InParams).
	AuthStyle oauth2.AuthStyle
}

// TokenManager предоставляет OAuth2 токен с auto-refresh и персистентным хранилищем.
type TokenManager struct {
	config  *TokenManagerConfig
	store   TokenStore
	metrics *Metrics
	logger  *slog.Logger

	// Параметры провайдера для хранения/загрузки из БД
	provider    string // 'servicenow', 'jira', 'toir'
	providerKey string // instance URL или tenant ID

	mu    sync.RWMutex
	token *oauth2.Token
}

// NewTokenManager создаёт новый TokenManager.
//
// Параметры:
//   - cfg: OAuth2 Client Credentials конфигурация (если пустой — basic auth fallback)
//   - store: TokenStore для персистентного хранения
//   - metrics: счётчики для мониторинга
//   - logger: логгер
//   - provider: имя провайдера (servicenow, jira, toir)
//   - providerKey: уникальный ключ для данного провайдера (instance URL)
func NewTokenManager(
	cfg *TokenManagerConfig,
	store TokenStore,
	metrics *Metrics,
	logger *slog.Logger,
	provider, providerKey string,
) *TokenManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &TokenManager{
		config:      cfg,
		store:       store,
		metrics:     metrics,
		logger:      logger.With("component", "oauth2-token-manager", "provider", provider),
		provider:    provider,
		providerKey: providerKey,
	}
}

// IsConfigured проверяет, настроен ли OAuth2 (клиентские данные).
func (m *TokenManager) IsConfigured() bool {
	return m.config != nil && m.config.ClientID != "" && m.config.ClientSecret != "" && m.config.TokenURL != ""
}

// GetToken возвращает валидный токен, с auto-refresh если истёк.
// Если токен близок к истечению (GracePeriod), инициирует превентивный refresh.
func (m *TokenManager) GetToken(ctx context.Context) (*oauth2.Token, error) {
	// Быстрое чтение без блокировки
	m.mu.RLock()
	token := m.token
	m.mu.RUnlock()

	if token != nil && !isExpired(token) {
		return token, nil
	}

	// Токен истёк или отсутствует — полная блокировка для refresh
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check после блокировки
	if m.token != nil && !isExpired(m.token) {
		return m.token, nil
	}

	// Пытаемся загрузить из хранилища
	if m.store != nil {
		storedToken, err := m.store.GetToken(ctx, m.provider, m.providerKey)
		if err != nil {
			m.logger.Warn("failed to load token from store, will re-fetch", "error", err)
		}
		if storedToken != nil && !isExpired(storedToken) {
			m.token = storedToken
			m.metrics.IncLoad()
			return storedToken, nil
		}
	}

	// Токена в хранилище нет или он истёк — получаем новый через Client Credentials
	if !m.IsConfigured() {
		return nil, fmt.Errorf("oauth2: token manager not configured for %s/%s", m.provider, m.providerKey)
	}

	newToken, err := m.fetchToken(ctx)
	if err != nil {
		m.metrics.IncRefreshError()
		return nil, fmt.Errorf("oauth2: fetch token for %s/%s: %w", m.provider, m.providerKey, err)
	}

	// Сохраняем в хранилище и в памяти
	m.token = newToken
	m.metrics.IncTokenRefresh()

	if m.store != nil {
		if saveErr := m.store.SaveToken(ctx, m.provider, m.providerKey, newToken); saveErr != nil {
			m.logger.Warn("failed to save token to store", "error", saveErr)
		} else {
			m.metrics.IncSave()
		}
	}

	return newToken, nil
}

// ForceRefresh принудительно обновляет токен (игнорируя кэш).
func (m *TokenManager) ForceRefresh(ctx context.Context) (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.IsConfigured() {
		return nil, fmt.Errorf("oauth2: token manager not configured for %s/%s", m.provider, m.providerKey)
	}

	newToken, err := m.fetchToken(ctx)
	if err != nil {
		m.metrics.IncRefreshError()
		return nil, fmt.Errorf("oauth2: force refresh for %s/%s: %w", m.provider, m.providerKey, err)
	}

	m.token = newToken
	m.metrics.IncTokenRefresh()

	if m.store != nil {
		if saveErr := m.store.SaveToken(ctx, m.provider, m.providerKey, newToken); saveErr != nil {
			m.logger.Warn("failed to save token after force refresh", "error", saveErr)
		} else {
			m.metrics.IncSave()
		}
	}

	return newToken, nil
}

// InvalidateToken инвалидирует текущий токен (при 401 от внешнего сервиса).
func (m *TokenManager) InvalidateToken(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.token = nil
	m.metrics.IncTokenExpired()

	if m.store != nil {
		if err := m.store.DeleteToken(ctx, m.provider, m.providerKey); err != nil {
			m.logger.Warn("failed to delete invalidated token", "error", err)
		}
	}

	m.logger.Info("token invalidated (will refresh on next request)")
}

// Metrics возвращает счётчики метрик.
func (m *TokenManager) Metrics() *Metrics {
	return m.metrics
}

// ── Internal ──────────────────────────────────────────────────────────────

// fetchToken получает новый токен через Client Credentials.
func (m *TokenManager) fetchToken(ctx context.Context) (*oauth2.Token, error) {
	cfg := &clientcredentials.Config{
		ClientID:     m.config.ClientID,
		ClientSecret: m.config.ClientSecret,
		TokenURL:     m.config.TokenURL,
		Scopes:       m.config.Scopes,
		AuthStyle:    m.config.AuthStyle,
	}

	if cfg.AuthStyle == 0 {
		cfg.AuthStyle = oauth2.AuthStyleInHeader
	}

	token, err := cfg.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("client credentials token: %w", err)
	}

	return token, nil
}

// isExpired проверяет, истёк ли токен (с учётом grace-периода).
func isExpired(token *oauth2.Token) bool {
	if token == nil {
		return true
	}
	if token.Expiry.IsZero() {
		return false // токен без expiry считается бессрочным
	}
	return time.Now().After(token.Expiry.Add(-GracePeriod))
}
