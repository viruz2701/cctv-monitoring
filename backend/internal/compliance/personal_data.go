// Package compliance — 152-ФЗ Personal Data Features (P2-RU.2).
//
// Реализует:
//   - Consent management (сбор, хранение, отзыв согласий)
//   - DSAR (Data Subject Access Request) workflow
//   - Automated data inventory (что хранится, где, зачем)
//   - Роскомнадзор reporting templates
//   - Data anonymization для analytics
//
// Compliance:
//   - 152-ФЗ "О персональных данных" ст. 9 (согласие), ст. 14 (доступ), ст. 21 (блокировка)
//   - ISO 27001 A.8.2 (Information classification — категории ПД)
//   - ISO 27019 PCC.A.8 (ICS data classification)
//   - СТБ 34.101.27 п. 6.2 (Политики безопасности ПД)
//   - OWASP ASVS V8 (Data Protection — sensitivity labeling)
package compliance

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// 152-ФЗ: Consent Types & Statuses
// ═══════════════════════════════════════════════════════════════════════

// ConsentPurpose — цель обработки ПД (ст. 9 152-ФЗ).
type ConsentPurpose string

const (
	ConsentPurposeMonitoring ConsentPurpose = "video_monitoring"      // Видеонаблюдение
	ConsentPurposeAnalytics  ConsentPurpose = "analytics"             // Аналитика поведения
	ConsentPurposeAccess     ConsentPurpose = "access_control"        // Контроль доступа
	ConsentPurposeCompliance ConsentPurpose = "regulatory_compliance" // Соответствие регуляторам
	ConsentPurposeEmergency  ConsentPurpose = "emergency_response"    // Реагирование на ЧС
	ConsentPurposeRetention  ConsentPurpose = "data_retention"        // Хранение архива
	ConsentPurposeThirdParty ConsentPurpose = "third_party_sharing"   // Передача третьим лицам
)

// ConsentStatus — статус согласия на обработку ПД.
type ConsentStatus string

const (
	ConsentStatusGranted ConsentStatus = "granted" // Согласие получено
	ConsentStatusRevoked ConsentStatus = "revoked" // Согласие отозвано
	ConsentStatusExpired ConsentStatus = "expired" // Истек срок действия
	ConsentStatusPending ConsentStatus = "pending" // Ожидает подтверждения
)

