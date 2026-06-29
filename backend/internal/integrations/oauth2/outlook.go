package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// ── Microsoft OAuth2 Config ───────────────────────────────────────────

// OutlookCalendarScopes — необходимые scopes для Microsoft Graph Calendar API.
var OutlookCalendarScopes = []string{
	"https://graph.microsoft.com/Calendars.ReadWrite",
	"https://graph.microsoft.com/User.Read",
	"offline_access",
}

// OutlookConfig возвращает OAuth2 config для Microsoft Outlook.
// tenantID может быть "common" (multi-tenant), "organizations" или конкретный tenant.
func OutlookConfig(clientID, clientSecret, redirectURL, tenantID string) *oauth2.Config {
	if tenantID == "" {
		tenantID = "common"
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       OutlookCalendarScopes,
		Endpoint:     microsoft.AzureADEndpoint(tenantID),
	}
}

// NewOutlookClient создаёт HTTP-клиент с OAuth2 токеном для Microsoft Graph API.
//
// refreshFn вызывается при успешном обновлении токена для сохранения в БД.
func NewOutlookClient(ctx context.Context, cfg *oauth2.Config,
	token *oauth2.Token, refreshFn func(*oauth2.Token) error) (*http.Client, error) {

	if token == nil {
		return nil, fmt.Errorf("outlook oauth2: token is nil")
	}

	if !token.Valid() && token.RefreshToken != "" {
		ts := cfg.TokenSource(ctx, token)
		newToken, err := ts.Token()
		if err != nil {
			return nil, fmt.Errorf("outlook oauth2: refresh token: %w", err)
		}

		if refreshFn != nil && newToken.AccessToken != token.AccessToken {
			if err := refreshFn(newToken); err != nil {
				return nil, fmt.Errorf("outlook oauth2: save refreshed token: %w", err)
			}
		}
		token = newToken
	}

	return cfg.Client(ctx, token), nil
}

// ParseOutlookToken создаёт oauth2.Token из полей БД.
func ParseOutlookToken(accessToken, refreshToken string, expiry time.Time) *oauth2.Token {
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
