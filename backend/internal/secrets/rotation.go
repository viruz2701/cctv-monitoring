// Package secrets — Automated Secrets Rotation (P1-SEC.4).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SEC.4: Secrets Rotation
//
// Automated rotation для JWT secrets, API keys, HMAC keys.
// Grace period: old + new secrets valid одновременно.
//
// Compliance:
//   - ISO 27001 A.9.2.1 (Key management — periodic rotation)
//   - ISO 27001 A.12.4.1 (Event logging — rotation audit)
//   - OWASP ASVS V2.1 (Secret verification — key rotation)
//   - Приказ ОАЦ №66 п. 7.18.1 (Key management)
//
// ═══════════════════════════════════════════════════════════════════════════
package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// DefaultRotationInterval — интервал ротации по умолчанию (90 дней).
	DefaultRotationInterval = 90 * 24 * time.Hour

	// DefaultGracePeriod — период, когда старый и новый ключи валидны (24 часа).
	DefaultGracePeriod = 24 * time.Hour

	// DefaultKeyLength — длина генерируемого ключа (32 байта = 256 бит).
	DefaultKeyLength = 32

	// RotationAuditEvent — имя события для audit log.
	RotationAuditEvent = "SECRET_ROTATION"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// SecretType — тип секрета для ротации.
type SecretType string

const (
	SecretJWT    SecretType = "jwt"
	SecretHMAC   SecretType = "hmac"
	SecretAPIKey SecretType = "api_key"
)