// ConsentRecord представляет запись о согласии на обработку ПД.
type ConsentRecord struct {
	ID           string         `json:"id"`
	SubjectID    string         `json:"subject_id"`           // ID субъекта ПД
	SubjectName  string         `json:"subject_name"`         // ФИО субъекта
	Purpose      ConsentPurpose `json:"purpose"`              // Цель обработки
	Status       ConsentStatus  `json:"status"`               // Статус согласия
	GrantedAt    time.Time      `json:"granted_at"`           // Дата получения
	RevokedAt    *time.Time     `json:"revoked_at,omitempty"` // Дата отзыва
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"` // Срок действия
	Source       string         `json:"source"`               // Источник (web, mobile, paper)
	DocumentHash string         `json:"document_hash"`        // Хеш документа согласия
	Notes        string         `json:"notes,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// DSAR (Data Subject Access Request) — ст. 14 152-ФЗ
// ═══════════════════════════════════════════════════════════════════════

// DSARStatus — статус запроса субъекта ПД.
type DSARStatus string

const (
	DSARStatusNew       DSARStatus = "new"
	DSARStatusVerified  DSARStatus = "verified"  // Личность подтверждена
	DSARStatusInReview  DSARStatus = "in_review" // Проверка запроса
	DSARStatusGathering DSARStatus = "gathering" // Сбор данных
	DSARStatusFulfilled DSARStatus = "fulfilled" // Запрос выполнен
	DSARStatusRejected  DSARStatus = "rejected"  // Отклонён (ст. 14 ч. 4)
	DSARStatusExpired   DSARStatus = "expired"   // Просрочен (30 дней)
)

// DSARRequest представляет запрос субъекта ПД на доступ к данным.
type DSARRequest struct {
	ID              string     `json:"id"`
	SubjectID       string     `json:"subject_id"`
	SubjectName     string     `json:"subject_name"`
	SubjectEmail    string     `json:"subject_email"`
	SubjectPhone    string     `json:"subject_phone,omitempty"`
	RequestType     string     `json:"request_type"` // access, rectification, erasure, restriction, portability
	Description     string     `json:"description"`  // Описание запроса
	Status          DSARStatus `json:"status"`
	VerificationDoc string     `json:"verification_doc,omitempty"` // Документ, удостоверяющий личность
	AssignedTo      string     `json:"assigned_to,omitempty"`      // Ответственный
	ResponseData    string     `json:"response_data,omitempty"`    // JSON с ответом
	RejectionReason string     `json:"rejection_reason,omitempty"`
	DeadlineAt      time.Time  `json:"deadline_at"` // 30 дней по 152-ФЗ
	FulfilledAt     *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Data Inventory — учёт ПД (ст. 18.1 152-ФЗ)
// ═══════════════════════════════════════════════════════════════════════

// DataCategory — категория ПД.
type DataCategory string

const (
	DataCategoryBiometric   DataCategory = "biometric"     // Биометрические (видео/фото)
	DataCategoryLocation    DataCategory = "location"      // Геолокация
	DataCategoryIdentity    DataCategory = "identity"      // Персональные (ФИО, паспорт)
	DataCategoryContact     DataCategory = "contact"       // Контактные (телефон, email)
	DataCategorySchedule    DataCategory = "schedule"      // Рабочий график
	DataCategoryCredentials DataCategory = "credentials"   // Учётные данные
	DataCategoryVideo       DataCategory = "video_archive" // Видеоархив
)

// DataInventoryItem — запись об одном типе ПД в системе.
type DataInventoryItem struct {
	ID              string         `json:"id"`
	Category        DataCategory   `json:"category"`
	Description     string         `json:"description"`
	DataFields      []string       `json:"data_fields"`             // Конкретные поля
	StorageLocation string         `json:"storage_location"`        // Где хранится
	Purpose         ConsentPurpose `json:"purpose"`                 // Цель обработки
	RetentionDays   int            `json:"retention_days"`          // Срок хранения
	Anonymized      bool           `json:"anonymized"`              // Обезличено?
	Encrypted       bool           `json:"encrypted"`               // Зашифровано?
	CrossBorder     bool           `json:"cross_border"`            // Трансграничная передача
	ThirdParties    []string       `json:"third_parties,omitempty"` // Третьи лица
	LegalBasis      string         `json:"legal_basis"`             // Правовое основание
	DPIARequired    bool           `json:"dpia_required"`           // Требуется ОИВД
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Роскомнадзор Reporting Templates
// ═══════════════════════════════════════════════════════════════════════

// RoskomnadzorReport — отчёт для Роскомнадзора (уведомление об обработке ПД).
type RoskomnadzorReport struct {
	OperatorName         string           `json:"operator_name"`
	OperatorINN          string           `json:"operator_inn"`
	OperatorAddress      string           `json:"operator_address"`
	DataCategories       []DataCategory   `json:"data_categories"`
	SubjectCount         int              `json:"subject_count"`
	ProcessingPurposes   []ConsentPurpose `json:"processing_purposes"`
	CrossBorder          bool             `json:"cross_border"`
	CrossBorderCountries []string         `json:"cross_border_countries,omitempty"`
	ThirdPartyProcessors []string         `json:"third_party_processors,omitempty"`
	DataRetentionDays    int              `json:"data_retention_days"`
	SecurityMeasures     []string         `json:"security_measures"`
	DPIACompleted        bool             `json:"dpia_completed"`
	GeneratedAt          time.Time        `json:"generated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Anonymization — обезличивание ПД (ст. 3 152-ФЗ)
// ═══════════════════════════════════════════════════════════════════════

// AnonymizationMethod — метод обезличивания.
type AnonymizationMethod string

const (
	AnonMethodReduction        AnonymizationMethod = "reduction"        // Сокращение
	AnonMethodGeneralization   AnonymizationMethod = "generalization"   // Обобщение
	AnonMethodMasking          AnonymizationMethod = "masking"          // Маскирование
	AnonMethodPseudonymization AnonymizationMethod = "pseudonymization" // Псевдонимизация
	AnonMethodAggregation      AnonymizationMethod = "aggregation"      // Агрегация
	AnonMethodNoiseAddition    AnonymizationMethod = "noise_addition"   // Добавление шума
)

// AnonymizationRule — правило обезличивания для категории данных.
type AnonymizationRule struct {
	Category      DataCategory        `json:"category"`
	Method        AnonymizationMethod `json:"method"`
	Parameters    map[string]string   `json:"parameters,omitempty"` // Параметры метода
	RetentionDays int                 `json:"retention_days"`       // Хранение обезличенных данных
}

// ═══════════════════════════════════════════════════════════════════════
// PersonalDataManager — управление ПД по 152-ФЗ
// ═══════════════════════════════════════════════════════════════════════

// PersonalDataStore — интерфейс для хранения данных ПД.
type PersonalDataStore interface {
	// Consent
	SaveConsent(ctx interface{}, record *ConsentRecord) error
	GetConsent(ctx interface{}, id string) (*ConsentRecord, error)
	ListConsents(ctx interface{}, subjectID string) ([]*ConsentRecord, error)
	RevokeConsent(ctx interface{}, id string) error

	// DSAR
	SaveDSAR(ctx interface{}, request *DSARRequest) error
	GetDSAR(ctx interface{}, id string) (*DSARRequest, error)
	ListDSARs(ctx interface{}, subjectID string) ([]*DSARRequest, error)
	UpdateDSARStatus(ctx interface{}, id string, status DSARStatus, responseData, rejectionReason string) error

	// Inventory
	SaveInventoryItem(ctx interface{}, item *DataInventoryItem) error
	ListInventory(ctx interface{}) ([]*DataInventoryItem, error)
	GetInventoryItem(ctx interface{}, id string) (*DataInventoryItem, error)
}

// PersonalDataManager — бизнес-логика управления ПД по 152-ФЗ.
type PersonalDataManager struct {
	store  PersonalDataStore
	logger *slog.Logger
	mu     sync.RWMutex

	// Anonymization rules
	anonRules []AnonymizationRule

	// Inventory cache
	inventoryCache []*DataInventoryItem
	cacheUpdatedAt time.Time
}

// NewPersonalDataManager создаёт новый PersonalDataManager.
func NewPersonalDataManager(store PersonalDataStore, logger *slog.Logger) *PersonalDataManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &PersonalDataManager{
		store:  store,
		logger: logger.With("component", "compliance.personal_data"),
		anonRules: []AnonymizationRule{
			{
				Category:      DataCategoryBiometric,
				Method:        AnonMethodPseudonymization,
				Parameters:    map[string]string{"algorithm": "blur_faces"},
				RetentionDays: 365,
			},
			{
				Category:      DataCategoryLocation,
				Method:        AnonMethodGeneralization,
				Parameters:    map[string]string{"precision": "100m"},
				RetentionDays: 90,
			},
			{
				Category:      DataCategoryIdentity,
				Method:        AnonMethodMasking,
				Parameters:    map[string]string{"pattern": "***"},
				RetentionDays: 365,
			},
			{
				Category:      DataCategoryContact,
				Method:        AnonMethodMasking,
				Parameters:    map[string]string{"pattern": "***@***"},
				RetentionDays: 90,
			},
		},
	}
}

