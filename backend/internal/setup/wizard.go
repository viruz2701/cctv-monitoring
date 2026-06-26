// Package setup — On-Premise Setup Wizard (P0-CE.4).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.4: Setup Wizard (On-Premise)
//
// 7-step wizard for initial on-premise configuration:
//  1. Region Selection — выбор региона + compliance checklist
//  2. Crypto Confirmation — подтверждение криптопараметров
//  3. Storage Configuration — выбор хранилища (локальное/S3)
//  4. Admin Account — создание первого администратора
//  5. Network Configuration — настройка сети (TLS, порты)
//  6. Notifications — настройка уведомлений (Telegram/Email/SMS)
//  7. Review & Complete — финальное подтверждение + compliance report
//
// После завершения: регион блокируется (нельзя сменить без миграции).
//
// Compliance:
//   - IEC 62443-3-3 SR 5.1 (Zone-based configuration)
//   - ISO 27001 A.8.1 (Asset management — initial setup)
//   - Приказ ОАЦ № 66 п. 7.18 (Initial device identification)
//   - GDPR Art. 25 (Data protection by design — выбор региона)
//
// ═══════════════════════════════════════════════════════════════════════════
package setup

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// Wizard steps
// ────────────────────────────────────────────────────────────────────────────

// WizardStep представляет шаг мастера настройки.
type WizardStep int

const (
	StepRegion        WizardStep = 1 // Выбор региона
	StepCrypto        WizardStep = 2 // Подтверждение криптографии
	StepStorage       WizardStep = 3 // Настройка хранилища
	StepAdmin         WizardStep = 4 // Создание администратора
	StepNetwork       WizardStep = 5 // Настройка сети
	StepNotifications WizardStep = 6 // Настройка уведомлений
	StepReview        WizardStep = 7 // Финальное подтверждение
	StepCompleted     WizardStep = 0 // Мастер завершён
)

