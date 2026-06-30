// Package crypto — Automatic Credential Rotation for CCTV devices.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-05: Automatic Credential Rotation (P2-EDGE)
//
// Обеспечивает автоматическую ротацию паролей устройств видеонаблюдения:
//   - Генерация cryptographically strong паролей (crypto/rand)
//   - Ротация через DevicePasswordChanger (HTTP API устройства)
//   - Хранение master keys в HashiCorp Vault
//   - Периодическая ротация по расписанию
//   - Уведомления о скором истечении credentials
//
// Compliance:
//   - IEC 62443-3-3 SR 2.2: Password management (регулярная смена паролей)
//   - IEC 62443-3-3 SR 1.5: Authenticator management
//   - ISO 27001 A.9.2.3: Password management
//   - ISO 27001 A.9.4.2: Secure log-on procedures
//   - ISO 27001 A.12.4: Audit logging
//   - СТБ 34.101.27 п. 5.1: Контроль доступа
//   - OWASP ASVS V2.1: Verify credentials stored using approved crypto
//
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"gb-telemetry-collector/internal/trace"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

// DefaultRotationInterval — интервал ротации по умолчанию (90 дней).
// Соответствует: IEC 62443-3-3 SR 2.2, ISO 27001 A.9.2.3
const DefaultRotationInterval = 90 * 24 * time.Hour

// DefaultExpiryThreshold — порог уведомления об истечении (14 дней до expiry).
const DefaultExpiryThreshold = 14 * 24 * time.Hour

// DefaultPasswordLength — длина пароля по умолчанию.
const DefaultPasswordLength = 24

// Минимальная длина пароля (IEC 62443-3-3 SR 2.2: минимум 8 символов).
const minPasswordLength = 12
const maxPasswordLength = 128

// Charsets для генерации паролей.
const (
	charsLower   = "abcdefghijklmnopqrstuvwxyz"
	charsUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsDigits  = "0123456789"
	charsSpecial = "!@#$%^&*()-_=+[]{}<>?,."
)

var defaultCharset = charsLower + charsUpper + charsDigits + charsSpecial

// ────────────────────────────────────────────────────────────────────────────
// Interfaces
// ────────────────────────────────────────────────────────────────────────────

// DevicePasswordChanger определяет интерфейс для смены пароля на устройстве.
// Реализуется через VendorDevice (HTTP API конкретного вендора: Hikvision, Dahua, и т.д.).
//
// Соответствует:
//   - IEC 62443-3-3 SR 2.2: Password management
//   - IEC 62443-3-3 SR 1.5: Authenticator management
type DevicePasswordChanger interface {
	// ChangePassword меняет пароль на устройстве.
	// Принимает текущий username и новый пароль.
	// Возвращает ошибку если устройство недоступно или отказало в смене.
	ChangePassword(ctx context.Context, username, newPassword string) error
}

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// CredentialRotatorConfig — параметры конфигурации ротатора.
type CredentialRotatorConfig struct {
	// RotationInterval — интервал автоматической ротации (default: 90 дней).
	RotationInterval time.Duration

	// ExpiryThreshold — за сколько до истечения отправлять уведомления (default: 14 дней).
	ExpiryThreshold time.Duration

	// PasswordLength — длина генерируемого пароля (default: 24, min: 12, max: 128).
	PasswordLength int
}

// DefaultCredentialRotatorConfig возвращает конфигурацию ротатора по умолчанию.
func DefaultCredentialRotatorConfig() CredentialRotatorConfig {
	return CredentialRotatorConfig{
		RotationInterval: DefaultRotationInterval,
		ExpiryThreshold:  DefaultExpiryThreshold,
		PasswordLength:   DefaultPasswordLength,
	}
}

// CredentialRotator управляет автоматической ротацией паролей устройств.
//
// Поток ротации:
//  1. GeneratePassword() — генерация нового cryptographically strong пароля
//  2. DevicePasswordChanger.ChangePassword() — смена пароля на устройстве
//  3. CredentialManager.Rotate() — сохранение нового пароля в БД (encrypted)
//  4. VaultClient.StoreMasterKey() — опционально, сохранение master key в Vault
//
// При ошибке на шаге 2: старый пароль сохраняется, возвращается ошибка.
type CredentialRotator struct {
	credentialMgr CredentialManager
	vaultClient   *VaultClient
	passwordChgr  DevicePasswordChanger
	config        CredentialRotatorConfig
	logger        *slog.Logger

	mu     sync.Mutex
	stopCh chan struct{}
}

// ────────────────────────────────────────────────────────────────────────────
// Constructor
// ────────────────────────────────────────────────────────────────────────────

