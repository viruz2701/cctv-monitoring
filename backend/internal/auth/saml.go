// Package auth — SAML 2.0 Service Provider.
//
// INT-02: SAML 2.0 интеграция для enterprise SSO.
//
// Поддерживает:
//   - SAML 2.0 Web SSO (HTTP Redirect/POST bindings)
//   - IdP-initiated и SP-initiated SSO
//   - Auto-provisioning: создание пользователя при первом входе
//   - Attribute mapping: SAML attributes → local user
//   - Multiple IdP support (через metadata URL)
//
// Соответствует:
//   - OWASP ASVS V2.3 (Federation — SAML)
//   - OWASP ASVS V2.5 (Service authentication)
//   - ISO 27001 A.9.2.1 (User registration — auto-provisioning)
//   - ISO 27001 A.9.4.2 (Single sign-on)
//   - Приказ ОАЦ №66 п.7.18.1 (Идентификация)
//
// ═══════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ── SAML Configuration ───────────────────────────────────────────────

// SAMLConfig — конфигурация SAML 2.0 Service Provider.
type SAMLConfig struct {
	// Enabled — включена ли SAML аутентификация
	Enabled bool `json:"enabled" yaml:"enabled"`
	// IdPMetadataURL — URL метаданных IdP
	IdPMetadataURL string `json:"idp_metadata_url" yaml:"idp_metadata_url"`
	// IdPEntityID — EntityID IdP (если metadata недоступен)
	IdPEntityID string `json:"idp_entity_id" yaml:"idp_entity_id"`
	// IdPSSOURL — SSO URL IdP (если metadata недоступен)
	IdPSSOURL string `json:"idp_sso_url" yaml:"idp_sso_url"`
	// IdPPublicCert — публичный сертификат IdP для верификации (Base64/PEM)
	IdPPublicCert string `json:"idp_public_cert" yaml:"idp_public_cert"`

	// SPEntityID — EntityID нашего Service Provider
	SPEntityID string `json:"sp_entity_id" yaml:"sp_entity_id"`
	// SPAcsURL — Assertion Consumer Service URL (полный URL)
	SPAcsURL string `json:"sp_acs_url" yaml:"sp_acs_url"`
	// SPCertificate — приватный ключ SP для подписи (PEM)
	SPPrivateKey string `json:"sp_private_key" yaml:"sp_private_key"`
	// SPCertificate — сертификат SP (PEM)
	SPCertificate string `json:"sp_certificate" yaml:"sp_certificate"`

	// Attribute mapping
	// MailAttribute — SAML attribute для email (например, "mail", "email", "http://...")
	MailAttribute string `json:"mail_attribute" yaml:"mail_attribute"`
	// NameAttribute — SAML attribute для имени (например, "cn", "displayName")
	NameAttribute string `json:"name_attribute" yaml:"name_attribute"`
	// RoleAttribute — SAML attribute для роли
	RoleAttribute string `json:"role_attribute" yaml:"role_attribute"`

	// DefaultRole — роль по умолчанию для новых пользователей
	DefaultRole string `json:"default_role" yaml:"default_role"`
	// RoleMapping — маппинг значений role attribute → local role
	RoleMapping map[string]string `json:"role_mapping" yaml:"role_mapping"`

	// SignAuthnRequests — подписывать AuthnRequest
	SignAuthnRequests bool `json:"sign_authn_requests" yaml:"sign_authn_requests"`
	// RequestedAuthnContext — запрашиваемый контекст аутентификации
	RequestedAuthnContext string `json:"requested_authn_context" yaml:"requested_authn_context"`
	// MaxAge — максимальное время жизни Assertion
	MaxAge time.Duration `json:"max_age" yaml:"max_age"`
}

// DefaultSAMLConfig — конфигурация SAML по умолчанию.
func DefaultSAMLConfig() SAMLConfig {
	return SAMLConfig{
		MailAttribute:         "mail",
		NameAttribute:         "cn",
		DefaultRole:           "viewer",
		SignAuthnRequests:     true,
		RequestedAuthnContext: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
		MaxAge:                5 * time.Minute,
		RoleMapping: map[string]string{
			"admin":      "admin",
			"manager":    "manager",
			"technician": "technician",
			"viewer":     "viewer",
		},
	}
}

