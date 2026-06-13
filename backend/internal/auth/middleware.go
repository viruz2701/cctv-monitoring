package auth

import (
    "context"
    "net/http"
    "strings"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "missing authorization header", http.StatusUnauthorized)
            return
        }
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
            return
        }
        claims, err := ValidateJWT(parts[1])
        if err != nil {
            http.Error(w, "invalid or expired token", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), UserContextKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetClaims(r *http.Request) *Claims {
    claims, ok := r.Context().Value(UserContextKey).(*Claims)
    if !ok {
        return nil
    }
    return claims
}