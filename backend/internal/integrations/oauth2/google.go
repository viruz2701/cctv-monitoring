// Package oauth2 — OAuth2 helpers для Google Calendar и Microsoft Outlook.
//
// ═══════════════════════════════════════════════════════════════════════
// P1-CALENDAR: External Calendar Sync
//
// Compliance:
//   - ISO 27001 A.9.2 (User authentication — OAuth2)
//   - IEC 62443-3-3 SL-2 (DMZ — token exchange)
//   - OWASP ASVS V3.1 (Session management — secure tokens)
//
// ═══════════════════════════════════════════════════════════════════════
package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ── Google OAuth2 Config ──────────────────────────────────────────────

// GoogleCalendarScopes — необходимые scopes для Google Calendar API.
var GoogleCalendarScopes = []string{
	"https://www.googleapis.com/auth/calendar",
	"https://www.googleapis.com/auth/calendar.events",
}

// GoogleConfig возвращает OAuth2 config для Google Calendar.
func GoogleConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       GoogleCalendarScopes,
		Endpoint:     google.Endpoint,
	}
}

// NewGoogleClient создаёт HTTP-клиент с OAuth2 токеном для Google Calendar.
// Если токен истёк, автоматически пытается обновить через refresh token.
//
// refreshFn вызывается при успешном обновлении токена для сохранения в БД.
func NewGoogleClient(ctx context.Context, cfg *oauth2.Config,
	token *oauth2.Token, refreshFn func(*oauth2.Token) error) (*http.Client, error) {

	if token == nil {
		return nil, fmt.Errorf("google oauth2: token is nil")
	}

	// Проверяем, не истёк ли токен
	if !token.Valid() && token.RefreshToken != "" {
		ts := cfg.TokenSource(ctx, token)
		newToken, err := ts.Token()
		if err != nil {
			return nil, fmt.Errorf("google oauth2: refresh token: %w", err)
		}

		// Сохраняем обновлённый токен
		if refreshFn != nil && newToken.AccessToken != token.AccessToken {
			if err := refreshFn(newToken); err != nil {
				return nil, fmt.Errorf("google oauth2: save refreshed token: %w", err)
			}
		}
		token = newToken
	}

	return cfg.Client(ctx, token), nil
}

// ParseGoogleToken создаёт oauth2.Token из полей БД.
func ParseGoogleToken(accessToken, refreshToken string, expiry time.Time) *oauth2.Token {
	t := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}
	if !expiry.IsZero() {
		t.Expiry = expiry
	}
	return t
}