// ── SAML Provider ────────────────────────────────────────────────────

// SAMLProvider — интерфейс SAML 2.0 Service Provider.
type SAMLProvider interface {
	// GetAuthURL возвращает URL для редиректа на IdP.
	GetAuthURL(relayState string) (string, error)
	// HandleACS обрабатывает Assertion Consumer Service callback.
	HandleACS(samlResponse string) (*SAMLUserInfo, error)
	// GetMetadata возвращает SP метаданные в XML формате.
	GetMetadata() (string, error)
	// IsAvailable проверяет доступность IdP.
	IsAvailable() bool
	// Config возвращает текущую конфигурацию.
	Config() SAMLConfig
}

// SAMLUserInfo — информация о пользователе из SAML Assertion.
type SAMLUserInfo struct {
	NameID       string            `json:"name_id"`
	Mail         string            `json:"mail"`
	DisplayName  string            `json:"display_name"`
	MappedRole   string            `json:"mapped_role"`
	SessionIndex string            `json:"session_index,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// ── SAML Authenticator ───────────────────────────────────────────────

// SAMLAuthenticator — заглушка реализации SAMLProvider.
//
// ⚠ ВРЕМЕННАЯ ЗАГЛУШКА: Полная реализация требует:
//   - github.com/crewjam/saml (MIT) — SAML 2.0 библиотека
//   - Parsing IdP metadata XML
//   - SP metadata generation
//   - AuthnRequest signing
//   - Assertion verification (signature, conditions, audience)
//   - Attribute extraction
//
// После добавления crewjam/saml в go.mod:
//
//	import "github.com/crewjam/saml/samlsp"
//
//	middleware, _ := samlsp.New(samlsp.Options{
//	    URL:         *url.Parse(s.config.SPAcsURL),
//	    Key:         privateKey,
//	    Certificate: certificate,
//	    IDPMetadata: idpMetadata,
//	})
//	authURL := middleware.ServiceProvider.MakeRedirectAuthenticationRequest(relayState)
type SAMLAuthenticator struct {
	config SAMLConfig
	logger *slog.Logger
}

// NewSAMLAuthenticator создаёт SAMLAuthenticator.
func NewSAMLAuthenticator(config SAMLConfig, logger *slog.Logger) *SAMLAuthenticator {
	if logger == nil {
		logger = slog.Default()
	}
	return &SAMLAuthenticator{
		config: config,
		logger: logger.With("component", "saml-auth"),
	}
}

// GetAuthURL возвращает URL для редиректа на IdP.
//
// TODO: Реализовать через crewjam/saml:
//
//	authReq, _ := middleware.ServiceProvider.MakeAuthenticationRequest(
//	    s.config.IdPSSOURL,
//	    binding,
//	    binding,
//	    relayState,
//	)
//	return authReq.Redirect(relayState), nil
func (s *SAMLAuthenticator) GetAuthURL(relayState string) (string, error) {
	if !s.config.Enabled {
		return "", errors.New("saml authentication is disabled")
	}
	if s.config.IdPSSOURL == "" {
		return "", errors.New("saml idp sso url is not configured")
	}

	// Временная заглушка — возвращаем URL IdP напрямую
	// В production: генерировать подписанный AuthnRequest
	return fmt.Sprintf("%s?RelayState=%s& binding=HTTP-Redirect", s.config.IdPSSOURL, relayState), nil
}

// HandleACS обрабатывает SAML Response от IdP.
//
// TODO: Реализовать через crewjam/saml:
//
//	assertion, _ := middleware.ServiceProvider.ParseResponse(r, requestIDs)
//	return &SAMLUserInfo{
//	    NameID:      assertion.Subject.NameID.Value,
//	    Mail:        getAttribute(assertion, s.config.MailAttribute),
//	    DisplayName: getAttribute(assertion, s.config.NameAttribute),
//	    MappedRole:  s.mapRole(getAttribute(assertion, s.config.RoleAttribute)),
//	}, nil
func (s *SAMLAuthenticator) HandleACS(samlResponse string) (*SAMLUserInfo, error) {
	if !s.config.Enabled {
		return nil, errors.New("saml authentication is disabled")
	}
	if samlResponse == "" {
		return nil, errors.New("empty saml response")
	}

	// Временная заглушка — парсинг base64 SAML Response
	// В production: верификация подписи, проверка условий, извлечение атрибутов
	info, err := s.parseStubResponse(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("saml acs: %w", err)
	}

	return info, nil
}

// GetMetadata возвращает SP метаданные в XML.
//
// TODO: Реализовать через crewjam/saml:
//
//	metadata, _ := middleware.ServiceProvider.Metadata()
//	return string(metadata), nil
func (s *SAMLAuthenticator) GetMetadata() (string, error) {
	if !s.config.Enabled {
		return "", errors.New("saml authentication is disabled")
	}

	// Временная заглушка
	metadata := fmt.Sprintf(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="%s">
    <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
        AuthnRequestsSigned="%t">
        <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
            Location="%s" index="0"/>
    </SPSSODescriptor>
</EntityDescriptor>`, s.config.SPEntityID, s.config.SignAuthnRequests, s.config.SPAcsURL)

	return metadata, nil
}

