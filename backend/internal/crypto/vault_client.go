// Package crypto — HashiCorp Vault integration for credential management.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-05: Vault Client for Master Key Storage
//
// Обеспечивает безопасное хранение master keys в HashiCorp Vault.
// Используется CredentialRotator для хранения/получения ключей шифрования
// паролей устройств.
//
// Compliance:
//   - IEC 62443-3-3 SR 2.2: Password management
//   - IEC 62443-3-3 SR 4.2: Centralized key management
//   - ISO 27001 A.9.2.3: Password management
//   - ISO 27001 A.10.1.1: Cryptographic key management
//   - СТБ 34.101.27 п. 5.1: Контроль доступа к ключевой информации
//   - Приказ ОАЦ №66 п. 7.18.3: Криптографическая защита
//
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	vaultapi "github.com/hashicorp/vault/api"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// VaultClient предоставляет интерфейс к HashiCorp Vault для хранения
// и получения master keys для шифрования credentials устройств.
//
// Путь в Vault: {mountPath}/data/devices/{deviceID}/master
// Структура данных:
//
//	{
//	  "master_key": "<base64-encoded key>"
//	}
type VaultClient struct {
	client *vaultapi.Logical
	path   string
	logger *slog.Logger
}

// VaultConfig — конфигурация подключения к HashiCorp Vault.
type VaultConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Address   string `mapstructure:"address"`
	Token     string `mapstructure:"token"` // из env VAULT_TOKEN
	MountPath string `mapstructure:"mount_path"`
}

// ────────────────────────────────────────────────────────────────────────────
// Constructor
// ────────────────────────────────────────────────────────────────────────────

// NewVaultClient создаёт новый VaultClient.
// Возвращает (nil, nil) если Vault отключён (config.Enabled == false).
// В production (КИИ РБ) Vault ДОЛЖЕН быть включён для хранения master keys.
func NewVaultClient(config VaultConfig, logger *slog.Logger) (*VaultClient, error) {
	if !config.Enabled {
		logger.Warn("CRED-05: vault integration disabled, master keys will not be persisted externally")
		return nil, nil
	}

	vcfg := vaultapi.DefaultConfig()
	vcfg.Address = config.Address

	client, err := vaultapi.NewClient(vcfg)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}
	client.SetToken(config.Token)

	path := config.MountPath
	if path == "" {
		path = "secret"
	}

	return &VaultClient{
		client: client.Logical(),
		path:   path,
		logger: logger.With("component", "vault_client"),
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Public Methods
// ────────────────────────────────────────────────────────────────────────────

// StoreMasterKey сохраняет master key для устройства в Vault.
//
// Путь: {mountPath}/data/devices/{deviceID}/master
// Данные:
//
//	{ "master_key": "<base64>" }
//
// Соответствует:
//   - ISO 27001 A.10.1.1: Cryptographic key management
//   - IEC 62443-3-3 SR 4.2: Centralized key management
func (v *VaultClient) StoreMasterKey(ctx context.Context, deviceID string, key []byte) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("CRED-05: vault client not initialized, enable vault in config")
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"master_key": base64.StdEncoding.EncodeToString(key),
		},
	}

	secretPath := fmt.Sprintf("%s/data/devices/%s/master", v.path, deviceID)

	_, err := v.client.WriteWithContext(ctx, secretPath, data)
	if err != nil {
		return fmt.Errorf("CRED-05: store master key in vault: %w", err)
	}

	v.logger.Info("master key stored in vault", "device_id", deviceID)
	return nil
}

// GetMasterKey получает master key для устройства из Vault.
//
// Возвращает ошибку если ключ не найден или Vault недоступен.
func (v *VaultClient) GetMasterKey(ctx context.Context, deviceID string) ([]byte, error) {
	if v == nil || v.client == nil {
		return nil, fmt.Errorf("CRED-05: vault client not initialized, enable vault in config")
	}

	secretPath := fmt.Sprintf("%s/data/devices/%s/master", v.path, deviceID)

	secret, err := v.client.ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("CRED-05: read master key from vault: %w", err)
	}
	if secret == nil || secret.Data["data"] == nil {
		return nil, fmt.Errorf("CRED-05: master key not found for device %s", deviceID)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CRED-05: invalid vault response format for device %s", deviceID)
	}

	keyStr, ok := data["master_key"].(string)
	if !ok || keyStr == "" {
		return nil, fmt.Errorf("CRED-05: master_key field not found or empty for device %s", deviceID)
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("CRED-05: decode master key from base64: %w", err)
	}

	v.logger.Debug("master key retrieved from vault", "device_id", deviceID)
	return key, nil
}

// DeleteMasterKey удаляет master key для устройства из Vault.
// Используется при удалении устройства или полной очистке credentials.
func (v *VaultClient) DeleteMasterKey(ctx context.Context, deviceID string) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("CRED-05: vault client not initialized, enable vault in config")
	}

	secretPath := fmt.Sprintf("%s/metadata/devices/%s/master", v.path, deviceID)

	_, err := v.client.DeleteWithContext(ctx, secretPath)
	if err != nil {
		return fmt.Errorf("CRED-05: delete master key from vault: %w", err)
	}

	v.logger.Info("master key deleted from vault", "device_id", deviceID)
	return nil
}
