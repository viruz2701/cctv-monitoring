// Package auth — WebAuthn/FIDO2 Authentication (P1-SEC.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SEC.1: WebAuthn/FIDO2 Support
//
// Добавляет WebAuthn (FIDO2) как альтернативу TOTP для 2FA:
//   - Hardware tokens (YubiKey, Titan)
//   - Biometric authentication (Touch ID, Face ID)
//   - Passwordless login option
//
// Архитектура:
//   - WebAuthnStore — in-memory store для сессий регистрации и credentials
//   - WebAuthnBackend — высокоуровневый API для registration/login
//   - Recovery codes — backup при потере устройства
//
// Compliance:
//   - OWASP ASVS V2.4 (Credential recovery — recovery codes)
//   - OWASP ASVS V2.5 (Hardware-backed authentication — FIDO2)
//   - ISO 27001 A.9.2.1 (Strong authentication — hardware tokens)
//   - Приказ ОАЦ №66 п. 7.18.2 (Secure authentication)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// ────────────────────────────────────────────────────────────────────────────
// WebAuthnStore — хранилище для WebAuthn данных
// ────────────────────────────────────────────────────────────────────────────

// WebAuthnUserData — данные пользователя для WebAuthn.
type WebAuthnUserData struct {
	UserID      string
	DisplayName string
	Credentials []webauthn.Credential
}

// WebAuthnSessionData — данные сессии WebAuthn (registration/login).
type WebAuthnSessionData struct {
	SessionData *webauthn.SessionData
	UserID      string
}

// WebAuthnStore — thread-safe хранилище для WebAuthn данных.
type WebAuthnStore struct {
	mu         sync.RWMutex
	users      map[string]*WebAuthnUserData
	sessions   map[string]*WebAuthnSessionData
	sessionIdx int
}

// NewWebAuthnStore создаёт новое WebAuthn хранилище.
func NewWebAuthnStore() *WebAuthnStore {
	return &WebAuthnStore{
		users:    make(map[string]*WebAuthnUserData),
		sessions: make(map[string]*WebAuthnSessionData),
	}
}

func (s *WebAuthnStore) SaveUser(user *WebAuthnUserData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.UserID] = user
}

func (s *WebAuthnStore) GetUser(userID string) (*WebAuthnUserData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userID]
	return user, ok
}

func (s *WebAuthnStore) AddCredential(userID string, cred *webauthn.Credential) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.users[userID]; ok {
		user.Credentials = append(user.Credentials, *cred)
	}
}

func (s *WebAuthnStore) SaveSession(sessionID string, data *WebAuthnSessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = data
}

func (s *WebAuthnStore) GetSession(sessionID string) (*WebAuthnSessionData, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	return data, ok
}

func (s *WebAuthnStore) GenerateSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionIdx++
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s-%d", base64.RawURLEncoding.EncodeToString(b), s.sessionIdx)
}

// ────────────────────────────────────────────────────────────────────────────
// WebAuthnUser — реализация интерфейса webauthn.User
// ────────────────────────────────────────────────────────────────────────────

type WebAuthnUser struct {
	data *WebAuthnUserData
}

func NewWebAuthnUser(data *WebAuthnUserData) *WebAuthnUser {
	return &WebAuthnUser{data: data}
}

func (u *WebAuthnUser) WebAuthnID() []byte                         { return []byte(u.data.UserID) }
func (u *WebAuthnUser) WebAuthnName() string                       { return u.data.UserID }
func (u *WebAuthnUser) WebAuthnDisplayName() string                { return u.data.DisplayName }
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.data.Credentials }

// ────────────────────────────────────────────────────────────────────────────
// WebAuthnBackend — высокоуровневый API для WebAuthn
// ────────────────────────────────────────────────────────────────────────────

type WebAuthnConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

var DefaultWebAuthnConfig = WebAuthnConfig{
	RPDisplayName: "CCTV Health Monitor",
	RPID:          "localhost",
	RPOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
}

type WebAuthnBackend struct {
	wa    *webauthn.WebAuthn
	store *WebAuthnStore
	log   *slog.Logger
}

func NewWebAuthnBackend(cfg WebAuthnConfig, store *WebAuthnStore, log *slog.Logger) (*WebAuthnBackend, error) {
	if log == nil {
		log = slog.Default()
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}
	return &WebAuthnBackend{wa: wa, store: store, log: log.With("component", "webauthn")}, nil
}

