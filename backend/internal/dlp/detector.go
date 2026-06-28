// Package dlp — Data Loss Prevention (P1-SEC.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SEC.2: Data Loss Prevention
//
// PII detection в exports, exports audit trail, configurable sensitivity.
//
// Sensitivity levels:
//   - low: emails, phones (redact automatically)
//   - medium: addresses (require approval)
//   - high: SSN, passport numbers (block export)
//
// Compliance:
//   - GDPR Art. 17 (Right to erasure — "right to be forgotten")
//   - ISO 27001 A.8.2.1 (Classification of information)
//   - OWASP ASVS V8.3 (Sensitive private data)
//   - Приказ ОАЦ №66 п. 7.18.5 (Data protection)
//
// ═══════════════════════════════════════════════════════════════════════════
package dlp

import (
	"regexp"
	"strings"
	"sync"
)

// ────────────────────────────────────────────────────────────────────────────
// Sensitivity levels
// ────────────────────────────────────────────────────────────────────────────

// SensitivityLevel — уровень чувствительности данных.
type SensitivityLevel int

const (
	SensitivityLow    SensitivityLevel = iota // auto-redact
	SensitivityMedium                         // require approval
	SensitivityHigh                           // block export
)

func (l SensitivityLevel) String() string {
	switch l {
	case SensitivityLow:
		return "low"
	case SensitivityMedium:
		return "medium"
	case SensitivityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PII patterns
// ────────────────────────────────────────────────────────────────────────────

// PIIType — тип PII данных.
type PIIType string

const (
	PIIEmail    PIIType = "email"
	PIIPhone    PIIType = "phone"
	PIIAddress  PIIType = "address"
	PIISSN      PIIType = "ssn"
	PIIPassport PIIType = "passport"
	PIIINN      PIIType = "inn" // Belarus INN (УНП)
	PIIBankCard PIIType = "bank_card"
)

// PIIPattern — паттерн для поиска PII.
type PIIPattern struct {
	Type        PIIType
	Regex       *regexp.Regexp
	Level       SensitivityLevel
	Description string
}

// DefaultPIIPatterns — набор стандартных PII паттернов.
// ВАЖНО: Более специфичные паттерны должны быть выше (SSN перед phone).
var DefaultPIIPatterns = []PIIPattern{
	{
		Type: PIISSN, Level: SensitivityHigh,
		Regex:       regexp.MustCompile(`\b\d{3}[-]\d{2}[-]\d{4}\b`),
		Description: "US Social Security Numbers",
	},
	{
		Type: PIIBankCard, Level: SensitivityHigh,
		Regex:       regexp.MustCompile(`\b\d{4}[-.\s]?\d{4}[-.\s]?\d{4}[-.\s]?\d{4}\b`),
		Description: "Bank card numbers",
	},
	{
		Type: PIIPassport, Level: SensitivityHigh,
		Regex:       regexp.MustCompile(`\b[A-Z]{2}\d{7}\b`),
		Description: "Passport numbers (e.g. MP1234567)",
	},
	{
		Type: PIIEmail, Level: SensitivityLow,
		Regex:       regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		Description: "Email addresses",
	},
	{
		Type: PIIPhone, Level: SensitivityLow,
		Regex:       regexp.MustCompile(`\b\+?\d{1,3}[-.\s]?\d{2,4}[-.\s]?\d{2,4}[-.\s]?\d{2,4}\b`),
		Description: "Phone numbers",
	},
	{
		Type: PIIAddress, Level: SensitivityMedium,
		Regex:       regexp.MustCompile(`\d{1,3}\s+[A-Za-zА-Яа-я]+(?:[-\s]+[A-Za-zА-Яа-я]+)*\s+(?:ул\.|Street|str\.|пр\.|проспект|бульвар)`),
		Description: "Street addresses",
	},
	{
		Type: PIIINN, Level: SensitivityMedium,
		Regex:       regexp.MustCompile(`\b\d{9}\b`),
		Description: "Belarus UNP (УНП) / Tax ID",
	},
}

// ────────────────────────────────────────────────────────────────────────────
// DLP Detector
// ────────────────────────────────────────────────────────────────────────────

// PIIMatch — найденное PII совпадение.
type PIIMatch struct {
	Type     PIIType          `json:"type"`
	Value    string           `json:"value,omitempty"`
	Level    SensitivityLevel `json:"level"`
	Start    int              `json:"start"`
	End      int              `json:"end"`
	Redacted bool             `json:"redacted"`
}

// DLPDetector — детектор PII данных.
type DLPDetector struct {
	mu       sync.RWMutex
	patterns []PIIPattern
}

// NewDLPDetector создаёт DLP детектор с паттернами по умолчанию.
func NewDLPDetector() *DLPDetector {
	return &DLPDetector{
		patterns: DefaultPIIPatterns,
	}
}

// NewDLPDetectorWithPatterns создаёт DLP детектор с кастомными паттернами.
func NewDLPDetectorWithPatterns(patterns []PIIPattern) *DLPDetector {
	return &DLPDetector{
		patterns: patterns,
	}
}

// AddPattern добавляет кастомный паттерн (thread-safe).
func (d *DLPDetector) AddPattern(pattern PIIPattern) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.patterns = append(d.patterns, pattern)
}

// Detect ищет PII в тексте. Возвращает все совпадения.
func (d *DLPDetector) Detect(data string) []PIIMatch {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var matches []PIIMatch
	seen := make(map[string]bool)

	for _, pattern := range d.patterns {
		locs := pattern.Regex.FindAllStringIndex(data, -1)
		for _, loc := range locs {
			if loc[0] >= len(data) || loc[1] > len(data) {
				continue
			}
			value := data[loc[0]:loc[1]]
			key := string(pattern.Type) + ":" + value
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, PIIMatch{
				Type:  pattern.Type,
				Value: value,
				Level: pattern.Level,
				Start: loc[0],
				End:   loc[1],
			})
		}
	}

	return matches
}

