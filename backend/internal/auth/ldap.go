// Package auth — LDAP authentication provider.
//
// INT-02: LDAP integration для enterprise-аутентификации.
//
// Поддерживает:
//   - LDAP bind authentication (service account → user search → user bind)
//   - Auto-provisioning: создание пользователя при первом входе
//   - Role mapping: LDAP group → local role
//   - Graceful degradation: при недоступности LDAP — возврат к local auth
//
// Соответствует:
//   - OWASP ASVS V2.1 (Password strength — LDAP password policy)
//   - OWASP ASVS V2.5 (Service authentication — LDAP bind)
//   - ISO 27001 A.9.2.1 (User registration — auto-provisioning)
//   - ISO 27001 A.9.4.2 (Authentication — LDAP)
//   - Приказ ОАЦ №66 п.7.18.1 (Идентификация конечных узлов)
//
// ═══════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ── LDAP Configuration ───────────────────────────────────────────────

// LDAPConfig — конфигурация LDAP подключения.
type LDAPConfig struct {
	// Host — LDAP сервер (например, "ldap.example.com")
	Host string `json:"host" yaml:"host"`
	// Port — порт (389 для LDAP, 636 для LDAPS)
	Port int `json:"port" yaml:"port"`
	// UseTLS — использовать TLS (LDAPS)
	UseTLS bool `json:"use_tls" yaml:"use_tls"`
	// InsecureSkipVerify — пропустить проверку сертификата (только dev)
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// BaseDN — базовый DN для поиска пользователей (например, "dc=example,dc=com")
	BaseDN string `json:"base_dn" yaml:"base_dn"`
	// BindDN — DN сервисного аккаунта для поиска
	BindDN string `json:"bind_dn" yaml:"bind_dn"`
	// BindPassword — пароль сервисного аккаунта
	BindPassword string `json:"bind_password" yaml:"bind_password"`
	// UserFilter — фильтр поиска пользователей (например, "(uid=%s)" или "(sAMAccountName=%s)")
	UserFilter string `json:"user_filter" yaml:"user_filter"`
	// LoginAttribute — атрибут для входа (по умолчанию "uid")
	LoginAttribute string `json:"login_attribute" yaml:"login_attribute"`

	// Attribute mapping
	MailAttribute   string `json:"mail_attribute" yaml:"mail_attribute"`
	NameAttribute   string `json:"name_attribute" yaml:"name_attribute"`
	RoleAttribute   string `json:"role_attribute" yaml:"role_attribute"`
	RoleGroupFilter string `json:"role_group_filter" yaml:"role_group_filter"`

	// Role mapping: LDAP group DN → local role
	RoleMapping map[string]string `json:"role_mapping" yaml:"role_mapping"`

	// DefaultRole — роль по умолчанию для новых пользователей
	DefaultRole string `json:"default_role" yaml:"default_role"`
	// Enabled — включена ли LDAP аутентификация
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Timeout — таймаут подключения
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultLDAPConfig — конфигурация по умолчанию.
func DefaultLDAPConfig() LDAPConfig {
	return LDAPConfig{
		Port:           389,
		UserFilter:     "(uid=%s)",
		LoginAttribute: "uid",
		MailAttribute:  "mail",
		NameAttribute:  "cn",
		RoleAttribute:  "cn",
		DefaultRole:    "viewer",
		Enabled:        false,
		Timeout:        10 * time.Second,
		RoleMapping: map[string]string{
			"cn=admin,ou=groups,dc=example,dc=com":      "admin",
			"cn=manager,ou=groups,dc=example,dc=com":    "manager",
			"cn=technician,ou=groups,dc=example,dc=com": "technician",
			"cn=viewer,ou=groups,dc=example,dc=com":     "viewer",
		},
	}
}

// ── LDAP Provider ────────────────────────────────────────────────────

// LDAPProvider — интерфейс LDAP аутентификации.
//
// Абстрагирует go-ldap/ldap/v3 для возможности тестирования.
type LDAPProvider interface {
	// Authenticate проверяет логин/пароль через LDAP.
	// Возвращает LDAPUserInfo при успехе.
	Authenticate(username, password string) (*LDAPUserInfo, error)
	// IsAvailable проверяет доступность LDAP сервера.
	IsAvailable() bool
	// Config возвращает текущую конфигурацию.
	Config() LDAPConfig
}

// LDAPUserInfo — информация о пользователе из LDAP.
type LDAPUserInfo struct {
	DN            string            `json:"dn"`
	UID           string            `json:"uid"`
	Mail          string            `json:"mail"`
	DisplayName   string            `json:"display_name"`
	Groups        []string          `json:"groups"`
	MappedRole    string            `json:"mapped_role"`
	RawAttributes map[string]string `json:"-"`
}

// ── LDAP Authenticator ───────────────────────────────────────────────

// LDAPAuthenticator — реализация LDAPProvider через go-ldap.
type LDAPAuthenticator struct {
	config LDAPConfig
	logger *slog.Logger
}

// NewLDAPAuthenticator создаёт LDAPAuthenticator.
func NewLDAPAuthenticator(config LDAPConfig, logger *slog.Logger) *LDAPAuthenticator {
	if logger == nil {
		logger = slog.Default()
	}
	return &LDAPAuthenticator{
		config: config,
		logger: logger.With("component", "ldap-auth"),
	}
}

// Authenticate выполняет LDAP bind аутентификацию.
//
// Алгоритм:
//  1. Подключение к LDAP серверу (TLS если настроено)
//  2. Bind с сервисным аккаунтом
//  3. Поиск пользователя по фильтру
//  4. Bind с учётными данными пользователя (верификация пароля)
//  5. Извлечение атрибутов и групп
//  6. Маппинг роли
func (l *LDAPAuthenticator) Authenticate(username, password string) (*LDAPUserInfo, error) {
	if !l.config.Enabled {
		return nil, errors.New("ldap authentication is disabled")
	}
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	conn, err := l.dial()
	if err != nil {
		return nil, fmt.Errorf("ldap dial: %w", err)
	}
	defer conn.Close()

	// Шаг 1: Bind с сервисным аккаунтом
	if err := conn.Bind(l.config.BindDN, l.config.BindPassword); err != nil {
		return nil, fmt.Errorf("ldap bind (service): %w", err)
	}

	// Шаг 2: Поиск пользователя
	loginAttr := l.config.LoginAttribute
	if loginAttr == "" {
		loginAttr = "uid"
	}

	filter := strings.ReplaceAll(l.config.UserFilter, "%s", ldapEscapeFilter(username))

	searchReq := &ldapSearchRequest{
		BaseDN:     l.config.BaseDN,
		Scope:      2, // WholeSubtree
		Filter:     filter,
		Attributes: l.getAttributes(),
	}

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("ldap search: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, errors.New("user not found in LDAP")
	}

	entry := result.Entries[0]
	userDN := entry.DN

	// Шаг 3: Bind с пользователем (верификация пароля)
	if err := conn.Bind(userDN, password); err != nil {
		return nil, errors.New("invalid ldap credentials")
	}

	// Шаг 4: Извлечение атрибутов
	userInfo := l.extractUserInfo(entry)

	return userInfo, nil
}

// IsAvailable проверяет доступность LDAP сервера.
func (l *LDAPAuthenticator) IsAvailable() bool {
	if !l.config.Enabled {
		return false
	}
	conn, err := l.dial()
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// Config возвращает текущую конфигурацию.
func (l *LDAPAuthenticator) Config() LDAPConfig {
	return l.config
}

// ── Internal ─────────────────────────────────────────────────────────

func (l *LDAPAuthenticator) dial() (ldapConn, error) {
	addr := fmt.Sprintf("%s:%d", l.config.Host, l.config.Port)

	if l.config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: l.config.InsecureSkipVerify,
		}
		return dialTLS(addr, tlsConfig)
	}

	return dial(addr)
}