// SecretEntry — запись секрета с версиями.
type SecretEntry struct {
	Current   string    `json:"current"`
	Previous  string    `json:"previous,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
	Version   int       `json:"version"`
}

// RotationConfig — конфигурация ротации для секрета.
type RotationConfig struct {
	SecretType       SecretType
	RotationInterval time.Duration
	GracePeriod      time.Duration
	KeyLength        int
}

// DefaultRotationConfig — конфигурации по умолчанию.
var DefaultRotationConfig = map[SecretType]RotationConfig{
	SecretJWT:  {SecretJWT, DefaultRotationInterval, DefaultGracePeriod, DefaultKeyLength},
	SecretHMAC: {SecretHMAC, DefaultRotationInterval, DefaultGracePeriod, DefaultKeyLength},
}

// RotationEvent — событие ротации для audit log.
type RotationEvent struct {
	Timestamp   time.Time  `json:"timestamp"`
	SecretType  SecretType `json:"secret_type"`
	OldVersion  int        `json:"old_version"`
	NewVersion  int        `json:"new_version"`
	Status      string     `json:"status"` // "success", "failure"
	Error       string     `json:"error,omitempty"`
	TriggeredBy string     `json:"triggered_by"` // "scheduler", "manual"
}

// ────────────────────────────────────────────────────────────────────────────
// Rotation Manager
// ────────────────────────────────────────────────────────────────────────────

// AuditLogger — интерфейс для audit log.
type AuditLogger interface {
	Log(event RotationEvent)
}

// SecretStore — интерфейс для хранения секретов.
type SecretStore interface {
	Get(secretType SecretType) (*SecretEntry, error)
	Set(secretType SecretType, entry *SecretEntry) error
}

// RotationManager — менеджер автоматической ротации секретов.
type RotationManager struct {
	mu      sync.RWMutex
	configs map[SecretType]RotationConfig
	store   SecretStore
	audit   AuditLogger
	log     *slog.Logger
	running atomic.Bool
	stopCh  chan struct{}
	secrets map[SecretType]*SecretEntry
}

// NewRotationManager создаёт менеджер ротации.
func NewRotationManager(store SecretStore, audit AuditLogger, log *slog.Logger) *RotationManager {
	if log == nil {
		log = slog.Default()
	}

	rm := &RotationManager{
		configs: DefaultRotationConfig,
		store:   store,
		audit:   audit,
		log:     log.With("component", "secret-rotation"),
		stopCh:  make(chan struct{}),
		secrets: make(map[SecretType]*SecretEntry),
	}

	// Инициализируем начальные секреты
	for st, cfg := range rm.configs {
		entry, err := rm.store.Get(st)
		if err != nil || entry == nil {
			key, _ := generateKey(cfg.KeyLength)
			entry = &SecretEntry{
				Current:   key,
				CreatedAt: time.Now(),
				Version:   1,
			}
			rm.store.Set(st, entry)
		}
		rm.secrets[st] = entry
	}

	return rm
}

// Start запускает фоновую ротацию.
func (rm *RotationManager) Start() {
	if !rm.running.CompareAndSwap(false, true) {
		return
	}
	go rm.runLoop()
	rm.log.Info("secret rotation manager started",
		"jwt_interval", rm.configs[SecretJWT].RotationInterval,
		"hmac_interval", rm.configs[SecretHMAC].RotationInterval,
	)
}

// Stop останавливает фоновую ротацию.
func (rm *RotationManager) Stop() {
	if rm.running.CompareAndSwap(true, false) {
		close(rm.stopCh)
		rm.log.Info("secret rotation manager stopped")
	}
}

// runLoop — основной цикл ротации.
func (rm *RotationManager) runLoop() {
	ticker := time.NewTicker(1 * time.Hour) // проверка каждый час
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.checkAndRotate()
		}
	}
}

// checkAndRotate проверяет и выполняет ротацию просроченных секретов.
func (rm *RotationManager) checkAndRotate() {
	for st := range rm.configs {
		rm.mu.RLock()
		entry := rm.secrets[st]
		cfg := rm.configs[st]
		rm.mu.RUnlock()

		if entry == nil {
			continue
		}

		if time.Since(entry.CreatedAt) >= cfg.RotationInterval {
			rm.Rotate(st, "scheduler")
		}
	}
}

// Rotate выполняет ротацию секрета.
func (rm *RotationManager) Rotate(secretType SecretType, triggeredBy string) error {
	cfg, ok := rm.configs[secretType]
	if !ok {
		return fmt.Errorf("secret rotation: unknown type %s", secretType)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	oldEntry := rm.secrets[secretType]
	if oldEntry == nil {
		return fmt.Errorf("secret rotation: no existing entry for %s", secretType)
	}

	// Генерируем новый ключ
	newKey, err := generateKey(cfg.KeyLength)
	if err != nil {
		event := RotationEvent{
			Timestamp:   time.Now(),
			SecretType:  secretType,
			OldVersion:  oldEntry.Version,
			Status:      "failure",
			Error:       err.Error(),
			TriggeredBy: triggeredBy,
		}
		if rm.audit != nil {
			rm.audit.Log(event)
		}
		rm.log.Error("secret rotation failed", "type", secretType, "error", err)
		return fmt.Errorf("generate key: %w", err)
	}

	// Создаём новый entry с previous = старым current
	newEntry := &SecretEntry{
		Current:   newKey,
		Previous:  oldEntry.Current,
		CreatedAt: time.Now(),
		RotatedAt: time.Now(),
		Version:   oldEntry.Version + 1,
	}

	rm.secrets[secretType] = newEntry
	if err := rm.store.Set(secretType, newEntry); err != nil {
		rm.log.Error("failed to persist rotated secret", "type", secretType, "error", err)
	}

	// Audit log
	event := RotationEvent{
		Timestamp:   time.Now(),
		SecretType:  secretType,
		OldVersion:  oldEntry.Version,
		NewVersion:  newEntry.Version,
		Status:      "success",
		TriggeredBy: triggeredBy,
	}
	if rm.audit != nil {
		rm.audit.Log(event)
	}

	rm.log.Info("secret rotated",
		"type", secretType,
		"old_version", oldEntry.Version,
		"new_version", newEntry.Version,
		"triggered_by", triggeredBy,
	)

	return nil
}

// GetCurrentSecret возвращает текущий секрет.
func (rm *RotationManager) GetCurrentSecret(secretType SecretType) (string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	entry, ok := rm.secrets[secretType]
	if !ok || entry == nil {
		return "", fmt.Errorf("secret %s not found", secretType)
	}
	return entry.Current, nil
}

// GetValidSecrets возвращает все валидные секреты (current + previous в grace period).
func (rm *RotationManager) GetValidSecrets(secretType SecretType) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	entry, ok := rm.secrets[secretType]
	if !ok || entry == nil {
		return nil, fmt.Errorf("secret %s not found", secretType)
	}

	secrets := []string{entry.Current}
	cfg := rm.configs[secretType]

	// Grace period: старый ключ ещё валиден
	if entry.Previous != "" && time.Since(entry.RotatedAt) < cfg.GracePeriod {
		secrets = append(secrets, entry.Previous)
	}

	return secrets, nil
}

// ────────────────────────────────────────────────────────────────────────────
// In-memory store (dev default)
// ────────────────────────────────────────────────────────────────────────────

// MemoryStore — in-memory реализация SecretStore.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[SecretType]*SecretEntry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[SecretType]*SecretEntry)}
}

func (s *MemoryStore) Get(secretType SecretType) (*SecretEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.data[secretType]
	if !ok {
		return nil, fmt.Errorf("secret %s not found", secretType)
	}
	return entry, nil
}

func (s *MemoryStore) Set(secretType SecretType, entry *SecretEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[secretType] = entry
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

func generateKey(length int) (string, error) {
	if length <= 0 {
		length = DefaultKeyLength
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// FormatRotationInterval возвращает человекочитаемый интервал.
func FormatRotationInterval(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days >= 365 {
		return fmt.Sprintf("%d years", days/365)
	}
	if days >= 30 {
		return fmt.Sprintf("%d months", days/30)
	}
	return fmt.Sprintf("%d days", days)
}