// DetectSensitivity определяет максимальный уровень чувствительности в данных.
func (d *DLPDetector) DetectSensitivity(data string) SensitivityLevel {
	matches := d.Detect(data)
	maxLevel := SensitivityLow - 1
	for _, m := range matches {
		if m.Level > maxLevel {
			maxLevel = m.Level
		}
	}
	if maxLevel < SensitivityLow {
		return SensitivityLow - 1 // no PII found
	}
	return maxLevel
}

// HasHighSensitivity проверяет, содержит ли данные высокочувствительную PII.
func (d *DLPDetector) HasHighSensitivity(data string) bool {
	return d.DetectSensitivity(data) >= SensitivityHigh
}

// ────────────────────────────────────────────────────────────────────────────
// Redaction
// ────────────────────────────────────────────────────────────────────────────

// Redact заменяет PII на маскированные значения.
// Email: "user@example.com" → "u***@example.com"
// Phone: "+375291234567" → "+37529******"
// Другие: "***REDACTED***"
func Redact(data string, matches []PIIMatch) string {
	// Сортируем от конца к началу, чтобы не менять индексы
	type replacement struct {
		start, end int
		text       string
	}
	var repls []replacement
	for _, m := range matches {
		repls = append(repls, replacement{
			start: m.Start,
			end:   m.End,
			text:  redactValue(m.Type, m.Value),
		})
	}

	// Применяем замены с конца строки
	result := data
	for i := len(repls) - 1; i >= 0; i-- {
		r := repls[i]
		if r.start >= 0 && r.end <= len(result) {
			result = result[:r.start] + r.text + result[r.end:]
		}
	}

	return result
}

// redactValue возвращает маскированное значение для PII типа.
func redactValue(t PIIType, value string) string {
	switch t {
	case PIIEmail:
		parts := strings.SplitN(value, "@", 2)
		if len(parts) == 2 && len(parts[0]) > 0 {
			return string(parts[0][0]) + "***@" + parts[1]
		}
	case PIIPhone:
		if len(value) > 6 {
			return value[:5] + strings.Repeat("*", len(value)-7) + value[len(value)-2:]
		}
	}
	return "***REDACTED***"
}

// ────────────────────────────────────────────────────────────────────────────
// Audit
// ────────────────────────────────────────────────────────────────────────────

// AuditRecord — запись аудита для DLP события.
type AuditRecord struct {
	Action      string           `json:"action"` // "export", "redact", "block"
	UserID      string           `json:"user_id"`
	DataType    string           `json:"data_type"` // "work_orders", "devices", etc.
	PIIFound    []PIIMatch       `json:"pii_found"`
	MaxLevel    SensitivityLevel `json:"max_level"`
	Result      string           `json:"result"` // "allowed", "redacted", "blocked"
	ExportCount int              `json:"export_count"`
}
