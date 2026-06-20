# ISO 27001 Compliance Matrix

**Дата:** 2026-06-20
**Стандарт:** ISO/IEC 27001:2022
**Область:** CCTV Intelligence Platform (Backend + Frontend + Mobile)

---

## A.5 — Information Security Policies

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.5.1.1 Policies for information security | ❌ | Нет документированной Security Policy | Создать `docs/iso27001/security-policy.md` |
| A.5.1.2 Review of the policies | ❌ | Нет процесса ревью | Внедрить ежегодный ревью |

---

## A.8 — Asset Management

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.8.1.1 Inventory of assets | ⚠️ | Есть Devices в БД, но нет CMDB | Добавить классификацию активов (HW, SW, Data) |
| A.8.1.2 Ownership of assets | ⚠️ | `owner_id` на устройствах | Добавить ответственного для всех активов |
| A.8.2.1 Classification of information | ❌ | Нет классификации данных | Внедрить метки: Public, Internal, Confidential, Secret |
| A.8.3.1 Management of removable media | ❌ | Не применимо (SaaS) | N/A |

---

## A.9 — Access Control

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.9.1.1 Access control policy | ⚠️ | RBAC с 6 ролями (admin, operator, viewer, technician, manager, auditor) | Документировать матрицу доступа |
| A.9.2.1 User registration and de-registration | ✅ | `POST/DELETE /api/v1/users` | |
| A.9.2.2 User access provisioning | ✅ | Роли назначаются при создании | |
| A.9.2.3 Management of privileged access rights | ⚠️ | Admin роль существует | Нет audit log для админских действий |
| A.9.2.4 Management of secret authentication | ⚠️ | JWT + API Keys | API Keys: SHA-256 вместо bcrypt (см. A.10) |
| A.9.2.5 Review of user access rights | ❌ | Нет автоматического ревью | Добавить quarterly access review |
| A.9.3.1 Use of secret authentication | ⚠️ | JWT в localStorage | Рекомендовать HttpOnly cookies |
| A.9.4.1 Information access restriction | ✅ | RoleProtectedRoute + PermissionGuard | |
| A.9.4.2 Secure log-on procedures | ⚠️ | Login + 2FA (TOTP) | Добавить rate limiting на login |
| A.9.4.3 Password management system | ⚠️ | Change password, reset password | Минимальные требования к паролю не enforce |
| A.9.4.4 Use of privileged utility programs | ❌ | Нет ограничений | N/A для текущей архитектуры |

---

## A.10 — Cryptography

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.10.1.1 Policy on the use of cryptographic controls | ❌ | Нет политики | Создать crypto policy |
| A.10.1.2 Key management | ⚠️ | [`jwt.go`](../../backend/internal/auth/jwt.go) — JWT secret из env/дефолтный | **КРИТИЧЕСКИ:** Дефолтный JWT secret в коде (`dev-secret-key-change-in-production-immediately`) |
| A.10.1.2 Key management (API Keys) | ⚠️ | [`apikey_handlers.go`](../../backend/internal/api/apikey_handlers.go) — SHA-256 | **КРИТИЧЕСКИ:** SHA-256 вместо bcrypt/argon2 для API-ключей |
| A.10.1.2 Key management (Push Tokens) | ❌ | [`cmms_repository.go`](../../backend/internal/db/cmms_repository.go) — `SavePushToken` plaintext | **КРИТИЧЕСКИ:** Push-токены в открытом виде |
| A.10.1.2 Key management (Config) | ❌ | [`config.yaml`](../../backend/config.yaml) — пароли в открытом виде | `p2p_api_key`, FTP пароль, Hikvision пароли |
| A.10.1.2 TLS | ⚠️ | CORS `*` разрешён | Нет enforcement TLS 1.3 |

---

## A.12 — Operations Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.12.1.1 Documented operating procedures | ❌ | Нет документации | Создать runbook |
| A.12.4.1 Event logging | ✅ | [`logAudit()`](../../backend/internal/api/cmms_handlers.go) — audit_log таблица | |
| A.12.4.2 Protection of log information | ⚠️ | Логи в файл + БД | Нет log Integrity (подпись/хэш) |
| A.12.4.3 Administrator and operator logs | ⚠️ | Есть audit_log | Нет алертов на подозрительную активность |
| A.12.4.4 Clock synchronisation | ⚠️ | `time.Now()` | Нет NTP enforcement |
| A.12.5.1 Installation of software on operational systems | ❌ | Нет контроля | N/A |
| A.12.6.1 Management of technical vulnerabilities | ❌ | Нет scanning | Добавить `npm audit`, `go vet -vettool`, Dependabot |
| A.12.7.1 Information systems audit controls | ❌ | Нет audit controls | |

---

## A.13 — Communications Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.13.1.1 Network controls | ⚠️ | CORS `*`, нет WAF | Ограничить CORS, добавить CSP headers |
| A.13.2.1 Information transfer policies | ❌ | Нет политики | |
| A.13.2.3 Electronic messaging | ⚠️ | Telegram Bot | Нет шифрования сообщений |

---

## A.14 — System Development

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.14.1.1 Information security requirements | ❌ | Нет security requirements | |
| A.14.2.1 Secure development policy | ❌ | Нет SDLC policy | |
| A.14.2.5 Secure system engineering principles | ⚠️ | CMMSAdapter — хорошая архитектура | Нет threat modeling |
| A.14.2.7 Outsourced development | ❌ | N/A | |

---

## A.16 — Incident Management

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.16.1.1 Responsibilities and procedures | ❌ | Нет incident response plan | Создать IR playbook |
| A.16.1.4 Assessment of and decision on information security events | ❌ | Нет процесса | |

---

## Статистика

| Категория | Compliant | Partial | Non-compliant |
|-----------|-----------|---------|---------------|
| A.5 Policies | 0 | 0 | 2 |
| A.8 Asset Mgmt | 0 | 2 | 2 |
| A.9 Access Control | 1 | 5 | 1 |
| A.10 Cryptography | 0 | 3 | 3 |
| A.12 Operations | 1 | 3 | 3 |
| A.13 Communications | 0 | 1 | 2 |
| A.14 Development | 0 | 1 | 3 |
| A.16 Incidents | 0 | 0 | 2 |
| **Итого** | **2 (6%)** | **15 (47%)** | **15 (47%)** |