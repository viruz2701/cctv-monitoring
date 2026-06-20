# ISO 27001 Remediation Plan

**Дата:** 2026-06-20
**Цель:** План устранения gaps, выявленных в gap-analysis

---

## Phase 1: Quick Wins (Неделя 1-2)

### QW-1: JWT Secret — убрать дефолтное значение
**Gap:** #7 (CRITICAL)
**Файл:** [`jwt.go`](../../backend/internal/auth/jwt.go)
**Действие:**
```go
func getJWTSecret() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        panic("JWT_SECRET environment variable is required")
    }
    return []byte(secret)
}
```
**Время:** 10 минут

### QW-2: API Keys — SHA-256 → bcrypt
**Gap:** #4 (CRITICAL)
**Файл:** [`apikey_handlers.go`](../../backend/internal/api/apikey_handlers.go)
**Действие:** Заменить `sha256.Sum256` на `bcrypt.GenerateFromPassword` с cost=12.
**Время:** 1 час

### QW-3: Push Tokens — AES-256-GCM шифрование
**Gap:** #8 (CRITICAL)
**Файл:** [`cmms_repository.go`](../../backend/internal/db/cmms_repository.go)
**Действие:** Добавить `crypto/aes` шифрование перед сохранением, расшифровку при чтении.
**Время:** 2 часа

### QW-4: Config secrets — env vars
**Gap:** #9 (HIGH)
**Файл:** [`config.yaml`](../../backend/config.yaml)
**Действие:** Вынести `p2p_api_key`, FTP пароль, Hikvision пароли в env vars.
**Время:** 1 час

### QW-5: Rate limiting на login
**Gap:** #5 (HIGH)
**Файл:** [`server.go`](../../backend/internal/api/server.go)
**Действие:** Добавить `chi-rate-limiter` middleware на `/api/v1/auth/login`.
**Время:** 1 час

### QW-6: Security headers
**Gap:** #14 (MEDIUM)
**Файл:** [`server.go`](../../backend/internal/api/server.go)
**Действие:** Добавить middleware с CSP, X-Frame-Options, X-Content-Type-Options.
**Время:** 30 минут

---

## Phase 2: Medium-term (Неделя 3-4)

### MT-1: Security Policy
**Gap:** #1 (HIGH)
**Действие:** Создать `docs/iso27001/security-policy.md`.
**Время:** 4 часа

### MT-2: Asset Classification
**Gap:** #2 (MEDIUM)
**Действие:** Добавить `asset_class` в БД, обновить модели.
**Время:** 2 часа

### MT-3: Audit Log Integrity
**Gap:** #11 (MEDIUM)
**Действие:** Добавить HMAC-подпись для audit-записей.
**Время:** 3 часа

### MT-4: CORS restriction
**Gap:** #10 (MEDIUM)
**Действие:** Заменить `*` на список разрешённых origins из конфига.
**Время:** 30 минут

### MT-5: Incident Response Plan
**Gap:** #17 (HIGH)
**Действие:** Создать `docs/iso27001/incident-response-plan.md`.
**Время:** 4 часа

### MT-6: Vulnerability scanning в CI/CD
**Gap:** #12 (MEDIUM)
**Действие:** Добавить `gosec`, `trivy`, `npm audit` в GitHub Actions.
**Время:** 3 часа

---

## Phase 3: Long-term (Месяц 2-3)

### LT-1: Threat Modeling
**Gap:** #16 (MEDIUM)
**Действие:** STRIDE-анализ для CMMS, Gatekeeper, P2P Gateway.
**Время:** 8 часов

### LT-2: Data Classification
**Gap:** #3 (MEDIUM)
**Действие:** Внедрить метки классификации в БД.
**Время:** 4 часа

### LT-3: JWT → HttpOnly Cookies
**Gap:** #6 (MEDIUM)
**Действие:** Перевести аутентификацию на HttpOnly cookies с CSRF-токеном.
**Время:** 8 часов

### LT-4: Security Alerts
**Gap:** #13 (LOW)
**Действие:** Настроить алерты на подозрительную активность в audit log.
**Время:** 4 часа

### LT-5: Access Review Automation
**Gap:** A.9.2.5 (MEDIUM)
**Действие:** Автоматический quarterly review прав доступа.
**Время:** 4 часа

---

## Phase 4: Certification (Месяц 4-6)

- Внешний пентест
- Аудит на соответствие ISO 27001
- Сертификационный аудит
- Continuous monitoring

---

## Дорожная карта

```
Phase 1 (QW)     Phase 2 (MT)     Phase 3 (LT)     Phase 4 (Cert)
│                │                │                │
├─ JWT secret    ├─ Sec Policy    ├─ Threat Model  ├─ PenTest
├─ bcrypt        ├─ Asset Class   ├─ Data Class    ├─ Audit
├─ AES tokens    ├─ Log Integrity ├─ HttpOnly      ├─ Cert
├─ Config env    ├─ CORS          ├─ Alerts        └─ Monitor
├─ Rate limit    ├─ IR Plan       └─ Access Review
└─ Sec headers   └─ CI/CD Vuln
```

**Всего:** 17 gaps → 17 remediations