func (l *LDAPAuthenticator) getAttributes() []string {
	attrs := []string{
		"dn", l.config.LoginAttribute,
	}
	if l.config.MailAttribute != "" {
		attrs = append(attrs, l.config.MailAttribute)
	}
	if l.config.NameAttribute != "" {
		attrs = append(attrs, l.config.NameAttribute)
	}
	// Всегда запрашиваем memberOf для определения групп
	attrs = append(attrs, "memberOf")
	return attrs
}

func (l *LDAPAuthenticator) extractUserInfo(entry *ldapEntry) *LDAPUserInfo {
	info := &LDAPUserInfo{
		DN:            entry.DN,
		UID:           entry.GetAttributeValue(l.config.LoginAttribute),
		Mail:          entry.GetAttributeValue(l.config.MailAttribute),
		DisplayName:   entry.GetAttributeValue(l.config.NameAttribute),
		Groups:        entry.GetAttributeValues("memberOf"),
		RawAttributes: make(map[string]string),
	}

	// Маппинг роли из групп
	role := l.config.DefaultRole
	if role == "" {
		role = "viewer"
	}

	for _, group := range info.Groups {
		if mappedRole, ok := l.config.RoleMapping[group]; ok {
			role = mappedRole
			break
		}
		// Также проверяем по CN группы
		for groupDN, mapped := range l.config.RoleMapping {
			if strings.Contains(group, groupDN) || strings.HasSuffix(group, groupDN) {
				role = mapped
				break
			}
		}
	}
	info.MappedRole = role

	return info
}

// ── Stub interfaces for testability ───────────────────────────────────

// ldapConn — интерфейс LDAP соединения (для возможности мокирования).
type ldapConn interface {
	Bind(username, password string) error
	Search(req *ldapSearchRequest) (*ldapSearchResult, error)
	Close() error
}

type ldapSearchRequest struct {
	BaseDN     string
	Scope      int
	Filter     string
	Attributes []string
}

type ldapSearchResult struct {
	Entries []*ldapEntry
}

type ldapEntry struct {
	DN         string
	Attributes []*ldapAttribute
}

type ldapAttribute struct {
	Name   string
	Values []string
}

func (e *ldapEntry) GetAttributeValue(name string) string {
	for _, attr := range e.Attributes {
		if strings.EqualFold(attr.Name, name) && len(attr.Values) > 0 {
			return attr.Values[0]
		}
	}
	return ""
}

func (e *ldapEntry) GetAttributeValues(name string) []string {
	for _, attr := range e.Attributes {
		if strings.EqualFold(attr.Name, name) {
			return attr.Values
		}
	}
	return nil
}

// ldapEscapeFilter экранирует спецсимволы в LDAP фильтре.
func ldapEscapeFilter(filter string) string {
	replacer := strings.NewReplacer(
		"\\", "\\5c",
		"*", "\\2a",
		"(", "\\28",
		")", "\\29",
		"\x00", "\\00",
		"/", "\\2f",
	)
	return replacer.Replace(filter)
}

// dial — заглушка для LDAP подключения (временно).
// TODO: Заменить на go-ldap/ldap/v3 после добавления в go.mod
var dial = func(addr string) (ldapConn, error) {
	return nil, fmt.Errorf("ldap not implemented: add github.com/go-ldap/ldap/v3 to go.mod")
}

var dialTLS = func(addr string, config *tls.Config) (ldapConn, error) {
	return nil, fmt.Errorf("ldaps not implemented: add github.com/go-ldap/ldap/v3 to go.mod")
}