// StepInfo содержит информацию о шаге.
type StepInfo struct {
	Step        WizardStep `json:"step"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Required    bool       `json:"required"`
}

// AllSteps возвращает все шаги мастера.
func AllSteps() []StepInfo {
	return []StepInfo{
		{Step: StepRegion, Name: "Region", Description: "Select deployment region and compliance profile", Required: true},
		{Step: StepCrypto, Name: "Cryptography", Description: "Confirm cryptographic parameters", Required: true},
		{Step: StepStorage, Name: "Storage", Description: "Configure data storage (local/S3)", Required: true},
		{Step: StepAdmin, Name: "Admin Account", Description: "Create initial administrator account", Required: true},
		{Step: StepNetwork, Name: "Network", Description: "Configure TLS, ports, and network settings", Required: false},
		{Step: StepNotifications, Name: "Notifications", Description: "Configure Telegram, Email, SMS", Required: false},
		{Step: StepReview, Name: "Review & Complete", Description: "Review configuration and complete setup", Required: true},
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Wizard configuration
// ────────────────────────────────────────────────────────────────────────────

// SetupConfig хранит конфигурацию, собранную мастером.
type SetupConfig struct {
	// Region — выбранный регион (BY, EU, INTL).
	Region string `json:"region"`

	// ComplianceProfileName — название выбранного compliance профиля.
	ComplianceProfileName string `json:"compliance_profile_name"`

	// CryptoConfirmed — подтверждены ли криптопараметры.
	CryptoConfirmed bool `json:"crypto_confirmed"`

	// StorageType — тип хранилища ("local", "s3").
	StorageType string `json:"storage_type"`

	// S3Endpoint — endpoint S3 (если StorageType="s3").
	S3Endpoint string `json:"s3_endpoint,omitempty"`

	// S3Bucket — bucket S3.
	S3Bucket string `json:"s3_bucket,omitempty"`

	// S3Region — регион S3.
	S3Region string `json:"s3_region,omitempty"`

	// AdminUsername — имя администратора.
	AdminUsername string `json:"admin_username"`

	// AdminEmail — email администратора.
	AdminEmail string `json:"admin_email"`

	// AdminSignature — цифровая подпись администратора (для КИИ регионов).
	AdminSignature string `json:"admin_signature,omitempty"`

	// TLSCertPath — путь к TLS сертификату.
	TLSCertPath string `json:"tls_cert_path,omitempty"`

	// TLSKeyPath — путь к TLS ключу.
	TLSKeyPath string `json:"tls_key_path,omitempty"`

	// APIPort — порт API сервера.
	APIPort int `json:"api_port"`

	// TelegramToken — токен Telegram бота.
	TelegramToken string `json:"telegram_token,omitempty"`

	// SMTPHost — SMTP сервер для email.
	SMTPHost string `json:"smtp_host,omitempty"`

	// SMTPPort — порт SMTP.
	SMTPPort int `json:"smtp_port,omitempty"`

	// SMTPUsername — пользователь SMTP.
	SMTPUsername string `json:"smtp_username,omitempty"`

	// RegionLocked — флаг блокировки региона (immutable после завершения).
	RegionLocked bool `json:"region_locked"`

	// CompletedAt — время завершения мастера.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ComplianceReport — ссылка на сгенерированный compliance report.
	ComplianceReport string `json:"compliance_report,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// RegionConfig
// ────────────────────────────────────────────────────────────────────────────

// RegionConfig содержит compliance информацию о регионе для мастера.
type RegionConfig struct {
	Region      string     `json:"region"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Compliance  []string   `json:"compliance"`
	CryptoInfo  CryptoInfo `json:"crypto_info"`
	LegalNotice string     `json:"legal_notice"`
}

// CryptoInfo содержит информацию о криптографии для отображения.
type CryptoInfo struct {
	Encryption string `json:"encryption"`
	Hash       string `json:"hash"`
	Signature  string `json:"signature"`
	KeySize    int    `json:"key_size"`
}

// AvailableRegions возвращает доступные регионы для мастера.
func AvailableRegions() []RegionConfig {
	return []RegionConfig{
		{
			Region:      compliance.RegionBY,
			Name:        "Республика Беларусь (КИИ)",
			Description: "Соответствие СТБ 34.101.27, СТБ 34.101.30, Приказ ОАЦ №66",
			Compliance:  []string{"СТБ 34.101.27 (Защита информации)", "СТБ 34.101.30 (Криптография)", "Приказ ОАЦ №66 п. 7.18", "IEC 62443 SL-3", "ISO 27001"},
			CryptoInfo: CryptoInfo{
				Encryption: "belt-GCM (СТБ 34.101.31)",
				Hash:       "bash-256 (СТБ 34.101.77)",
				Signature:  "bign-curve256v1 (СТБ 34.101.45)",
				KeySize:    256,
			},
			LegalNotice: "ВНИМАНИЕ: Выбор региона РБ активирует режим КИИ. " +
				"Смена региона после первого логина невозможна без полной миграции данных. " +
				"Требуется цифровая подпись администратора.",
		},
		{
			Region:      compliance.RegionEU,
			Name:        "European Union (GDPR)",
			Description: "Соответствие GDPR, NIS2, eIDAS, ISO 27001",
			Compliance:  []string{"GDPR (General Data Protection Regulation)", "NIS2 Directive", "eIDAS", "ISO 27001", "IEC 62443"},
			CryptoInfo: CryptoInfo{
				Encryption: "AES-256-GCM (NIST SP 800-38D)",
				Hash:       "SHA-256",
				Signature:  "ECDSA P-256 (ES256)",
				KeySize:    256,
			},
			LegalNotice: "GDPR Art. 44-49 applies to cross-border data transfers. " +
				"Standard Contractual Clauses (SCC) may be required for non-EU data access.",
		},
		{
			Region:      compliance.RegionINTL,
			Name:        "International (ISO 27001)",
			Description: "Базовое соответствие ISO 27001, ISO 27019, IEC 62443, OWASP ASVS L3",
			Compliance:  []string{"ISO 27001:2022", "ISO 27019 (ICS Security)", "IEC 62443-3-3", "OWASP ASVS Level 3"},
			CryptoInfo: CryptoInfo{
				Encryption: "AES-256-GCM (NIST SP 800-38D)",
				Hash:       "SHA-256",
				Signature:  "ECDSA P-256 (ES256)",
				KeySize:    256,
			},
			LegalNotice: "International profile — suitable for organizations without specific regional compliance requirements.",
		},
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Wizard state machine
// ────────────────────────────────────────────────────────────────────────────

// SetupWizard управляет процессом первоначальной настройки.
type SetupWizard struct {
	mu        sync.RWMutex
	config    *SetupConfig
	step      WizardStep
	started   bool
	completed bool
	registry  *compliance.ProfileRegistry
	logger    *slog.Logger

	// setupCompleteFn — callback при завершении настройки.
	setupCompleteFn func(config *SetupConfig) error
}

// NewSetupWizard создаёт новый мастер настройки.
func NewSetupWizard(registry *compliance.ProfileRegistry, opts ...WizardOption) *SetupWizard {
	w := &SetupWizard{
		config:   &SetupConfig{APIPort: 8080},
		step:     StepRegion,
		registry: registry,
		logger:   slog.Default().With("component", "setup.wizard"),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// WizardOption — функциональная опция для SetupWizard.
type WizardOption func(*SetupWizard)

// WithSetupCompleteHandler устанавливает callback при завершении настройки.
func WithSetupCompleteHandler(fn func(config *SetupConfig) error) WizardOption {
	return func(w *SetupWizard) {
		w.setupCompleteFn = fn
	}
}

// WithLogger устанавливает логгер.
func WithLogger(logger *slog.Logger) WizardOption {
	return func(w *SetupWizard) {
		w.logger = logger
	}
}

// Start начинает процесс настройки.
func (w *SetupWizard) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.completed {
		return ErrSetupAlreadyCompleted
	}
	if w.started {
		return ErrSetupAlreadyStarted
	}

	w.started = true
	w.step = StepRegion
	w.logger.Info("setup wizard started")
	return nil
}

// CurrentStep возвращает текущий шаг.
func (w *SetupWizard) CurrentStep() WizardStep {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.step
}

// IsCompleted возвращает true если мастер завершён.
func (w *SetupWizard) IsCompleted() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.completed
}

// IsStarted возвращает true если мастер запущен.
func (w *SetupWizard) IsStarted() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.started
}

// Config возвращает текущую конфигурацию.
func (w *SetupWizard) Config() *SetupConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()
	cfg := *w.config
	return &cfg
}

// ────────────────────────────────────────────────────────────────────────────
// Step handlers
// ────────────────────────────────────────────────────────────────────────────

// SetRegion устанавливает регион и переходит к шагу криптографии.
func (w *SetupWizard) SetRegion(region string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepRegion {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepRegion, w.step)
	}

	// Validate region
	if !w.registry.IsRegistered(region) {
		return fmt.Errorf("%w: %s", compliance.ErrProfileNotFound, region)
	}

	profile, err := w.registry.Get(region)
	if err != nil {
		return fmt.Errorf("get profile for region %s: %w", region, err)
	}

	w.config.Region = region
	w.config.ComplianceProfileName = profile.Name()
	w.step = StepCrypto

	w.logger.Info("region selected", "region", region, "profile", profile.Name())
	return nil
}

// ConfirmCrypto подтверждает криптопараметры и переходит к шагу хранилища.
func (w *SetupWizard) ConfirmCrypto(confirmed bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepCrypto {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepCrypto, w.step)
	}
	if !confirmed {
		return ErrCryptoNotConfirmed
	}

	w.config.CryptoConfirmed = true
	w.step = StepStorage
	w.logger.Info("crypto confirmed")
	return nil
}

// SetStorage настраивает хранилище и переходит к шагу администратора.
func (w *SetupWizard) SetStorage(storageType, s3Endpoint, s3Bucket, s3Region string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepStorage {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepStorage, w.step)
	}

	if storageType != "local" && storageType != "s3" {
		return fmt.Errorf("%w: invalid storage type %s", ErrInvalidConfig, storageType)
	}

	w.config.StorageType = storageType
	if storageType == "s3" {
		if s3Endpoint == "" || s3Bucket == "" {
			return fmt.Errorf("%w: S3 endpoint and bucket required", ErrInvalidConfig)
		}
		w.config.S3Endpoint = s3Endpoint
		w.config.S3Bucket = s3Bucket
		w.config.S3Region = s3Region
	}

	w.step = StepAdmin
	w.logger.Info("storage configured", "type", storageType)
	return nil
}

// SetAdmin создаёт учётную запись администратора.
// Для КИИ регионов (BY) требуется цифровая подпись.
func (w *SetupWizard) SetAdmin(username, email, signature string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepAdmin {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepAdmin, w.step)
	}

	if username == "" || email == "" {
		return fmt.Errorf("%w: username and email required", ErrInvalidConfig)
	}

	// Для КИИ (BY) требуется цифровая подпись
	if w.config.Region == compliance.RegionBY && signature == "" {
		return ErrSignatureRequired
	}

	w.config.AdminUsername = username
	w.config.AdminEmail = email
	w.config.AdminSignature = signature

	w.step = StepNetwork
	w.logger.Info("admin configured", "username", username)
	return nil
}

// SetNetwork настраивает сетевые параметры.
func (w *SetupWizard) SetNetwork(apiPort int, tlsCert, tlsKey string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepNetwork {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepNetwork, w.step)
	}

	if apiPort <= 0 || apiPort > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidConfig, apiPort)
	}

	w.config.APIPort = apiPort
	w.config.TLSCertPath = tlsCert
	w.config.TLSKeyPath = tlsKey

	w.step = StepNotifications
	w.logger.Info("network configured", "port", apiPort)
	return nil
}

// SetNotifications настраивает уведомления.
func (w *SetupWizard) SetNotifications(telegramToken, smtpHost string, smtpPort int, smtpUsername string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepNotifications {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepNotifications, w.step)
	}

	w.config.TelegramToken = telegramToken
	w.config.SMTPHost = smtpHost
	w.config.SMTPPort = smtpPort
	w.config.SMTPUsername = smtpUsername

	w.step = StepReview
	w.logger.Info("notifications configured")
	return nil
}

// Complete завершает мастер настройки. После завершения регион блокируется.
func (w *SetupWizard) Complete() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.requireStarted(); err != nil {
		return err
	}
	if w.step != StepReview {
		return fmt.Errorf("%w: expected step %d, got %d", ErrInvalidStep, StepReview, w.step)
	}

	if err := w.validateConfig(); err != nil {
		return fmt.Errorf("config validation: %w", err)
	}

	w.config.RegionLocked = true
	now := time.Now().UTC()
	w.config.CompletedAt = &now

	// Callback
	if w.setupCompleteFn != nil {
		if err := w.setupCompleteFn(w.config); err != nil {
			return fmt.Errorf("setup complete handler: %w", err)
		}
	}

	w.completed = true
	w.step = StepCompleted

	w.logger.Info("setup wizard completed",
		"region", w.config.Region,
		"admin", w.config.AdminUsername,
	)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

func (w *SetupWizard) requireStarted() error {
	if w.completed {
		return ErrSetupAlreadyCompleted
	}
	if !w.started {
		return ErrSetupNotStarted
	}
	return nil
}

func (w *SetupWizard) validateConfig() error {
	if w.config.Region == "" {
		return fmt.Errorf("%w: region not selected", ErrInvalidConfig)
	}
	if !w.config.CryptoConfirmed {
		return fmt.Errorf("%w: crypto not confirmed", ErrInvalidConfig)
	}
	if w.config.StorageType == "" {
		return fmt.Errorf("%w: storage not configured", ErrInvalidConfig)
	}
	if w.config.AdminUsername == "" || w.config.AdminEmail == "" {
		return fmt.Errorf("%w: admin not configured", ErrInvalidConfig)
	}
	if w.config.Region == compliance.RegionBY && w.config.AdminSignature == "" {
		return fmt.Errorf("%w: digital signature required for КИИ", ErrInvalidConfig)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Errors
// ═══════════════════════════════════════════════════════════════════════════

var (
	ErrSetupAlreadyCompleted = fmt.Errorf("setup: already completed")
	ErrSetupAlreadyStarted   = fmt.Errorf("setup: already started")
	ErrSetupNotStarted       = fmt.Errorf("setup: not started")
	ErrInvalidStep           = fmt.Errorf("setup: invalid step")
	ErrInvalidConfig         = fmt.Errorf("setup: invalid configuration")
	ErrCryptoNotConfirmed    = fmt.Errorf("setup: crypto not confirmed")
	ErrSignatureRequired     = fmt.Errorf("setup: digital signature required for КИИ region")
)
