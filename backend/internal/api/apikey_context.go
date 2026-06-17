package api

import (
	"context"
	"gb-telemetry-collector/internal/db"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

// setAPIKeyContext adds API key to context
func setAPIKeyContext(ctx context.Context, key *db.APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, key)
}

// GetAPIKeyFromContext retrieves API key from context
func GetAPIKeyFromContext(ctx context.Context) *db.APIKey {
	if key, ok := ctx.Value(apiKeyContextKey).(*db.APIKey); ok {
		return key
	}
	return nil
}