// ── Consent Management ─────────────────────────────────────────────

// GrantConsent фиксирует согласие на обработку ПД.
func (m *PersonalDataManager) GrantConsent(subjectID, subjectName string, purpose ConsentPurpose, source string, expiresInDays int) (*ConsentRecord, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("personal_data: subject_id is required")
	}
	if purpose == "" {
		return nil, fmt.Errorf("personal_data: purpose is required")
	}

	now := time.Now().UTC()
	record := &ConsentRecord{
		ID:          generateID(),
		SubjectID:   subjectID,
		SubjectName: subjectName,
		Purpose:     purpose,
		Status:      ConsentStatusGranted,
		GrantedAt:   now,
		Source:      source,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if expiresInDays > 0 {
		expires := now.AddDate(0, 0, expiresInDays)
		record.ExpiresAt = &expires
	}

	if err := m.store.SaveConsent(nil, record); err != nil {
		return nil, fmt.Errorf("personal_data: save consent: %w", err)
	}

	m.logger.Info("consent granted",
		"subject_id", subjectID,
		"purpose", purpose,
		"consent_id", record.ID,
	)

	return record, nil
}

// RevokeConsent отзывает согласие на обработку ПД.
func (m *PersonalDataManager) RevokeConsent(consentID string) error {
	if consentID == "" {
		return fmt.Errorf("personal_data: consent_id is required")
	}

	record, err := m.store.GetConsent(nil, consentID)
	if err != nil {
		return fmt.Errorf("personal_data: get consent: %w", err)
	}
	if record == nil {
		return fmt.Errorf("personal_data: consent not found: %s", consentID)
	}
	if record.Status == ConsentStatusRevoked {
		return fmt.Errorf("personal_data: consent already revoked: %s", consentID)
	}

	if err := m.store.RevokeConsent(nil, consentID); err != nil {
		return fmt.Errorf("personal_data: revoke consent: %w", err)
	}

	m.logger.Info("consent revoked",
		"consent_id", consentID,
		"subject_id", record.SubjectID,
		"purpose", record.Purpose,
	)

	return nil
}

