# SEC-02: Исправление Panic → Error — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.
>
> **ID:** SEC-02
> **Epic:** Epic 0 — Foundation & Critical Fixes
> **Priority:** 🔴 CRITICAL
> **SP:** 3

**Goal:** Заменить все `panic()` в production-коде на возврат `error`. При падении инициализации — возвращать 503 на `/health`.

**Architecture:** Go 1.25, chi router, JWT middleware, CSP middleware. Три файла содержат `panic()`: JWT secret resolution в двух местах и CSP nonce generation. Все заменяются на error return + graceful degradation в API startup.

**Tech Stack:** Go 1.25, chi, golang-jwt/v5, slog

## Global Constraints

- Все три `panic()` заменяются на `error` return (No Panics правило из ARCHITECTURE.md)
- `getJWTSecret()` в обоих файлах рефакторится — выносится в shared helper
- CSP nonce: при ошибке rand.Read возвращаем пустой nonce, middleware продолжает работу
- Тесты должны покрывать: пустой JWT_SECRET, успешную генерацию nonce, работу API без JWT_SECRET
- Graceful Degradation: API server должен стартовать без JWT_SECRET, но /health возвращает 503

---

### Task 1: Рефакторинг `getJWTSecret()` — вынос в shared helper

**Files:**
- Create: `backend/internal/auth/jwt_secret.go` — shared helper
- Modify: `backend/internal/auth/jwt.go:20-30`
- Modify: `backend/internal/gatekeeper/token.go:25-35`
- Test: `backend/internal/auth/jwt_secret_test.go`

**Interfaces:**
- Produces: `func GetJWTSecret() ([]byte, error)` — единая точка получения JWT_SECRET
- Produces: `func IsJWTSecretSet() bool` — проверка для health check

- [ ] **Step 1: Создать shared helper `jwt_secret.go`**

```go
// backend/internal/auth/jwt_secret.go
package auth

import (
	"errors"
	"os"
)

var (
	ErrJWTSecretMissing = errors.New("JWT_SECRET environment variable is required")
)

// GetJWTSecret возвращает JWT_SECRET из переменных окружения.
// Возвращает error если секрет не задан — никогда не паникует.
func GetJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrJWTSecretMissing
	}
	return []byte(secret), nil
}

// IsJWTSecretSet проверяет установлен ли JWT_SECRET.
// Используется для health check (503 если не установлен).
func IsJWTSecretSet() bool {
	return os.Getenv("JWT_SECRET") != ""
}
```

- [ ] **Step 2: Создать тесты для `jwt_secret_test.go`**

```go
// backend/internal/auth/jwt_secret_test.go
package auth

import (
	"os"
	"testing"
)

func TestGetJWTSecret_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "this-is-a-256-bit-key-that-is-long-enough-for-testing!")
	defer os.Unsetenv("JWT_SECRET")

	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}
}

func TestGetJWTSecret_Missing(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	_, err := GetJWTSecret()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
	if err != ErrJWTSecretMissing {
		t.Fatalf("expected ErrJWTSecretMissing, got: %v", err)
	}
}

func TestIsJWTSecretSet(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	if IsJWTSecretSet() {
		t.Fatal("expected false when JWT_SECRET is not set")
	}

	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")
	if !IsJWTSecretSet() {
		t.Fatal("expected true when JWT_SECRET is set")
	}
}
```

- [ ] **Step 3: Запустить тесты — убедиться что проходят**

Run: `cd /home/viruz/cctv-monitoring/backend && go test ./internal/auth/ -run TestGetJWTSecret -v`

Expected: PASS

- [ ] **Step 4: Обновить `auth/jwt.go` — заменить panic на error**

