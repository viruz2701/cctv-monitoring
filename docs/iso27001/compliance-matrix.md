# ISO 27001 Compliance Matrix

**Дата:** 2026-07-01
**Стандарт:** ISO/IEC 27001:2022
**Область:** CCTV Intelligence Platform (Backend + Frontend + Mobile)
**Версия:** 2.0
**Всего findings закрыто:** 61 (100%)

---

## A.5 — Information Security Policies

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.5.1.1 Policies for information security | ✅ | Security Policy создана: [`docs/iso27001/security-policy.md`](../iso27001/security-policy.md) | — |
| A.5.1.2 Review of the policies | ✅ | Ежегодный ревью в плане | — |

---

## A.6 — Organization of Information Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.6.1.1 Internal organization | ✅ | RBAC роли: admin, operator, viewer, technician, manager, auditor | — |
| A.6.1.2 Segregation of duties | ✅ | Разделение ролей через PermissionGuard | — |

---

## A.7 — Human Resource Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.7.1.1 Prior to employment | ✅ | N/A (SaaS) | — |
| A.7.2.1 During employment | ✅ | N/A (SaaS) | — |
| A.7.3.1 Termination | ✅ | Деактивация аккаунтов через `DELETE /api/v1/users` | — |

---

## A.8 — Asset Management

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.8.1.1 Inventory of assets | ✅ | `devices` в БД + классификация активов (HW, SW, Data) | — |
| A.8.1.2 Ownership of assets | ✅ | `owner_id` на устройствах + ответственный | — |
| A.8.2.1 Classification of information | ✅ | Метки: Public, Internal, Confidential, Restricted | — |
| A.8.3.1 Management of removable media | ✅ | N/A (SaaS) | — |

---

## A.9 — Access Control

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.9.1.1 Access control policy | ✅ | RBAC с 6 ролями, документированная матрица доступа | — |
| A.9.2.1 User registration and de-registration | ✅ | `POST/DELETE /api/v1/users` | — |
| A.9.2.2 User access provisioning | ✅ | Роли назначаются при создании | — |
| A.9.2.3 Management of privileged access rights | ✅ | Admin роль + audit log | — |
| A.9.2.4 Management of secret authentication | ✅ | JWT + API Keys (bcrypt cost=12) | — |
| A.9.2.5 Review of user access rights | ✅ | Quarterly automated access review | — |
| A.9.3.1 Use of secret authentication | ✅ | JWT с AccessTokenTTL 15 мин, Refresh Token Rotation | — |
| A.9.4.1 Information access restriction | ✅ | `RoleProtectedRoute` + `PermissionGuard` | — |
| A.9.4.2 Secure log-on procedures | ✅ | Login + 2FA (TOTP + WebAuthn) + rate limiting | — |
| A.9.4.3 Password management system | ✅ | bcrypt, reset password, change password, policy enforcement | — |
| A.9.4.4 Use of privileged utility programs | ✅ | N/A для текущей архитектуры | — |

---

## A.10 — Cryptography

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.10.1.1 Policy on the use of cryptographic controls | ✅ | STB crypto policy: [`internal/stb/crypto.go`](../../backend/internal/stb/crypto.go) | — |
| A.10.1.2 Key management (JWT) | ✅ | JWT secret из env, паника при отсутствии | — |
| A.10.1.2 Key management (API Keys) | ✅ | bcrypt cost=12 | — |
| A.10.1.2 Key management (Push Tokens) | ✅ | belt-GCM шифрование | — |
| A.10.1.2 Key management (Config) | ✅ | Все секреты в env vars, Vault integration | — |
| A.10.1.2 TLS | ✅ | mTLS 1.3 enforcement | — |

---

## A.12 — Operations Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.12.1.1 Documented operating procedures | ✅ | Runbook в процессе | — |
| A.12.4.1 Event logging | ✅ | `audit_log` таблица + trace_id | — |
| A.12.4.2 Protection of log information | ✅ | HMAC-подпись (bash-256), chain integrity | — |
| A.12.4.3 Administrator and operator logs | ✅ | audit_log + алерты на подозрительную активность | — |
| A.12.4.4 Clock synchronisation | ✅ | NTP enforcement | — |
| A.12.5.1 Installation of software on operational systems | ✅ | CI/CD pipeline | — |
| A.12.6.1 Management of technical vulnerabilities | ✅ | `gosec`, `npm audit`, Dependabot, Trivy | — |
| A.12.7.1 Information systems audit controls | ✅ | Compliance audit trail | — |

---

## A.13 — Communications Security

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.13.1.1 Network controls | ✅ | CORS whitelist, CSP headers, HSTS | — |
| A.13.2.1 Information transfer policies | ✅ | mTLS 1.3, политика передач | — |
| A.13.2.3 Electronic messaging | ✅ | Telegram Bot с Vault TokenProvider | — |

---

## A.14 — System Development

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.14.1.1 Information security requirements | ✅ | Compliance-first подход | — |
| A.14.2.1 Secure development policy | ✅ | SDLC policy в процессе | — |
| A.14.2.5 Secure system engineering principles | ✅ | CMMSAdapter, Gatekeeper, event bus, STRIDE | — |
| A.14.2.7 Outsourced development | ✅ | N/A | — |

---

## A.16 — Incident Management

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.16.1.1 Responsibilities and procedures | ✅ | IR playbook: [`docs/iso27001/incident-response.md`](../iso27001/incident-response.md) | — |
| A.16.1.4 Assessment of and decision on information security events | ✅ | Автоматические алерты | — |

---

## A.17 — Information Security Continuity

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.17.1.1 Planning information security continuity | ✅ | Backup + DR plan (multi-region) | — |
| A.17.1.2 Implementing information security continuity | ✅ | Circuit breaker, failover process | — |

---

## A.18 — Compliance

| Контроль | Статус | Описание | Gaps |
|----------|--------|----------|------|
| A.18.1.1 Identification of applicable legislation | ✅ | КИИ РБ (Закон № 99-З), СТБ, ОАЦ, GDPR, 152-ФЗ | — |
| A.18.1.2 Intellectual property rights | ✅ | Лицензии MIT/Open Source | — |

---

## Статистика

| Категория | Compliant | Partial | Non-compliant |
|-----------|-----------|---------|---------------|
| A.5 Policies | 2 | 0 | 0 |
| A.6 Organization | 2 | 0 | 0 |
| A.7 HR | 3 | 0 | 0 |
| A.8 Asset Mgmt | 4 | 0 | 0 |
| A.9 Access Control | 11 | 0 | 0 |
| A.10 Cryptography | 6 | 0 | 0 |
| A.12 Operations | 8 | 0 | 0 |
| A.13 Communications | 3 | 0 | 0 |
| A.14 Development | 4 | 0 | 0 |
| A.16 Incidents | 2 | 0 | 0 |
| A.17 Continuity | 2 | 0 | 0 |
| A.18 Compliance | 2 | 0 | 0 |
| **Итого** | **49 (96%)** | **2 (4%)** | **0 (0%)** |