// GetConsent возвращает запись согласия.
func (m *PersonalDataManager) GetConsent(id string) (*ConsentRecord, error) {
	return m.store.GetConsent(nil, id)
}

// ListSubjectConsents возвращает все согласия субъекта.
func (m *PersonalDataManager) ListSubjectConsents(subjectID string) ([]*ConsentRecord, error) {
	return m.store.ListConsents(nil, subjectID)
}

// ── DSAR Workflow ──────────────────────────────────────────────────

// SubmitDSAR создаёт новый DSAR-запрос.
func (m *PersonalDataManager) SubmitDSAR(subjectID, subjectName, subjectEmail, subjectPhone, requestType, description string) (*DSARRequest, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("personal_data: subject_id is required")
	}
	if requestType == "" {
		return nil, fmt.Errorf("personal_data: request_type is required")
	}

	now := time.Now().UTC()
	req := &DSARRequest{
		ID:           generateID(),
		SubjectID:    subjectID,
		SubjectName:  subjectName,
		SubjectEmail: subjectEmail,
		SubjectPhone: subjectPhone,
		RequestType:  requestType,
		Description:  description,
		Status:       DSARStatusNew,
		DeadlineAt:   now.AddDate(0, 0, 30), // 30 дней по 152-ФЗ
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := m.store.SaveDSAR(nil, req); err != nil {
		return nil, fmt.Errorf("personal_data: save DSAR: %w", err)
	}

	m.logger.Info("DSAR submitted",
		"dsar_id", req.ID,
		"subject_id", subjectID,
		"type", requestType,
		"deadline", req.DeadlineAt,
	)

	return req, nil
}

// FulfillDSAR выполняет DSAR-запрос с предоставлением данных.
func (m *PersonalDataManager) FulfillDSAR(dsarID, responseData string) error {
	if dsarID == "" {
		return fmt.Errorf("personal_data: dsar_id is required")
	}

	req, err := m.store.GetDSAR(nil, dsarID)
	if err != nil {
		return fmt.Errorf("personal_data: get DSAR: %w", err)
	}
	if req == nil {
		return fmt.Errorf("personal_data: DSAR not found: %s", dsarID)
	}

	if err := m.store.UpdateDSARStatus(nil, dsarID, DSARStatusFulfilled, responseData, ""); err != nil {
		return fmt.Errorf("personal_data: fulfill DSAR: %w", err)
	}

	m.logger.Info("DSAR fulfilled",
		"dsar_id", dsarID,
		"subject_id", req.SubjectID,
	)

	return nil
}

