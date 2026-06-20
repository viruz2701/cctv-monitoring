# ISO 27001 Gap Analysis — CCTV Intelligence Platform

**Дата:** 2026-06-20
**Аудитор:** Phase 0 Research
**Методология:** Анализ кода бэкенда, конфигурации и архитектуры

---

## 1. A.5 — Information Security Policies

### Gap 1: Отсутствует Security Policy
**Серьёзность:** HIGH
**Описание:** Нет документированной политики информационной безопасности.
**Рекомендация:** Создать `docs/iso27001/security-policy.md` с разделами:
- Scope и Objectives
- Roles and Responsibilities (CISO, Admin, Developer)
- Acceptable Use Policy
- Data Classification Policy
- Incident Response Procedure

---

## 2. A.8 — Asset Management

### Gap 2: Нет CMDB
**Серьёзность:** MEDIUM
**Описание:** Устройства есть в БД, но нет классификации активов (HW, SW, Data, People).
**Рекомендация:** Добавить поле `asset_class` в таблицу `devices`:
- `hardware` — камеры, NVR, серверы
- `software` — лицензии, прошивки
- `data` — записи, конфигурации
- `network` — коммутаторы, маршрутизаторы

### Gap 3: Нет классификации данных
**Серьёзность:** MEDIUM
**Описание:** Все данные хранятся одинаково, без классификации.
**Рекомендация:** Внедрить метки:
- `public` — статистика, документация
- `internal` — служебная информация
- `confidential` — PII (push tokens, emails), пароли
- `restricted` — JWT secrets, API keys, ключи шифрования

---

## 3. A.9 — Access Control

### Gap 4: SHA-256 вместо bcrypt для API-ключей
**Серьёзность:** CRITICAL
**Файл:** [`apikey_handlers.go`](../../backend/internal/api/apikey_handlers.go)
**Описание:** API-ключи хэшируются SHA-256 — не предназначен для хранения паролей/ключей.
**Рекомендация:** Заменить на bcrypt (cost factor 12):
```go
hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), 12)
```

### Gap 5: Нет rate limiting на login
**Серьёзность:** HIGH
**Файл:** [`server.go`](../../backend/internal/api/server.go) — `handleLogin`
**Описание:** Нет защиты от brute-force атак на логин.
**Рекомендация:** Добавить rate limiter middleware (5 попыток в минуту на IP).

### Gap 6: JWT в localStorage
**Серьёзность:** MEDIUM
**Описание:** Frontend хранит JWT в localStorage — уязвим к XSS.
**Рекомендация:** Использовать HttpOnly cookies для JWT, либо добавить fingerprint.

---

## 4. A.10 — Cryptography

### Gap 7: Дефолтный JWT secret в коде
**Серьёзность:** CRITICAL
**Файл:** [`jwt.go`](../../backend/internal/auth/jwt.go), строка 22
**Описание:**
```go
return []byte("dev-secret-key-change-in-production-immediately")
```
**Рекомендация:** Убрать дефолтное значение. Если `JWT_SECRET` не задан — паниковать при старте.

### Gap 8: Push-токены в открытом виде
**Серьёзность:** CRITICAL
**Файл:** [`cmms_repository.go`](../../backend/internal/db/cmms_repository.go) — `SavePushToken`
**Описание:** Push-токены хранятся plaintext в таблице `users`:
```go
UPDATE users SET push_token = $1, push_platform = $2
```
**Рекомендация:** Шифровать AES-256-GCM перед сохранением. Ключ шифрования из env.

### Gap 9: Пароли в config.yaml
**Серьёзность:** HIGH
**Файл:** [`config.yaml`](../../backend/config.yaml)
**Описание:** `p2p_api_key`, FTP пароль, Hikvision пароли — в открытом виде.
**Рекомендация:** Использовать env vars или secrets manager (Vault).

### Gap 10: CORS `*` разрешён
**Серьёзность:** MEDIUM
**Файл:** [`server.go`](../../backend/internal/api/server.go), строка 57
**Описание:** `AllowedOrigins: []string{"*"}` — любой origin может делать запросы.
**Рекомендация:** Ограничить конкретными доменами в production.

---

## 5. A.12 — Operations Security

### Gap 11: Нет log integrity
**Серьёзность:** MEDIUM
**Описание:** Audit log пишется в БД, но нет защиты от подделки.
**Рекомендация:** Добавить HMAC-подпись для каждой audit-записи или использовать append-only log.

### Gap 12: Нет vulnerability scanning
**Серьёзность:** MEDIUM
**Описание:** Нет автоматического сканирования уязвимостей.
**Рекомендация:** Добавить в CI/CD:
- `npm audit` для frontend
- `gosec` для Go backend
- `trivy` для Docker images
- Dependabot для обновлений

### Gap 13: Нет алертов на подозрительную активность
**Серьёзность:** LOW
**Описание:** Audit log пишется, но не анализируется.
**Рекомендация:** Добавить алерты на:
- 5+ failed login attempts
- Admin-действия в нерабочее время
- Массовое создание/удаление пользователей

---

## 6. A.13 — Communications Security

### Gap 14: Нет CSP headers
**Серьёзность:** MEDIUM
**Описание:** Нет Content-Security-Policy, X-Frame-Options, X-Content-Type-Options.
**Рекомендация:** Добавить security headers middleware.

### Gap 15: Нет TLS enforcement
**Серьёзность:** MEDIUM
**Описание:** API слушает на `:8080` без TLS (предполагается reverse proxy).
**Рекомендация:** Документировать требование TLS termination на reverse proxy (nginx/Caddy).

---

## 7. A.14 — System Development

### Gap 16: Нет threat modeling
**Серьёзность:** MEDIUM
**Описание:** Архитектура хорошая, но нет формального threat model.
**Рекомендация:** Провести STRIDE-анализ для ключевых компонентов (CMMS, Gatekeeper, P2P Gateway).

---

## 8. A.16 — Incident Management

### Gap 17: Нет Incident Response Plan
**Серьёзность:** HIGH
**Описание:** Нет документированного плана реагирования на инциденты.
**Рекомендация:** Создать IR playbook:
- Detection (алерты, мониторинг)
- Containment (блокировка аккаунта, отзыв ключей)
- Eradication (исправление уязвимости)
- Recovery (восстановление из бэкапа)
- Lessons Learned (post-mortem)

---

## Итого: 17 Gaps

| Приоритет | Количество |
|-----------|-----------|
| CRITICAL | 3 (JWT secret, API Keys, Push Tokens) |
| HIGH | 4 (Security Policy, Rate Limiting, Config Secrets, IR Plan) |
| MEDIUM | 8 |
| LOW | 2 |