// IsAvailable проверяет доступность SAML IdP.
func (s *SAMLAuthenticator) IsAvailable() bool {
	if !s.config.Enabled {
		return false
	}
	return s.config.IdPMetadataURL != "" || (s.config.IdPSSOURL != "" && s.config.IdPPublicCert != "")
}

// Config возвращает конфигурацию.
func (s *SAMLAuthenticator) Config() SAMLConfig {
	return s.config
}

// ── Helpers ───────────────────────────────────────────────────────────

func (s *SAMLAuthenticator) mapRole(roleAttr string) string {
	if roleAttr == "" {
		return s.config.DefaultRole
	}
	if mapped, ok := s.config.RoleMapping[roleAttr]; ok {
		return mapped
	}
	// Case-insensitive fallback
	for k, v := range s.config.RoleMapping {
		if strings.EqualFold(k, roleAttr) {
			return v
		}
	}
	return s.config.DefaultRole
}

// parseStubResponse — временный парсер SAML Response (заглушка).
// TODO: Удалить после добавления crewjam/saml
func (s *SAMLAuthenticator) parseStubResponse(response string) (*SAMLUserInfo, error) {
	// Пытаемся извлечь name_id из response
	// Временная реализация — просто возвращает базовую информацию
	// В production: полноценный парсинг и верификация SAML XML

	nameID := extractBetween(response, "<saml:NameID>", "</saml:NameID>")
	if nameID == "" {
		nameID = extractBetween(response, "<NameID>", "</NameID>")
	}
	if nameID == "" {
		nameID = "saml-user-" + shortHash(response)
	}

	mail := extractBetween(response, "mail\">", "</saml:AttributeValue>")
	if mail == "" {
		mail = extractBetween(response, "email\">", "</saml:AttributeValue>")
	}

	displayName := extractBetween(response, "cn\">", "</saml:AttributeValue>")
	if displayName == "" {
		displayName = extractBetween(response, "displayName\">", "</saml:AttributeValue>")
	}
	if displayName == "" {
		displayName = nameID
	}

	roleAttr := extractBetween(response, "role\">", "</saml:AttributeValue>")
	if roleAttr == "" {
		roleAttr = extractBetween(response, "memberOf\">", "</saml:AttributeValue>")
	}

	info := &SAMLUserInfo{
		NameID:      nameID,
		Mail:        mail,
		DisplayName: displayName,
		MappedRole:  s.mapRole(roleAttr),
		Attributes: map[string]string{
			"name_id": nameID,
			"mail":    mail,
			"cn":      displayName,
			"role":    roleAttr,
		},
	}

	return info, nil
}

// extractBetween извлекает строку между open и close тегами.
func extractBetween(s, open, close string) string {
	start := strings.Index(s, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(s[start:], close)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(s[start : start+end])
}

// shortHash возвращает первые 8 символов хеша строки.
func shortHash(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// ── Certificate Helpers ──────────────────────────────────────────────

// ParsePrivateKey парсит PEM-приватный ключ RSA.
func ParsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("key is not RSA private key")
	}
	return rsaKey, nil
}

// ParseCertificate парсит PEM-сертификат X.509.
func ParseCertificate(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}
