// Package crypto — Credential Manager для безопасного хранения credentials устройств.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-02: Credential Manager Interface + DB Implementation
//
// Управляет шифрованием и хранением username/password для устройств
// видеонаблюдения. Использует существующий Encryptor (stb.DefaultCrypto)
// для шифрования перед записью в БД.
//
// Compliance:
//   - ISO 27001 A.9.2.1: User registration and de-registration
//   - ISO 27001 A.9.4.2: Secure log-on procedures
//   - ISO 27001 A.10.1.1: Cryptographic controls (encryption at rest)
//   - IEC 62443-3-3 SR 1.5: Authenticator management
//   - OWASP ASVS V2.1: Verify credentials are stored using approved crypto
//   - OWASP ASVS V2.5: Verify credentials are encrypted at rest
//   - СТБ 34.101.27 п. 7.18: Защита аутентификационных данных
//   - Приказ ОАЦ №66 п. 7.18.3: Криптографическая защита
//
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"context"
	"errors"
)

// ErrCredentialNotFound возвращается, когда credentials для устройства не найдены.
var ErrCredentialNotFound = errors.New("credential not found for device")

// ErrDeviceIDRequired возвращается при пустом device_id.
var ErrDeviceIDRequired = errors.New("device_id is required")

// CredentialManager определяет интерфейс для управления credentials устройств.
//
// Все методы принимают контекст для traceID propagation и cancellation.
// Username/password шифруются перед записью в БД и дешифруются при чтении.
// Каждая операция логируется в audit_log (ISO 27001 A.12.4).
type CredentialManager interface {
	// Store сохраняет credentials для устройства.
	// Шифрует username/password перед записью в БД.
	// Если credentials уже существуют — возвращает error (используйте Update).
	Store(ctx context.Context, deviceID, username, password string) error

	// Retrieve возвращает username/password для устройства.
	// Дешифрует данные при чтении из БД.
	// Возвращает ErrCredentialNotFound если credentials не найдены.
	Retrieve(ctx context.Context, deviceID string) (username, password string, err error)

	// Rotate обновляет credentials для устройства.
	// Отличается от Store тем, что не требует предварительного удаления.
	// Логирует предыдущие credentials в audit_log.
	Rotate(ctx context.Context, deviceID, newUsername, newPassword string) error

	// Delete удаляет credentials для устройства.
	// Логирует факт удаления в audit_log.
	Delete(ctx context.Context, deviceID string) error
}

// CredentialRecord представляет запись credentials в БД.
type CredentialRecord struct {
	DeviceID    string `json:"device_id"`
	UsernameEnc []byte `json:"-"` // зашифрованный username
	PasswordEnc []byte `json:"-"` // зашифрованный password
	Algorithm   string `json:"algorithm"`
	KeyRef      string `json:"key_ref"`
}
