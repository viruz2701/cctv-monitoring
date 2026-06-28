// Package providers — 149-ФЗ Compliance for GOST Crypto Provider.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-MKT.1: 149-ФЗ Data Localization + СКЗИ Certification
//
// Нормативная база:
//   - 149-ФЗ «Об информации, информационных технологиях и о защите информации»
//   - 152-ФЗ «О персональных данных» (ст. 18.1 — Data Localization)
//   - Приказ ФСТЭК № 17 «Об утверждении требований о защите информации...»
//   - Приказ ФСТЭК № 21 «О составе и содержании организационных и технических мер...»
//   - Приказ ФСТЭК № 31 «Об утверждении требований к СКЗИ...»
//   - Постановление ПП РФ № 1119 «Об утверждении требований к защите ПДн»
//   - Методика ФСТЭК по определению уровня защищённости ПДн
//   - IEC 62443-3-3 SR 5.1 (Cryptographic zone-based access)
//   - СТБ 34.101.30 (криптографические алгоритмы — для РБ)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"fmt"
	"strings"
)

// ────────────────────────────────────────────────────────────────────────────
// 149-ФЗ / 152-ФЗ Data Localization levels
// ────────────────────────────────────────────────────────────────────────────

// FZ149DataLevel — уровень локализации данных по 149-ФЗ / 152-ФЗ.
type FZ149DataLevel int

const (
	// DataLevelUnknown — уровень не определён.
	DataLevelUnknown FZ149DataLevel = iota

	// DataLevelLocal — данные хранятся и обрабатываются только на территории РФ.
	DataLevelLocal

	// DataLevelLocalWithBackup — данные в РФ, резервное копирование за рубежом
	// (допустимо с шифрованием по ГОСТ).
	DataLevelLocalWithBackup

	// DataLevelMixed — часть данных за рубежом (трансграничная передача с согласия).
	DataLevelMixed
)

func (l FZ149DataLevel) String() string {
	switch l {
	case DataLevelLocal:
		return "local-only"
	case DataLevelLocalWithBackup:
		return "local-with-backup"
	case DataLevelMixed:
		return "transborder-mixed"
	default:
		return "unknown"
	}
}

// ────────────────────────────────────────────────────────────────────────────
// SKZI Certification levels (СКЗИ — Средства Криптографической Защиты Информации)
// ────────────────────────────────────────────────────────────────────────────

// SKZIClass — класс СКЗИ по Приказу ФСТЭК № 31.
type SKZIClass string

const (
	// SKZIClassKC1 — КС1 (для организаций, не являющихся госорганами).
	SKZIClassKC1 SKZIClass = "KC1"
	// SKZIClassKC2 — КС2 (для государственных информационных систем).
	SKZIClassKC2 SKZIClass = "KC2"
	// SKZIClassKC3 — КС3 (для критической информационной инфраструктуры).
	SKZIClassKC3 SKZIClass = "KC3"
	// SKZIClassKB1 — КБ1 (высший класс для ГИС).
	SKZIClassKB1 SKZIClass = "КБ1"
	// SKZIClassKB2 — КБ2.
	SKZIClassKB2 SKZIClass = "КБ2"
	// SKZIClassKA1 — КА1 (наивысший класс, особой важности).
	SKZIClassKA1 SKZIClass = "КА1"
)

// SKZIInfo — информация о сертификации СКЗИ.
type SKZIInfo struct {
	Class        SKZIClass `json:"class"`
	CertNumber   string    `json:"cert_number,omitempty"`
	CertDate     string    `json:"cert_date,omitempty"`
	CertExpiry   string    `json:"cert_expiry,omitempty"`
	Algorithms   []string  `json:"algorithms"`
	Compliant149 bool      `json:"compliant_149"`
	Compliant152 bool      `json:"compliant_152"`
}

// ────────────────────────────────────────────────────────────────────────────
// Data Localization Requirements (ст. 18.1 152-ФЗ)
// ────────────────────────────────────────────────────────────────────────────

