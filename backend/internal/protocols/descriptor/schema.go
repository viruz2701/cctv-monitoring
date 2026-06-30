// Package descriptor — Protocol Descriptor Schema для декларативных протоколов.
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-01: Protocol Descriptor JSON Schema + Go structs
//
// Protocol Descriptor — это JSON-документ, который декларативно описывает
// протокол взаимодействия с устройством (HTTP endpoints, парсинг ответов,
// аутентификация и т.д.). Агент интерпретирует дескрипторы на лету,
// без необходимости компилировать новый код для каждого вендора.
//
// Архитектура:
//   - ProtocolDescriptor — корневой объект
//   - Protocol — описание транспорта (HTTP, TCP, UDP)
//   - Endpoint — HTTP endpoint с методом, путём, парсером
//   - ResponseParser — правила парсинга ответа (JSON, XML, key-value)
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS V5: Input validation (JSON Schema validation)
//
// ═══════════════════════════════════════════════════════════════════════════
package descriptor

import (
	"encoding/json"
	"fmt"
)

// ────────────────────────────────────────────────────────────────────────────
// Protocol Descriptor — корневой объект
// ────────────────────────────────────────────────────────────────────────────

// ProtocolDescriptor описывает протокол взаимодействия с устройством.
type ProtocolDescriptor struct {
	Vendor          string              `json:"vendor"`
	Version         string              `json:"version"`
	DefaultProtocol string              `json:"default_protocol,omitempty"` // какой протокол использовать по умолчанию
	Protocols       map[string]Protocol `json:"protocols"`
	RawJSON         []byte              `json:"-"` // оригинальный JSON для кэширования

	// Compliance
	Signature string `json:"signature,omitempty"` // HMAC-подпись (bash-256)
	SignedAt  string `json:"signed_at,omitempty"`
}

// Protocol описывает транспортный протокол.
type Protocol struct {
	Transport string              `json:"transport"`           // http, tcp, udp
	BaseURL   string              `json:"base_url,omitempty"`  // для HTTP: шаблон URL
	Port      int                 `json:"port,omitempty"`      // для TCP/UDP
	Auth      AuthConfig          `json:"auth,omitempty"`      // конфигурация аутентификации
	Headers   map[string]string   `json:"headers,omitempty"`   // HTTP headers по умолчанию
	Endpoints map[string]Endpoint `json:"endpoints"`           // доступные endpoints
}

// AuthConfig содержит настройки аутентификации.
type AuthConfig struct {
	Type     string `json:"type"`               // basic, digest, bearer, none
	Username string `json:"username,omitempty"` // шаблон: {{.Credentials.Username}}
	Password string `json:"password,omitempty"` // шаблон: {{.Credentials.Password}}
	Token    string `json:"token,omitempty"`    // шаблон: {{.Credentials.Token}}
}

// Endpoint описывает один endpoint устройства.
type Endpoint struct {
	Method         string            `json:"method"`                    // GET, POST, PUT, DELETE
	Path           string            `json:"path"`                      // путь с шаблонами: /ISAPI/System/deviceInfo
	Headers        map[string]string `json:"headers,omitempty"`         // переопределение headers
	Body           string            `json:"body,omitempty"`            // тело запроса (для POST/PUT)
	ContentType    string            `json:"content_type,omitempty"`    // Content-Type
	ResponseParser ResponseParser    `json:"response_parser,omitempty"` // парсинг ответа
	TimeoutSec     int               `json:"timeout_sec,omitempty"`     // таймаут в секундах
	RetryCount     int               `json:"retry_count,omitempty"`     // количество ретраев
}

// ResponseParser содержит правила парсинга ответа.
type ResponseParser struct {
	Format       string            `json:"format"`                  // json, xml, key_value, raw
	Mappings     map[string]string `json:"mappings,omitempty"`      // поле→путь (JSONPath/XPath/key)
	Separator    string            `json:"separator,omitempty"`     // для key_value: "="
	Iterator     string            `json:"iterator,omitempty"`      // XPath/JSONPath для итерации
	SuccessCheck string            `json:"success_check,omitempty"` // выражение для проверки успеха
}

// ────────────────────────────────────────────────────────────────────────────
// Execution Result
// ────────────────────────────────────────────────────────────────────────────

// ExecutionResult содержит результат выполнения операции через дескриптор.
type ExecutionResult struct {
	StatusCode int                    `json:"status_code"`
	Data       map[string]interface{} `json:"data"`
	RawBody    []byte                 `json:"-"`
	Success    bool                   `json:"success"`
	DurationMs int64                  `json:"duration_ms"`
}

// ────────────────────────────────────────────────────────────────────────────
// Validation
// ────────────────────────────────────────────────────────────────────────────

// Validate проверяет корректность дескриптора.
func (d *ProtocolDescriptor) Validate() error {
	if d.Vendor == "" {
		return fmt.Errorf("vendor is required")
	}
	if d.Version == "" {
		return fmt.Errorf("version is required")
	}
	if len(d.Protocols) == 0 {
		return fmt.Errorf("at least one protocol is required")
	}

	for name, proto := range d.Protocols {
		if proto.Transport == "" {
			return fmt.Errorf("protocol %q: transport is required", name)
		}
		if len(proto.Endpoints) == 0 {
			return fmt.Errorf("protocol %q: at least one endpoint is required", name)
		}
		for epName, ep := range proto.Endpoints {
			if ep.Method == "" {
				return fmt.Errorf("protocol %q, endpoint %q: method is required", name, epName)
			}
			if ep.Path == "" {
				return fmt.Errorf("protocol %q, endpoint %q: path is required", name, epName)
			}
		}
	}

	return nil
}

// Clone создаёт глубокую копию дескриптора.
func (d *ProtocolDescriptor) Clone() *ProtocolDescriptor {
	data, _ := json.Marshal(d)
	var clone ProtocolDescriptor
	_ = json.Unmarshal(data, &clone)
	clone.RawJSON = d.RawJSON
	return &clone
}
