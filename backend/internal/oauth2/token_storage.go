// Package oauth2 предоставляет общее хранилище OAuth2 токенов для внешних адаптеров.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.2: OAuth2 for External Adapters
//
// TokenStorage — зашифрованное хранилище OAuth2 токенов в БД.
//   - Шифрование: crypto.Encrypt/Decrypt (AES-256-GCM → belt-gcm)
//   - Multi-tenant: RLS на уровне БД
//   - Audit trail: каждая мутация логируется
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application data)
//   - ISO 27001 A.10.1 (Cryptographic controls)
//   - СТБ 34.101.30 (belt-gcm after migration)
//   - OWASP ASVS V6.2 (Stored credentials)
//
// ═══════════════════════════════════════════════════════════════════════════
package oauth2

import (
	"context"
	"fmt"
	"time"

	"gb-telemetry-collector/internal/crypto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

// ────────────────────────────────────────────────────────────────────────────
// TokenStore — интерфейс для хранения OAuth2 токенов.
// ────────────────────────────────────────────────────────────────────────────

// TokenStore определяет контракт для сохранения и загрузки токенов.
// Может быть реализован для БД, Vault, или in-memory (для тестов).
type TokenStore interface {
	// GetToken возвращает сохранённый токен для провайдера.
	GetToken(ctx context.Context, provider, providerKey string) (*oauth2.Token, error)

	// SaveToken сохраняет токен для провайдера.
	SaveToken(ctx context.Context, provider, providerKey string, token *oauth2.Token) error

	// DeleteToken удаляет сохранённый токен.
	DeleteToken(ctx context.Context, provider, providerKey string) error
}

// ────────────────────────────────────────────────────────────────────────────
// PGTokensStore — реализация TokenStore через PostgreSQL.
// ────────────────────────────────────────────────────────────────────────────

// PGTokensStore хранит OAuth2 токены в таблице oauth2_tokens.
// Токены шифруются перед сохранением через crypto.Encrypt/Decrypt.
type PGTokensStore struct {
	pool *pgxpool.Pool
}

// NewPGTokensStore создаёт новое PostgreSQL-хранилище токенов.
func NewPGTokensStore(pool *pgxpool.Pool) *PGTokensStore {
	return &PGTokensStore{pool: pool}
}

// GetToken загружает и расшифровывает токен из БД.
func (s *PGTokensStore) GetToken(ctx context.Context, provider, providerKey string) (*oauth2.Token, error) {
	var (
		encryptedAccess  string
		encryptedRefresh string
		tokenType        string
		expiry           *time.Time
	)

	err := s.pool.QueryRow(ctx, `
		SELECT access_token, refresh_token, token_type, expiry
		FROM oauth2_tokens
		WHERE provider = $1 AND provider_key = $2
	`, provider, providerKey).Scan(&encryptedAccess, &encryptedRefresh, &tokenType, &expiry)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // token not found — not an error
		}
		return nil, fmt.Errorf("oauth2: get token for %s/%s: %w", provider, providerKey, err)
	}

	// Расшифровываем токены
	accessToken, err := crypto.Decrypt(encryptedAccess)
	if err != nil {
		return nil, fmt.Errorf("oauth2: decrypt access token for %s/%s: %w", provider, providerKey, err)
	}

	token := &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   tokenType,
	}

	if encryptedRefresh != "" {
		refreshToken, err := crypto.Decrypt(encryptedRefresh)
		if err != nil {
			return nil, fmt.Errorf("oauth2: decrypt refresh token for %s/%s: %w", provider, providerKey, err)
		}
		token.RefreshToken = refreshToken
	}

	if expiry != nil {
		token.Expiry = *expiry
	}

	return token, nil
}

// SaveToken шифрует и сохраняет токен в БД (upsert).
func (s *PGTokensStore) SaveToken(ctx context.Context, provider, providerKey string, token *oauth2.Token) error {
	// Шифруем токены
	encryptedAccess, err := crypto.Encrypt(token.AccessToken)
	if err != nil {
		return fmt.Errorf("oauth2: encrypt access token for %s/%s: %w", provider, providerKey, err)
	}

	var encryptedRefresh string
	if token.RefreshToken != "" {
		encryptedRefresh, err = crypto.Encrypt(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("oauth2: encrypt refresh token for %s/%s: %w", provider, providerKey, err)
		}
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	var expiry *time.Time
	if !token.Expiry.IsZero() {
		expiry = &token.Expiry
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO oauth2_tokens (provider, provider_key, access_token, refresh_token, token_type, expiry)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (provider, provider_key) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_type = EXCLUDED.token_type,
			expiry = EXCLUDED.expiry,
			updated_at = NOW()
	`, provider, providerKey, encryptedAccess, encryptedRefresh, tokenType, expiry)

	if err != nil {
		return fmt.Errorf("oauth2: save token for %s/%s: %w", provider, providerKey, err)
	}

	return nil
}

// DeleteToken удаляет токен из БД.
func (s *PGTokensStore) DeleteToken(ctx context.Context, provider, providerKey string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM oauth2_tokens WHERE provider = $1 AND provider_key = $2
	`, provider, providerKey)

	if err != nil {
		return fmt.Errorf("oauth2: delete token for %s/%s: %w", provider, providerKey, err)
	}

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// InMemoryTokenStore — для тестов
// ────────────────────────────────────────────────────────────────────────────

// InMemoryTokenStore хранит токены в памяти (только для тестов).
type InMemoryTokenStore struct {
	tokens map[string]*oauth2.Token
}

// NewInMemoryTokenStore создаёт in-memory хранилище для тестов.
func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{tokens: make(map[string]*oauth2.Token)}
}

func (s *InMemoryTokenStore) GetToken(ctx context.Context, provider, providerKey string) (*oauth2.Token, error) {
	key := provider + ":" + providerKey
	token, ok := s.tokens[key]
	if !ok {
		return nil, nil
	}
	return token, nil
}

func (s *InMemoryTokenStore) SaveToken(ctx context.Context, provider, providerKey string, token *oauth2.Token) error {
	key := provider + ":" + providerKey
	s.tokens[key] = token
	return nil
}

func (s *InMemoryTokenStore) DeleteToken(ctx context.Context, provider, providerKey string) error {
	key := provider + ":" + providerKey
	delete(s.tokens, key)
	return nil
}