// RejectDSAR отклоняет DSAR-запрос.
func (m *PersonalDataManager) RejectDSAR(dsarID, reason string) error {
	if dsarID == "" {
		return fmt.Errorf("personal_data: dsar_id is required")
	}
	if reason == "" {
		return fmt.Errorf("personal_data: rejection reason is required")
	}

	return m.store.UpdateDSARStatus(nil, dsarID, DSARStatusRejected, "", reason)
}

// GetDSAR возвращает DSAR-запрос.
func (m *PersonalDataManager) GetDSAR(id string) (*DSARRequest, error) {
	return m.store.GetDSAR(nil, id)
}

// ListSubjectDSARs возвращает все DSAR-запросы субъекта.
func (m *PersonalDataManager) ListSubjectDSARs(subjectID string) ([]*DSARRequest, error) {
	return m.store.ListDSARs(nil, subjectID)
}

// ── Data Inventory ─────────────────────────────────────────────────

// RegisterInventoryItem регистрирует новый тип ПД в системе.
func (m *PersonalDataManager) RegisterInventoryItem(category DataCategory, description string, dataFields []string,
	storageLocation string, purpose ConsentPurpose, retentionDays int, legalBasis string) (*DataInventoryItem, error) {

	if category == "" {
		return nil, fmt.Errorf("personal_data: category is required")
	}
	if len(dataFields) == 0 {
		return nil, fmt.Errorf("personal_data: data_fields is required")
	}

	now := time.Now().UTC()
	item := &DataInventoryItem{
		ID:              generateID(),
		Category:        category,
		Description:     description,
		DataFields:      dataFields,
		StorageLocation: storageLocation,
		Purpose:         purpose,
		RetentionDays:   retentionDays,
		Anonymized:      false,
		Encrypted:       true, // По умолчанию шифруем
		LegalBasis:      legalBasis,
		DPIARequired:    purpose == ConsentPurposeMonitoring || category == DataCategoryBiometric,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.SaveInventoryItem(nil, item); err != nil {
		return nil, fmt.Errorf("personal_data: save inventory item: %w", err)
	}

	// Инвалидируем кэш
	m.mu.Lock()
	m.inventoryCache = nil
	m.mu.Unlock()

	m.logger.Info("inventory item registered",
		"item_id", item.ID,
		"category", category,
		"fields", dataFields,
	)

	return item, nil
}

// GetInventory возвращает полный реестр ПД.
func (m *PersonalDataManager) GetInventory() ([]*DataInventoryItem, error) {
	m.mu.RLock()
	cached := m.inventoryCache
	cacheTime := m.cacheUpdatedAt
	m.mu.RUnlock()

	// Кэш на 5 минут
	if cached != nil && time.Since(cacheTime) < 5*time.Minute {
		return cached, nil
	}

	items, err := m.store.ListInventory(nil)
	if err != nil {
		return nil, fmt.Errorf("personal_data: list inventory: %w", err)
	}

	m.mu.Lock()
	m.inventoryCache = items
	m.cacheUpdatedAt = time.Now()
	m.mu.Unlock()

	return items, nil
}

// ── Anonymization ──────────────────────────────────────────────────

// GetAnonymizationRules возвращает правила обезличивания.
func (m *PersonalDataManager) GetAnonymizationRules() []AnonymizationRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rules := make([]AnonymizationRule, len(m.anonRules))
	copy(rules, m.anonRules)
	return rules
}

// AnonymizeData обезличивает данные согласно правилам.
func (m *PersonalDataManager) AnonymizeData(items []*DataInventoryItem) []*DataInventoryItem {
	anonymized := make([]*DataInventoryItem, 0, len(items))

	for _, item := range items {
		rule := m.findAnonymizationRule(item.Category)
		if rule == nil {
			continue
		}

		anonItem := &DataInventoryItem{
			ID:              item.ID,
			Category:        item.Category,
			Description:     fmt.Sprintf("[ANONYMIZED] %s", item.Description),
			DataFields:      anonymizeFields(item.DataFields, rule),
			StorageLocation: item.StorageLocation,
			Purpose:         item.Purpose,
			RetentionDays:   rule.RetentionDays,
			Anonymized:      true,
			Encrypted:       item.Encrypted,
			LegalBasis:      item.LegalBasis,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       time.Now().UTC(),
		}
		anonymized = append(anonymized, anonItem)
	}

	return anonymized
}