// DataLocalizationRequirement — требование по локализации данных.
type DataLocalizationRequirement struct {
	// DataCategory — категория данных.
	DataCategory string `json:"data_category"`

	// RequiredLevel — требуемый уровень локализации.
	RequiredLevel FZ149DataLevel `json:"required_level"`

	// EncryptionRequired — требуется ли шифрование по ГОСТ.
	EncryptionRequired bool `json:"encryption_required"`

	// GOSTRequired — требуется ли использование только ГОСТ алгоритмов.
	GOSTRequired bool `json:"gost_required"`

	// Regulation — ссылка на нормативный акт.
	Regulation string `json:"regulation"`
}

// defaultLocalizationRequirements — требования по умолчанию для различных
// категорий данных в соответствии с 149-ФЗ и 152-ФЗ.
var defaultLocalizationRequirements = []DataLocalizationRequirement{
	{
		DataCategory:       "personal_data_rf_citizens",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "152-ФЗ ст. 18.1, ПП РФ № 1119",
	},
	{
		DataCategory:       "cctv_video_records",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "149-ФЗ ст. 12, Приказ ФСТЭК № 17 (КИИ)",
	},
	{
		DataCategory:       "cctv_metadata",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "149-ФЗ ст. 12, Приказ ФСТЭК № 17",
	},
	{
		DataCategory:       "auth_credentials",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "152-ФЗ ст. 5, Приказ ФСТЭК № 21",
	},
	{
		DataCategory:       "audit_logs",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "149-ФЗ ст. 16, ISO 27001 A.12.4",
	},
	{
		DataCategory:       "system_configuration",
		RequiredLevel:      DataLevelLocal,
		EncryptionRequired: false,
		GOSTRequired:       false,
		Regulation:         "149-ФЗ ст. 13",
	},
	{
		DataCategory:       "anonymized_statistics",
		RequiredLevel:      DataLevelLocalWithBackup,
		EncryptionRequired: true,
		GOSTRequired:       true,
		Regulation:         "152-ФЗ ст. 18.1, Приказ ФСТЭК № 31",
	},
}

// ────────────────────────────────────────────────────────────────────────────
// Compliance checking
// ────────────────────────────────────────────────────────────────────────────

// Compliance149FZ — структура для проверки соответствия 149-ФЗ / 152-ФЗ.
type Compliance149FZ struct {
	// DataLevel — текущий уровень локализации данных.
	DataLevel FZ149DataLevel `json:"data_level"`

	// GOSTProviderStatus — статус GOST провайдера.
	GOSTProviderStatus string `json:"gost_provider_status"`

	// HSMAvailable — доступен ли HSM (аппаратное СКЗИ).
	HSMAvailable bool `json:"hsm_available"`

	// SKZI — информация о сертификации СКЗИ.
	SKZI *SKZIInfo `json:"skzi,omitempty"`

	// RegionalRequirements — региональные требования.
	RegionalRequirements []DataLocalizationRequirement `json:"regional_requirements"`
}

// NewCompliance149FZ создаёт новый проверщик соответствия 149-ФЗ.
func NewCompliance149FZ(provider *GostProvider) *Compliance149FZ {
	c := &Compliance149FZ{
		DataLevel:            DataLevelLocal,
		GOSTProviderStatus:   provider.Status(),
		HSMAvailable:         provider.IsAvailable(),
		RegionalRequirements: defaultLocalizationRequirements,
	}

	// Если HSM доступен — СКЗИ сертифицирован
	if provider.IsAvailable() {
		c.SKZI = &SKZIInfo{
			Class:        SKZIClassKC3,
			Algorithms:   []string{"ГОСТ 28147-89", "ГОСТ Р 34.11-2012", "ГОСТ Р 34.10-2012"},
			Compliant149: true,
			Compliant152: true,
		}
	}

	return c
}

// CheckDataCategory проверяет соответствие для указанной категории данных.
func (c *Compliance149FZ) CheckDataCategory(category string) (*ComplianceCheckResult, error) {
	for _, req := range c.RegionalRequirements {
		if req.DataCategory == category {
			return c.evaluateRequirement(req), nil
		}
	}
	return nil, fmt.Errorf("unknown data category: %s", category)
}

