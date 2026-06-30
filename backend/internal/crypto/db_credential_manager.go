// Package crypto — DB-backed Credential Manager implementation.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-02: DBCredentialManager
//
// Реализация CredentialManager с хранением в PostgreSQL.
// Username/password шифруются через stb.DefaultCrypto (AES-256-GCM / belt-gcm)
// перед записью в таблицу device_credentials.
//
// Каждая операция логируется в audit_log с device_id, operation и trace_id.
//
// Compliance:
//   - ISO 27001 A.12.4: Audit logging
//   - ISO 27001 A.10.1: Cryptographic controls
//   - OWASP ASVS V7.1: Error handling (no information leakage)
//   - Приказ ОАЦ №66 п. 7.18.3: Криптографическая защита
//
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/trace"
)

// DBCredentialManager реализует CredentialManager с хранением в PostgreSQL.
type DBCredentialManager struct {
	pool       *pgxpool.Pool
	auditSigner *audit.Signer
	logger     *slog.Logger
}

// NewDBCredentialManager создаёт новый DBCredentialManager.
func NewDBCredentialManager(pool *pgxpool.Pool, auditSigner *audit.Signer, logger *slog.Logger) *DBCredentialManager {
	return &DBCredentialManager{
		pool:        pool,
		auditSigner: auditSigner,
		logger:      logger.With("component", "credential_manager"),
	}
}

// Store сохраняет credentials для устройства.
// Шифрует username/password перед записью в БД.
// Возвращает error если credentials уже существуют.
func (m *DBCredentialManager) Store(ctx context.Context, deviceID, username, password string) error {
	if deviceID == "" {
		return ErrDeviceIDRequired
	}

	// Шифруем username и password
	usernameEnc, err := Encrypt(username)
	if err != nil {
		return fmt.Errorf("encrypt username: %w", err)
	}
	passwordEnc, err := Encrypt(password)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}

	// Декодируем hex в bytea для PostgreSQL
	usernameBytes, err := hex.DecodeString(usernameEnc)
	if err != nil {
		return fmt.Errorf("decode username hex: %w", err)
	}
	passwordBytes, err := hex.DecodeString(passwordEnc)
	if err != nil {
		return fmt.Errorf("decode password hex: %w", err)
	}

	query := `
		INSERT INTO device_credentials (device_id, username_enc, password_enc, algorithm, key_ref)
		VALUES ($1, $2, $3, 'aes-256-gcm', 'primary')
		ON CONFLICT (device_id) DO NOTHING
		RETURNING id`

	var credID string
	err = m.pool.QueryRow(ctx, query, deviceID, usernameBytes, passwordBytes).Scan(&credID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("credentials already exist for device %s: use Rotate instead", deviceID)
		}
		return fmt.Errorf("store credentials: %w", err)
	}

	m.logAudit(ctx, "STORE", deviceID, "credentials stored")
	m.logger.Info("credentials stored", "device_id", deviceID, "cred_id", credID)
	return nil
}

// Retrieve возвращает username/password для устройства.
// Дешифрует данные при чтении из БД.
func (m *DBCredentialManager) Retrieve(ctx context.Context, deviceID string) (string, string, error) {
	if deviceID == "" {
		return "", "", ErrDeviceIDRequired
	}

	query := `
		SELECT username_enc, password_enc
		FROM device_credentials
		WHERE device_id = $1
		AND (expires_at IS NULL OR expires_at > NOW())`

	var usernameBytes, passwordBytes []byte
	err := m.pool.QueryRow(ctx, query, deviceID).Scan(&usernameBytes, &passwordBytes)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", fmt.Errorf("%w: %s", ErrCredentialNotFound, deviceID)
		}
		return "", "", fmt.Errorf("retrieve credentials: %w", err)
	}

	// Дешифруем
	username, err := Decrypt(hex.EncodeToString(usernameBytes))
	if err != nil {
		return "", "", fmt.Errorf("decrypt username: %w", err)
	}
	password, err := Decrypt(hex.EncodeToString(passwordBytes))
	if err != nil {
		return "", "", fmt.Errorf("decrypt password: %w", err)
	}

	m.logAudit(ctx, "RETRIEVE", deviceID, "credentials retrieved")
	return username, password, nil
}

// Rotate обновляет credentials для устройства.
func (m *DBCredentialManager) Rotate(ctx context.Context, deviceID, newUsername, newPassword string) error {
	if deviceID == "" {
		return ErrDeviceIDRequired
	}

	// Шифруем новые credentials
	usernameEnc, err := Encrypt(newUsername)
	if err != nil {
		return fmt.Errorf("encrypt new username: %w", err)
	}
	passwordEnc, err := Encrypt(newPassword)
	if err != nil {
		return fmt.Errorf("encrypt new password: %w", err)
	}

	usernameBytes, err := hex.DecodeString(usernameEnc)
	if err != nil {
		return fmt.Errorf("decode username hex: %w", err)
	}
	passwordBytes, err := hex.DecodeString(passwordEnc)
	if err != nil {
		return fmt.Errorf("decode password hex: %w", err)
	}

	query := `
		INSERT INTO device_credentials (device_id, username_enc, password_enc, algorithm, key_ref)
		VALUES ($1, $2, $3, 'aes-256-gcm', 'primary')
		ON CONFLICT (device_id)
		DO UPDATE SET username_enc = $2, password_enc = $3, updated_at = NOW()`

	tag, err := m.pool.Exec(ctx, query, deviceID, usernameBytes, passwordBytes)
	if err != nil {
		return fmt.Errorf("rotate credentials: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	m.logAudit(ctx, "ROTATE", deviceID, "credentials rotated")
	m.logger.Info("credentials rotated", "device_id", deviceID)
	return nil
}

// Delete удаляет credentials для устройства.
func (m *DBCredentialManager) Delete(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return ErrDeviceIDRequired
	}

	query := `DELETE FROM device_credentials WHERE device_id = $1`
	tag, err := m.pool.Exec(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("delete credentials: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrCredentialNotFound, deviceID)
	}

	m.logAudit(ctx, "DELETE", deviceID, "credentials deleted")
	m.logger.Info("credentials deleted", "device_id", deviceID)
	return nil
}

// logAudit логирует операцию в audit_log.
func (m *DBCredentialManager) logAudit(ctx context.Context, action, deviceID, description string) {
	traceID := trace.FromContext(ctx)
	if traceID == "" {
		traceID = "unknown"
	}

	entry := fmt.Sprintf("CREDENTIAL:%s:%s:%s:%d", action, deviceID, traceID, time.Now().UnixNano())
	signature := ""
	if m.auditSigner != nil {
		signature = m.auditSigner.Sign(entry)
	}

	m.logger.Info("audit",
		"action", action,
		"device_id", deviceID,
		"trace_id", traceID,
		"signature", signature,
		"description", description,
	)
}