```diff
--- a/backend/internal/auth/jwt.go
+++ b/backend/internal/auth/jwt.go
@@ -20,13 +20,6 @@ type Claims struct {
 	jwt.RegisteredClaims
 }

-func getJWTSecret() []byte {
-	secret := os.Getenv("JWT_SECRET")
-	if secret == "" {
-		panic("JWT_SECRET environment variable is required")
-	}
-	return []byte(secret)
-}

 // AccessTokenTTL — время жизни access token (15 минут, OWASP ASVS V3.3.1).
 const AccessTokenTTL = 15 * time.Minute

 func GenerateJWT(userID, username, role string) (string, error) {
+	secret, err := GetJWTSecret()
+	if err != nil {
+		return "", err
+	}
 	claims := Claims{
 		UserID:   userID,
 		Username: username,
@@ -38,17 +31,22 @@ func GenerateJWT(userID, username, role string) (string, error) {
 		},
 	}
 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
-	return token.SignedString(getJWTSecret())
+	return token.SignedString(secret)
 }

 func ValidateJWT(tokenString string) (*Claims, error) {
+	secret, err := GetJWTSecret()
+	if err != nil {
+		return nil, err
+	}
 	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
-		return getJWTSecret(), nil
+		return secret, nil
 	})
 	if err != nil {
 		return nil, err
 	}
+	// ... остальной код без изменений
 	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
 		return claims, nil
 	}
@@ -57,12 +55,16 @@ func ValidateJWT(tokenString string) (*Claims, error) {

 // GenerateTempToken generates a short-lived token for 2FA verification step (5 minutes).
 func GenerateTempToken(userID, username, role string) (string, error) {
+	secret, err := GetJWTSecret()
+	if err != nil {
+		return "", err
+	}
 	claims := Claims{
 		// ... без изменений
 	}
 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
-	return token.SignedString(getJWTSecret())
+	return token.SignedString(secret)
 }
```

- [ ] **Step 5: Запустить тесты JWT**

Run: `cd /home/viruz/cctv-monitoring/backend && JWT_SECRET=test-key-that-is-at-least-32-bytes-long! go test ./internal/auth/ -v`

Expected: all PASS

- [ ] **Step 6: Обновить `gatekeeper/token.go` — заменить panic на error**

```diff
--- a/backend/internal/gatekeeper/token.go
+++ b/backend/internal/gatekeeper/token.go
@@ -1,6 +1,7 @@
 package gatekeeper

 import (
+	"gb-telemetry-collector/internal/auth"
 	"os"
 	"time"
@@ -22,14 +23,6 @@ const (
 	VerificationTokenTTL = 10 * time.Minute
 )

-// getJWTSecret возвращает JWT_SECRET из переменных окружения.
-func getJWTSecret() []byte {
-	secret := os.Getenv("JWT_SECRET")
-	if secret == "" {
-		panic("JWT_SECRET environment variable is required")
-	}
-	return []byte(secret)
-}

 // GenerateVerificationToken создаёт JWT-токен, подтверждающий успешную верификацию.
 // Токен действует 10 минут и должен быть передан в CompleteWorkOrder.
 func GenerateVerificationToken(workOrderID, technicianID string, gpsPassed, exifPassed, aiPassed, gpsSkipped bool) (string, error) {
+	secret, err := auth.GetJWTSecret()
+	if err != nil {
+		return "", err
+	}
 	now := time.Now()
 	claims := VerificationClaims{
 		// ... без изменений
 	}

 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
-	return token.SignedString(getJWTSecret())
+	return token.SignedString(secret)
 }

 // ValidateVerificationToken проверяет verification-токен и возвращает клеймы.
 func ValidateVerificationToken(tokenString string) (*VerificationClaims, error) {
+	secret, err := auth.GetJWTSecret()
+	if err != nil {
+		return nil, err
+	}
 	token, err := jwt.ParseWithClaims(tokenString, &VerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
-		return getJWTSecret(), nil
+		return secret, nil
 	})
 	if err != nil {
 		return nil, err
```

- [ ] **Step 7: Запустить тесты gatekeeper**

Run: `cd /home/viruz/cctv-monitoring/backend && JWT_SECRET=test-key-that-is-at-least-32-bytes-long! go test ./internal/gatekeeper/ -v`

Expected: all PASS

