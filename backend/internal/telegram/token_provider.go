// ═══════════════════════════════════════════════════════════════════════════
// Package telegram — Token Provider (P2-MED-04)
//
// TokenProvider предоставляет абстракцию для получения токена Telegram бота
// с поддержкой Vault и environment variable fallback.
//
// Проблема: Токен Telegram бота хранился в plaintext config.yaml.
// Решение: Читать из Vault (VAULT_ENABLED=true) или GB_TELEGRAM_TOKEN env var.
//
// Поддержка rotation: TokenProvider.GetToken() вызывается при каждом
// переподключении бота, что позволяет ротировать токен без перезапуска.
//
// Соответствие:
//   - IEC 62443-3-3 SR 4.2: Централизованное управление секретами
//   - ISO 27001 A.9.4.3: Password management system
//   - OWASP ASVS V2.10: Secret storage
//   - Приказ ОАЦ №66 п. 7.18.4: Защита credentials
// ═══════════════════════════════════════════════════════════════════════════

package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// TokenProvider — интерфейс для получения токена Telegram бота.
//
// P2-MED-04: Поддерживает Vault + env fallback для rotation.
type TokenProvider interface {
	// GetToken возвращает текущий токен Telegram бота.
	// При rotation (смена токена в Vault) возвращает новый токен.
	GetToken(ctx context.Context) (string, error)
}

// EnvTokenProvider читает токен из переменной окружения.
type EnvTokenProvider struct {
	envVar string
	logger *slog.Logger
}

// NewEnvTokenProvider создаёт провайдер, читающий токен из env.
func NewEnvTokenProvider(envVar string, logger *slog.Logger) *EnvTokenProvider {
	return &EnvTokenProvider{
		envVar: envVar,
		logger: logger.With("component", "env-token-provider"),
	}
}

// GetToken читает токен из переменной окружения.
func (p *EnvTokenProvider) GetToken(_ context.Context) (string, error) {
	token := os.Getenv(p.envVar)
	if token == "" {
		return "", fmt.Errorf("env-token: %s is not set", p.envVar)
	}
	return token, nil
}

// VaultTokenProvider читает токен из HashiCorp Vault.
type VaultTokenProvider struct {
	client      VaultReader
	path        string
	field       string
	logger      *slog.Logger
	envFallback EnvTokenProvider
}

// VaultReader — интерфейс для чтения секретов из Vault.
type VaultReader interface {
	ReadSecret(ctx context.Context, path string) (map[string]interface{}, error)
}

// NewVaultTokenProvider создаёт провайдер, читающий токен из Vault.
// Если Vault недоступен, использует env fallback.
func NewVaultTokenProvider(client VaultReader, path, field, envVar string, logger *slog.Logger) *VaultTokenProvider {
	return &VaultTokenProvider{
		client:      client,
		path:        path,
		field:       field,
		logger:      logger.With("component", "vault-token-provider"),
		envFallback: *NewEnvTokenProvider(envVar, logger),
	}
}

// GetToken читает токен из Vault с fallback на env.
func (p *VaultTokenProvider) GetToken(ctx context.Context) (string, error) {
	if p.client == nil {
		p.logger.Warn("vault client is nil, falling back to env")
		return p.envFallback.GetToken(ctx)
	}

	secret, err := p.client.ReadSecret(ctx, p.path)
	if err != nil {
		p.logger.Warn("failed to read token from vault, falling back to env",
			"path", p.path,
			"error", err,
		)
		return p.envFallback.GetToken(ctx)
	}

	tokenRaw, ok := secret[p.field]
	if !ok {
		p.logger.Warn("token field not found in vault secret, falling back to env",
			"path", p.path,
			"field", p.field,
		)
		return p.envFallback.GetToken(ctx)
	}

	token, ok := tokenRaw.(string)
	if !ok || token == "" {
		return "", fmt.Errorf("vault-token: invalid token type at %s/%s", p.path, p.field)
	}

	return token, nil
}