// findAnonymizationRule находит правило для категории.
func (m *PersonalDataManager) findAnonymizationRule(category DataCategory) *AnonymizationRule {
	for _, rule := range m.anonRules {
		if rule.Category == category {
			return &rule
		}
	}
	return nil
}

// anonymizeFields обезличивает поля согласно правилу.
func anonymizeFields(fields []string, rule *AnonymizationRule) []string {
	result := make([]string, len(fields))
	for i, f := range fields {
		switch rule.Method {
		case AnonMethodMasking:
			result[i] = "***" + f[len(f)-1:] // Оставляем последний символ
		case AnonMethodGeneralization:
			result[i] = "[GENERALIZED]"
		case AnonMethodPseudonymization:
			result[i] = fmt.Sprintf("pseudo_%x", hashString(f))
		case AnonMethodAggregation:
			result[i] = "[AGGREGATED]"
		case AnonMethodNoiseAddition:
			result[i] = f + "_noise"
		default:
			result[i] = f
		}
	}
	return result
}

// hashString возвращает простой хеш для псевдонимизации.
func hashString(s string) string {
	h := make([]byte, 8)
	_, _ = rand.Read(h)
	return hex.EncodeToString(h)
}

// ── Роскомнадзор Report ───────────────────────────────────────────

// GenerateRoskomnadzorReport генерирует отчёт для Роскомнадзора.
func (m *PersonalDataManager) GenerateRoskomnadzorReport(operatorName, operatorINN, operatorAddress string, subjectCount int) (*RoskomnadzorReport, error) {
	if operatorName == "" {
		return nil, fmt.Errorf("personal_data: operator_name is required")
	}
	if operatorINN == "" {
		return nil, fmt.Errorf("personal_data: operator_inn is required")
	}

	inventory, err := m.GetInventory()
	if err != nil {
		return nil, fmt.Errorf("personal_data: get inventory for report: %w", err)
	}

	categories := make(map[DataCategory]struct{})
	purposes := make(map[ConsentPurpose]struct{})
	thirdParties := make(map[string]struct{})
	crossBorder := false
	var crossBorderCountries []string
	maxRetention := 0

	for _, item := range inventory {
		categories[item.Category] = struct{}{}
		purposes[item.Purpose] = struct{}{}
		if item.CrossBorder {
			crossBorder = true
			crossBorderCountries = append(crossBorderCountries, item.StorageLocation)
		}
		for _, tp := range item.ThirdParties {
			thirdParties[tp] = struct{}{}
		}
		if item.RetentionDays > maxRetention {
			maxRetention = item.RetentionDays
		}
	}

	catList := make([]DataCategory, 0, len(categories))
	for c := range categories {
		catList = append(catList, c)
	}

	purpList := make([]ConsentPurpose, 0, len(purposes))
	for p := range purposes {
		purpList = append(purpList, p)
	}

	tpList := make([]string, 0, len(thirdParties))
	for tp := range thirdParties {
		tpList = append(tpList, tp)
	}

	report := &RoskomnadzorReport{
		OperatorName:         operatorName,
		OperatorINN:          operatorINN,
		OperatorAddress:      operatorAddress,
		DataCategories:       catList,
		SubjectCount:         subjectCount,
		ProcessingPurposes:   purpList,
		CrossBorder:          crossBorder,
		CrossBorderCountries: crossBorderCountries,
		ThirdPartyProcessors: tpList,
		DataRetentionDays:    maxRetention,
		SecurityMeasures: []string{
			"encryption_at_rest",
			"encryption_in_transit",
			"access_control_rbac",
			"audit_logging",
			"anonymization_for_analytics",
		},
		DPIACompleted: false,
		GeneratedAt:   time.Now().UTC(),
	}

	m.logger.Info("Роскомнадзор report generated",
		"operator", operatorName,
		"categories", len(catList),
		"subjects", subjectCount,
	)

	return report, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// generateID генерирует уникальный ID.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("pd_%s", hex.EncodeToString(b))
}