// NewCredentialRotator создаёт новый CredentialRotator.
//
// Параметры:
//   - credentialMgr: менеджер credentials (обязателен)
//   - vaultClient: клиент Vault (может быть nil)
//   - passwordChgr: интерфейс для смены пароля на устройстве (обязателен)
//   - config: конфигурация (DefaultCredentialRotatorConfig() для значений по умолчанию)
func NewCredentialRotator(
	credentialMgr CredentialManager,
	vaultClient *VaultClient,
	passwordChgr DevicePasswordChanger,
	config CredentialRotatorConfig,
	logger *slog.Logger,
) (*CredentialRotator, error) {
	if credentialMgr == nil {
		return nil, fmt.Errorf("CRED-05: credentialManager is required")
	}
	if passwordChgr == nil {
		return nil, fmt.Errorf("CRED-05: devicePasswordChanger is required")
	}

	// Валидация длины пароля
	if config.PasswordLength < minPasswordLength {
		config.PasswordLength = minPasswordLength
	} else if config.PasswordLength > maxPasswordLength {
		config.PasswordLength = maxPasswordLength
	}

	// Валидация интервалов
	if config.RotationInterval <= 0 {
		config.RotationInterval = DefaultRotationInterval
	}
	if config.ExpiryThreshold <= 0 {
		config.ExpiryThreshold = DefaultExpiryThreshold
	}

	return &CredentialRotator{
		credentialMgr: credentialMgr,
		vaultClient:   vaultClient,
		passwordChgr:  passwordChgr,
		config:        config,
		logger:        logger.With("component", "credential_rotator"),
		stopCh:        make(chan struct{}),
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Password Generation
// ────────────────────────────────────────────────────────────────────────────

// GeneratePassword генерирует cryptographically strong пароль.
//
// Использует crypto/rand для всех операций:
//   - Минимум 1 символ из каждого набора (lower, upper, digit, special)
//   - Оставшиеся символы — случайные из полного charset'а
//   - Перемешивание через Fisher-Yates с crypto/rand
//
// Соответствует:
//   - IEC 62443-3-3 SR 2.2: Password management (сильные пароли)
//   - OWASP ASVS V2.1.1: Verify that passwords are at least 8 characters
//   - OWASP ASVS V2.1.7: Verify that passwords are generated with cryptographically secure RNG
func (r *CredentialRotator) GeneratePassword(length int) (string, error) {
	if length < minPasswordLength {
		length = minPasswordLength
	} else if length > maxPasswordLength {
		length = maxPasswordLength
	}

	// Гарантируем минимум 1 символ из каждого набора
	requiredSets := []string{charsLower, charsUpper, charsDigits, charsSpecial}
	if length < len(requiredSets) {
		length = len(requiredSets)
	}

	password := make([]byte, length)

	// Шаг 1: добавляем по одному символу из каждого набора
	idx := 0
	for _, set := range requiredSets {
		b, err := cryptoRandInt(len(set))
		if err != nil {
			return "", fmt.Errorf("CRED-05: generate random index: %w", err)
		}
		password[idx] = set[b.Int64()]
		idx++
	}

	// Шаг 2: заполняем оставшиеся позиции случайными символами
	for i := idx; i < length; i++ {
		b, err := cryptoRandInt(len(defaultCharset))
		if err != nil {
			return "", fmt.Errorf("CRED-05: generate random char: %w", err)
		}
		password[i] = defaultCharset[b.Int64()]
	}

	// Шаг 3: Fisher-Yates shuffle с crypto/rand
	for i := length - 1; i > 0; i-- {
		b, err := cryptoRandInt(i + 1)
		if err != nil {
			return "", fmt.Errorf("CRED-05: shuffle random index: %w", err)
		}
		j := int(b.Int64())
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// cryptoRandInt возвращает равномерно распределённое случайное число
// в диапазоне [0, max) с использованием crypto/rand.
func cryptoRandInt(max int) (*big.Int, error) {
	return rand.Int(rand.Reader, big.NewInt(int64(max)))
}

// ────────────────────────────────────────────────────────────────────────────
// Credential Rotation
// ────────────────────────────────────────────────────────────────────────────

// RotateCredentials выполняет полную ротацию credentials для устройства.
//
// Этапы:
//  1. Получение текущего username из CredentialManager
//  2. Генерация нового пароля через GeneratePassword
//  3. Смена пароля на устройстве через DevicePasswordChanger
//  4. Сохранение нового пароля в CredentialManager (шифрование)
//  5. Опционально: сохранение master key в Vault
//
// Если шаг 3 (смена на устройстве) не удался:
//   - Старый пароль СОХРАНЯЕТСЯ (откат не требуется)
//   - Возвращается ошибка
//
// Каждый шаг логируется в audit trail.
//
// Соответствует:
//   - IEC 62443-3-3 SR 2.2: Password management
//   - ISO 27001 A.9.2.3: Password management
//   - ISO 27001 A.12.4: Audit logging
func (r *CredentialRotator) RotateCredentials(ctx context.Context, deviceID string) error {
	traceID := trace.FromContextOrDefault(ctx)
	logger := r.logger.With("device_id", deviceID, "trace_id", traceID)

	logger.Info("CRED-05: starting credential rotation")

	// Шаг 1: получаем текущий username
	username, _, err := r.credentialMgr.Retrieve(ctx, deviceID)
	if err != nil {
		logger.Error("CRED-05: failed to retrieve current credentials", "error", err)
		return fmt.Errorf("CRED-05: retrieve credentials for %s: %w", deviceID, err)
	}

	// Шаг 2: генерируем новый пароль
	newPassword, err := r.GeneratePassword(r.config.PasswordLength)
	if err != nil {
		logger.Error("CRED-05: failed to generate new password", "error", err)
		return fmt.Errorf("CRED-05: generate password: %w", err)
	}
	logger.Debug("CRED-05: new password generated", "length", len(newPassword))

	// Шаг 3: смена пароля на устройстве
	if err := r.passwordChgr.ChangePassword(ctx, username, newPassword); err != nil {
		logger.Error("CRED-05: device rejected password change", "error", err)
		return fmt.Errorf("CRED-05: change password on device %s: %w", deviceID, err)
	}
	logger.Info("CRED-05: password changed on device")

	// Шаг 4: сохраняем новый пароль в CredentialManager
	if err := r.credentialMgr.Rotate(ctx, deviceID, username, newPassword); err != nil {
		logger.Error("CRED-05: failed to save rotated credentials", "error", err)
		// Пароль на устройстве уже изменён, но не сохранён в БД.
		// Критическая ситуация — устройство заблокировано для доступа.
		// Нужно повторить операцию сохранения.
		return fmt.Errorf("CRED-05: save rotated credentials for %s: %w", deviceID, err)
	}
	logger.Info("CRED-05: credentials rotated and saved")

	// Шаг 5: опционально сохраняем master key в Vault
	if r.vaultClient != nil {
		// Генерируем новый master key для устройства
		masterKey := make([]byte, 32) // 256-bit key (СТБ 34.101.30)
		if _, err := rand.Read(masterKey); err != nil {
			logger.Warn("CRED-05: failed to generate master key", "error", err)
			// Не фатально — credentials уже сохранены
		} else {
			if err := r.vaultClient.StoreMasterKey(ctx, deviceID, masterKey); err != nil {
				logger.Warn("CRED-05: failed to store master key in vault", "error", err)
				// Не фатально — ключ шифрования уже в памяти
			} else {
				logger.Info("CRED-05: master key stored in vault")
			}
		}
	}

	logger.Info("CRED-05: credential rotation completed successfully")
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Scheduled Rotation
// ────────────────────────────────────────────────────────────────────────────

// ScheduleRotation запускает периодическую ротацию credentials для всех устройств.
// Работает в фоновой горутине.
//
// На каждой итерации:
//  1. Получает список устройств с истекающими credentials (через external callback)
//  2. Для каждого устройства выполняет RotateCredentials
//  3. Ждёт следующий интервал
//
// Для graceful shutdown: отмените ctx или вызовите Stop().
//
// Соответствует:
//   - IEC 62443-3-3 SR 2.2: Регулярная смена паролей
//   - ISO 27001 A.9.2.3: Password management policy
func (r *CredentialRotator) ScheduleRotation(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = r.config.RotationInterval
	}

	r.logger.Info("CRED-05: starting scheduled credential rotation",
		"interval", interval,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("CRED-05: scheduled rotation stopped")
			return
		case <-r.stopCh:
			r.logger.Info("CRED-05: scheduled rotation stopped via stop channel")
			return
		case <-ticker.C:
			r.processRotationCycle(ctx)
		}
	}
}

// processRotationCycle выполняет один цикл ротации.
// В текущей реализации — заглушка для списка устройств.
// В production: получает список устройств через DeviceRepository.
func (r *CredentialRotator) processRotationCycle(ctx context.Context) {
	traceID := trace.NewID()
	ctx = trace.WithContext(ctx, traceID)
	logger := r.logger.With("trace_id", traceID)

	logger.Info("CRED-05: rotation cycle started")

	// TODO: получить список устройств для ротации через DeviceRepository
	// Пример:
	//   devices, err := r.deviceRepo.ListDevicesWithExpiringCredentials(ctx, r.config.RotationInterval)
	//   for _, dev := range devices {
	//       if err := r.RotateCredentials(ctx, dev.ID); err != nil {
	//           logger.Error("CRED-05: rotation failed", "device_id", dev.ID, "error", err)
	//       }
	//   }

	logger.Debug("CRED-05: rotation cycle completed (no devices yet)")
}

// Stop останавливает фоновую ротацию (ScheduleRotation).
func (r *CredentialRotator) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-r.stopCh:
		// уже остановлен
	default:
		close(r.stopCh)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Expiry Notifications
// ────────────────────────────────────────────────────────────────────────────

// NotifyExpiry проверяет credentials на приближающееся истечение
// и возвращает список устройств, чьи credentials истекают в течение threshold.
//
// В текущей реализации возвращает пустой список (заглушка).
// В production: запрос к БД через DeviceRepository с фильтром expires_at.
//
// Соответствует:
//   - ISO 27001 A.9.2.3: Password management notification
//   - IEC 62443-3-3 SR 2.2: Password expiry notification
func (r *CredentialRotator) NotifyExpiry(ctx context.Context, threshold time.Duration) ([]ExpiringCredential, error) {
	if threshold <= 0 {
		threshold = r.config.ExpiryThreshold
	}

	traceID := trace.FromContextOrDefault(ctx)
	logger := r.logger.With("trace_id", traceID, "threshold", threshold)

	logger.Debug("CRED-05: checking expiring credentials")

	// TODO: запрос к БД
	//   SELECT device_id, username, expires_at
	//   FROM device_credentials
	//   WHERE expires_at IS NOT NULL
	//     AND expires_at > NOW()
	//     AND expires_at <= NOW() + $1::interval
	//   ORDER BY expires_at ASC
	//
	// Пример:
	//   rows, err := r.db.Query(ctx, query, threshold)
	//   if err != nil {
	//       return nil, fmt.Errorf("query expiring credentials: %w", err)
	//   }
	//   defer rows.Close()
	//
	//   var result []ExpiringCredential
	//   for rows.Next() {
	//       var ec ExpiringCredential
	//       if err := rows.Scan(&ec.DeviceID, &ec.Username, &ec.ExpiresAt); err != nil {
	//           return nil, fmt.Errorf("scan expiring credential: %w", err)
	//       }
	//       result = append(result, ec)
	//   }
	//
	//   if len(result) == 0 {
	//       logger.Info("CRED-05: no expiring credentials found")
	//   } else {
	//       logger.Info("CRED-05: expiring credentials found", "count", len(result))
	//   }
	//
	//   return result, nil

	logger.Debug("CRED-05: expiry check completed (no devices yet)")
	return nil, nil
}

// ExpiringCredential представляет запись об истекающем credentials.
type ExpiringCredential struct {
	DeviceID  string    `json:"device_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
	DaysLeft  int       `json:"days_left"`
}

// RotatorStatus возвращает статус ротатора для health checks.
type RotatorStatus struct {
	Running          bool   `json:"running"`
	RotationInterval string `json:"rotation_interval"`
	ExpiryThreshold  string `json:"expiry_threshold"`
	PasswordLength   int    `json:"password_length"`
	VaultEnabled     bool   `json:"vault_enabled"`
}

// Status возвращает текущий статус ротатора.
func (r *CredentialRotator) Status() RotatorStatus {
	return RotatorStatus{
		Running:          true,
		RotationInterval: r.config.RotationInterval.String(),
		ExpiryThreshold:  r.config.ExpiryThreshold.String(),
		PasswordLength:   r.config.PasswordLength,
		VaultEnabled:     r.vaultClient != nil,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Audit Trail Helper
// ────────────────────────────────────────────────────────────────────────────

// rotationAuditEntry формирует строку для audit log.
func rotationAuditEntry(deviceID, username, traceID string) string {
	return fmt.Sprintf("CRED-05:ROTATION:%s:%s:%s:%d", deviceID, username, traceID, time.Now().UnixNano())
}

// getOrCreateTraceID возвращает traceID из контекста или генерирует новый.
func getOrCreateTraceID(ctx context.Context, logger *slog.Logger) string {
	id := trace.FromContext(ctx)
	if id == "" {
		id = trace.NewID()
		logger.Debug("CRED-05: generated new trace ID for background operation", "trace_id", id)
	}
	return id
}