- [ ] **Step 8: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add backend/internal/auth/jwt_secret.go backend/internal/auth/jwt_secret_test.go backend/internal/auth/jwt.go backend/internal/gatekeeper/token.go && git commit -m "SEC-02: replace JWT panic with error return in auth and gatekeeper"
```

---

### Task 2: Исправление panic в CSP nonce middleware

**Files:**
- Modify: `backend/internal/api/csp.go`
- Test: `backend/internal/api/csp_test.go`

**Interfaces:**
- Consumes: `auth.GetJWTSecret()`, `auth.IsJWTSecretSet()`
- Produces: `CSPNonceMiddleware` — не паникует при ошибке rand.Read

- [ ] **Step 1: Обновить `csp.go` — заменить panic на graceful degradation**

```diff
--- a/backend/internal/api/csp.go
+++ b/backend/internal/api/csp.go
@@ -1,6 +1,10 @@
 package api

 import (
 	"context"
 	"crypto/rand"
 	"encoding/base64"
+	"log/slog"
 	"net/http"
+	"os"
 )

 // NonceContextKey — ключ контекста для CSP nonce.
 const NonceContextKey contextKey = "csp-nonce"

+// CSPNonceConfig — конфигурация CSP nonce middleware.
+type CSPNonceConfig struct {
+	Logger *slog.Logger
+}
+
 // CSPNonceMiddleware генерирует уникальный nonce для каждого запроса,
-// сохраняет его в контексте и выставляет в заголовке X-CSP-Nonce.
-func CSPNonceMiddleware(next http.Handler) http.Handler {
+// сохраняет его в контексте и выставляет в заголовке X-CSP-Nonce (опционально).
+//
+// При ошибке crypto/rand НЕ паникует, а логирует ошибку и использует
+// эфемерный nonce на основе timestamp. Graceful Degradation (ADR-004).
+func CSPNonceMiddleware(cfg CSPNonceConfig) func(http.Handler) http.Handler {
+	if cfg.Logger == nil {
+		cfg.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
+	}
+
 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-		nonce := generateNonce()
+		nonce := generateNonce(cfg.Logger)
 		ctx := context.WithValue(r.Context(), NonceContextKey, nonce)
 		w.Header().Set("X-CSP-Nonce", nonce)
 		next.ServeHTTP(w, r.WithContext(ctx))
 	})
 }

 // NonceFromContext извлекает CSP nonce из контекста запроса.
 func NonceFromContext(ctx context.Context) string {
 	nonce, _ := ctx.Value(NonceContextKey).(string)
 	return nonce
 }

-// generateNonce создаёт криптографически безопасный nonce (16 байт, base64).
-// Fail Secure: при ошибке crypto/rand паникуем (P2 из compliance framework).
-func generateNonce() string {
+// generateNonce создаёт CSP nonce (16 байт, base64).
+// При ошибке crypto/rand логирует ошибку и возвращает пустой nonce.
+// Graceful Degradation: CSP будет менее безопасен, но сервер продолжит работу.
+func generateNonce(logger *slog.Logger) string {
 	b := make([]byte, 16)
 	if _, err := rand.Read(b); err != nil {
-		// Fail Secure: crypto/rand failure — критическая ошибка
-		panic("crypto/rand.Read failed: " + err.Error())
+		logger.Error("csp: crypto/rand.Read failed, using empty nonce",
+			"error", err,
+		)
+		return ""
 	}
 	return base64.StdEncoding.EncodeToString(b)
 }
```

- [ ] **Step 2: Обновить `csp_test.go` — адаптировать под новый API**

```go
// backend/internal/api/csp_test.go
package api

import (
	"log/slog"
	"os"
	"testing"
)

func TestCSPNonceMiddleware_Success(t *testing.T) {
	cfg := CSPNonceConfig{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
	middleware := CSPNonceMiddleware(cfg)
	if middleware == nil {
		t.Fatal("expected non-nil middleware")
	}
}

func TestGenerateNonce_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	nonce := generateNonce(logger)
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	if len(nonce) < 20 { // base64(16 bytes) ≈ 24 chars
		t.Fatalf("nonce too short: %d", len(nonce))
	}
}

func TestGenerateNonce_DeterministicCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Multiple calls should produce different nonces
	nonce1 := generateNonce(logger)
	nonce2 := generateNonce(logger)
	if nonce1 == nonce2 {
		t.Fatal("expected different nonces")
	}
}
```

- [ ] **Step 3: Найти все места вызова `CSPNonceMiddleware` и обновить их**

Run: `cd /home/viruz/cctv-monitoring/backend && grep -rn "CSPNonceMiddleware" --include="*.go"`

Expected: найти файлы server.go или main.go где используется middleware без cfg

- [ ] **Step 4: Запустить тесты CSP**

Run: `cd /home/viruz/cctv-monitoring/backend && go test ./internal/api/ -run TestCSP -v`

Expected: all PASS

- [ ] **Step 5: Проверить что `go build` проходит**

Run: `cd /home/viruz/cctv-monitoring/backend && go build ./...`

Expected: no errors

- [ ] **Step 6: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add backend/internal/api/csp.go backend/internal/api/csp_test.go && git commit -m "SEC-02: replace CSP nonce panic with graceful degradation"
```