// BeginRegistration — начинает регистрацию WebAuthn credentials.
func (b *WebAuthnBackend) BeginRegistration(userID, displayName string) (*protocol.CredentialCreation, string, error) {
	userData := &WebAuthnUserData{UserID: userID, DisplayName: displayName}
	b.store.SaveUser(userData)
	user := NewWebAuthnUser(userData)
	options, session, err := b.wa.BeginRegistration(user)
	if err != nil {
		return nil, "", fmt.Errorf("begin registration: %w", err)
	}
	sessionID := b.store.GenerateSessionID()
	b.store.SaveSession(sessionID, &WebAuthnSessionData{SessionData: session, UserID: userID})
	return options, sessionID, nil
}

// FinishRegistration — завершает регистрацию через HTTP request.
func (b *WebAuthnBackend) FinishRegistration(userID, sessionID string, r *http.Request) (*webauthn.Credential, error) {
	session, ok := b.store.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("webauthn: session not found or expired")
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("webauthn: session user mismatch")
	}
	userData, ok := b.store.GetUser(userID)
	if !ok {
		return nil, fmt.Errorf("webauthn: user not found")
	}
	user := NewWebAuthnUser(userData)
	credential, err := b.wa.FinishRegistration(user, *session.SessionData, r)
	if err != nil {
		return nil, fmt.Errorf("finish registration: %w", err)
	}
	b.store.AddCredential(userID, credential)
	b.log.Info("WebAuthn credential registered", "user_id", userID)
	return credential, nil
}

// BeginLogin — начинает WebAuthn login.
func (b *WebAuthnBackend) BeginLogin(userID string) (*protocol.CredentialAssertion, string, error) {
	userData, ok := b.store.GetUser(userID)
	if !ok {
		return nil, "", fmt.Errorf("webauthn: user %s has no credentials", userID)
	}
	user := NewWebAuthnUser(userData)
	options, session, err := b.wa.BeginLogin(user)
	if err != nil {
		return nil, "", fmt.Errorf("begin login: %w", err)
	}
	sessionID := b.store.GenerateSessionID()
	b.store.SaveSession(sessionID, &WebAuthnSessionData{SessionData: session, UserID: userID})
	return options, sessionID, nil
}

// FinishLogin — завершает WebAuthn login через HTTP request.
func (b *WebAuthnBackend) FinishLogin(userID, sessionID string, r *http.Request) error {
	session, ok := b.store.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("webauthn: session not found or expired")
	}
	if session.UserID != userID {
		return fmt.Errorf("webauthn: session user mismatch")
	}
	userData, ok := b.store.GetUser(userID)
	if !ok {
		return fmt.Errorf("webauthn: user not found")
	}
	user := NewWebAuthnUser(userData)
	_, err := b.wa.FinishLogin(user, *session.SessionData, r)
	if err != nil {
		return fmt.Errorf("finish login: %w", err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Recovery Codes
// ────────────────────────────────────────────────────────────────────────────

type RecoveryCode struct {
	Code string `json:"code"`
	Used bool   `json:"used"`
}

func GenerateRecoveryCodes(count int) ([]RecoveryCode, []string) {
	if count <= 0 {
		count = 8
	}
	codes := make([]RecoveryCode, count)
	raw := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 9)
		rand.Read(b)
		code := base64.RawURLEncoding.EncodeToString(b)
		codes[i] = RecoveryCode{Code: code}
		raw[i] = code
	}
	return codes, raw
}

func GenerateRecoveryCodesJSON(count int) ([]RecoveryCode, string, error) {
	codes, raw := GenerateRecoveryCodes(count)
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, "", fmt.Errorf("marshal recovery codes: %w", err)
	}
	return codes, string(jsonBytes), nil
}

func ValidateRecoveryCode(codes []RecoveryCode, input string) ([]RecoveryCode, bool) {
	for i, c := range codes {
		if !c.Used && c.Code == input {
			codes[i].Used = true
			return codes, true
		}
	}
	return codes, false
}

func HasRecoveryCodesLeft(codes []RecoveryCode) bool {
	for _, c := range codes {
		if !c.Used {
			return true
		}
	}
	return false
}