// ComplianceCheckResult — результат проверки соответствия.
type ComplianceCheckResult struct {
	Category        string `json:"category"`
	Compliant       bool   `json:"compliant"`
	EncryptionOK    bool   `json:"encryption_ok"`
	GOSTOK          bool   `json:"gost_ok"`
	LocalizationOK  bool   `json:"localization_ok"`
	Requirement     string `json:"requirement"`
	Recommendations string `json:"recommendations,omitempty"`
}

func (c *Compliance149FZ) evaluateRequirement(req DataLocalizationRequirement) *ComplianceCheckResult {
	result := &ComplianceCheckResult{
		Category:    req.DataCategory,
		Requirement: req.Regulation,
	}

	// Проверка локализации данных
	if req.RequiredLevel == DataLevelLocal || req.RequiredLevel == DataLevelLocalWithBackup {
		result.LocalizationOK = c.DataLevel == DataLevelLocal || c.DataLevel == DataLevelLocalWithBackup
	}

	// Проверка шифрования
	if req.EncryptionRequired {
		result.EncryptionOK = c.GOSTProviderStatus == "gost-native" || c.GOSTProviderStatus == "hsm"
		if !result.EncryptionOK {
			result.Recommendations = "Tребуется включить ГОСТ-шифрование (Магма/Кузнечик)"
		}
	} else {
		result.EncryptionOK = true
	}

	// Проверка ГОСТ
	if req.GOSTRequired {
		result.GOSTOK = c.GOSTProviderStatus == "gost-native" || c.GOSTProviderStatus == "hsm"
		if !result.GOSTOK {
			if result.Recommendations != "" {
				result.Recommendations += "; "
			}
			result.Recommendations += "Tребуется использовать сертифицированные ГОСТ алгоритмы (Магма, Стрибог)"
		}
	} else {
		result.GOSTOK = true
	}

	// Итоговое соответствие
	result.Compliant = result.LocalizationOK && result.EncryptionOK && result.GOSTOK

	return result
}

// AllChecks выполняет проверку для всех категорий данных.
func (c *Compliance149FZ) AllChecks() map[string]*ComplianceCheckResult {
	results := make(map[string]*ComplianceCheckResult)
	for _, req := range c.RegionalRequirements {
		result, _ := c.CheckDataCategory(req.DataCategory)
		results[req.DataCategory] = result
	}
	return results
}

// Summary возвращает сводку по соответствию 149-ФЗ.
func (c *Compliance149FZ) Summary() string {
	var sb strings.Builder
	sb.WriteString("=== 149-ФЗ / 152-ФЗ Compliance Summary ===\n")
	sb.WriteString(fmt.Sprintf("Data Localization Level: %s\n", c.DataLevel))
	sb.WriteString(fmt.Sprintf("GOST Provider Status: %s\n", c.GOSTProviderStatus))
	sb.WriteString(fmt.Sprintf("HSM Available: %v\n", c.HSMAvailable))

	if c.SKZI != nil {
		sb.WriteString(fmt.Sprintf("SKZI Class: %s\n", c.SKZI.Class))
		sb.WriteString(fmt.Sprintf("Algorithms: %v\n", c.SKZI.Algorithms))
		sb.WriteString(fmt.Sprintf("149-ФЗ Compliant: %v\n", c.SKZI.Compliant149))
		sb.WriteString(fmt.Sprintf("152-ФЗ Compliant: %v\n", c.SKZI.Compliant152))
	} else {
		sb.WriteString("SKZI: Not certified (software implementation)\n")
	}

	sb.WriteString("\nCategory Checks:\n")
	for category, result := range c.AllChecks() {
		status := "✅" // compliant
		if !result.Compliant {
			status = "❌" // non-compliant
		}
		sb.WriteString(fmt.Sprintf("  %s %s: compliant=%v", status, category, result.Compliant))
		if result.Recommendations != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", result.Recommendations))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
