// Package config — SSO (SAML/LDAP) configuration.
//
// INT-02: SAML 2.0 / LDAP SSO configuration loader.
//
// Соответствует:
//   - OWASP ASVS V2 (Authentication)
//   - ISO 27001 A.9.2 (User access)
//   - Приказ ОАЦ №66 п.7.18.1 (Идентификация)
package config

// SSOConfig — конфигурация SSO (загружается из основного Config).
type SSOConfig struct {
	LDAP LDAPSSOConfig `json:"ldap" yaml:"ldap"`
	SAML SAMLSSOConfig `json:"saml" yaml:"saml"`
}

// LDAPSSOConfig — LDAP аутентификация.
type LDAPSSOConfig struct {
	Enabled        bool              `json:"enabled" yaml:"enabled"`
	Host           string            `json:"host" yaml:"host"`
	Port           int               `json:"port" yaml:"port"`
	UseTLS         bool              `json:"use_tls" yaml:"use_tls"`
	BaseDN         string            `json:"base_dn" yaml:"base_dn"`
	BindDN         string            `json:"bind_dn" yaml:"bind_dn"`
	BindPassword   string            `json:"bind_password" yaml:"bind_password"`
	UserFilter     string            `json:"user_filter" yaml:"user_filter"`
	LoginAttribute string            `json:"login_attribute" yaml:"login_attribute"`
	MailAttribute  string            `json:"mail_attribute" yaml:"mail_attribute"`
	NameAttribute  string            `json:"name_attribute" yaml:"name_attribute"`
	DefaultRole    string            `json:"default_role" yaml:"default_role"`
	RoleMapping    map[string]string `json:"role_mapping" yaml:"role_mapping"`
}

// SAMLSSOConfig — SAML 2.0 аутентификация.
type SAMLSSOConfig struct {
	Enabled        bool              `json:"enabled" yaml:"enabled"`
	IdPMetadataURL string            `json:"idp_metadata_url" yaml:"idp_metadata_url"`
	IdPEntityID    string            `json:"idp_entity_id" yaml:"idp_entity_id"`
	IdPSSOURL      string            `json:"idp_sso_url" yaml:"idp_sso_url"`
	SPEntityID     string            `json:"sp_entity_id" yaml:"sp_entity_id"`
	AcsURL         string            `json:"acs_url" yaml:"acs_url"`
	DefaultRole    string            `json:"default_role" yaml:"default_role"`
	MailAttribute  string            `json:"mail_attribute" yaml:"mail_attribute"`
	NameAttribute  string            `json:"name_attribute" yaml:"name_attribute"`
	RoleAttribute  string            `json:"role_attribute" yaml:"role_attribute"`
	RoleMapping    map[string]string `json:"role_mapping" yaml:"role_mapping"`
}