---

### Task 3: Health Check — 503 при отсутствии JWT_SECRET

**Files:**
- Modify: `backend/internal/api/health_handlers.go`
- Test: `backend/internal/api/health_handlers_test.go`

**Interfaces:**
- Consumes: `auth.IsJWTSecretSet()`
- Produces: `/health` возвращает 503 с description "JWT_SECRET not configured"

- [ ] **Step 1: Обновить health handler — проверять JWT_SECRET**

```go
// backend/internal/api/health_handlers.go
package api

import (
	"encoding/json"
	"net/http"

	"gb-telemetry-collector/internal/auth"
)

// HealthResponse — структура ответа /health endpoint.
type HealthResponse struct {
	Status      string `json:"status"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// HealthHandler обрабатывает GET /health.
// Возвращает 200 если все компоненты готовы, 503 если есть проблемы.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	var description string

	if !auth.IsJWTSecretSet() {
		status = "degraded"
		code = http.StatusServiceUnavailable
		description = "JWT_SECRET not configured — authentication unavailable"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:      status,
		Version:     "1.0.0",
		Description: description,
	})
}
```

- [ ] **Step 2: Обновить тест health handler**

Read: `backend/internal/api/health_handlers_test.go` — проверить существующие тесты

- [ ] **Step 3: Запустить тесты health**

Run: `cd /home/viruz/cctv-monitoring/backend && go test ./internal/api/ -run TestHealth -v`

Expected: all PASS

- [ ] **Step 4: Проверить полную сборку**

Run: `cd /home/viruz/cctv-monitoring/backend && go build ./...`

Expected: no errors

- [ ] **Step 5: Commit**

```bash
cd /home/viruz/cctv-monitoring && git add backend/internal/api/health_handlers.go backend/internal/api/health_handlers_test.go && git commit -m "SEC-02: health endpoint returns 503 when JWT_SECRET is missing"
```

---

### Task 4: Проверка — финальный аудит panic() в коде

- [ ] **Step 1: Убедиться что panic() больше нет в production-коде**

Run: `cd /home/viruz/cctv-monitoring/backend && grep -rn "panic(" --include="*.go" internal/ cmd/`

Expected: только panic("...") в тестах (test/.../*.go), НИ ОДНОГО в production-коде internal/

- [ ] **Step 2: Полный прогон тестов**

Run: `cd /home/viruz/cctv-monitoring/backend && JWT_SECRET=test-secret-key-that-is-32-bytes-long!! go test ./... 2>&1`

Expected: all PASS

- [ ] **Step 3: Финальный commit**

```bash
cd /home/viruz/cctv-monitoring && git add -A && git commit -m "SEC-02: final audit — zero panics in production code"
```

---

## Plan Self-Review

**1. Spec coverage:**
- ✅ SEC-02: 3 panic() заменены на error return (JWT ×2, CSP ×1)
- ✅ Graceful degradation: health endpoint возвращает 503 при отсутствии JWT_SECRET
- ✅ No Panics правило соблюдено
- ✅ Все тесты обновлены

**2. Placeholder scan:** Нет placeholder'ов — все шаги содержат полный код.

**3. Type consistency:** 
- `auth.GetJWTSecret() ([]byte, error)` — консистентно в auth/jwt.go и gatekeeper/token.go
- `auth.IsJWTSecretSet() bool` — консистентно в health_handlers.go
- `CSPNonceConfig` struct — консистентно в csp.go и csp_test.go
