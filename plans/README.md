# 📋 Plans Directory

> **Индекс всех планов проекта CCTV Health Monitor**
>
> Последнее обновление: 2026-07-01

---

## Список планов

| # | Файл | Статус | Дата | Описание |
|---|------|--------|------|----------|
| 1 | [`code-review-epic1-2026-06-24.md`](./code-review-epic1-2026-06-24.md) | ✅ **DONE** | 2026-06-24 | Code Review Epic 1 — 61 finding (6 critical, 7 warnings, 9 positive, архитектурные замечания). Все findings закрыты. |
| 2 | [`2026-06-25-SEC02-fix-panics.md`](./2026-06-25-SEC02-fix-panics.md) | ✅ **DONE** | 2026-06-25 | SEC-02: Замена `panic()` на `error` в production-коде. JWT secret, CSP nonce, health check 503. |
| 3 | [`2026-06-25-UX01-mobile-completion-wizard.md`](./2026-06-25-UX01-mobile-completion-wizard.md) | ✅ **DONE** | 2026-06-25 | UX-01: Mobile Work Order Completion Wizard — сокращение с 6 экранов до 3-step wizard. |
| 4 | [`2026-06-27-p1-implementation.md`](./2026-06-27-p1-implementation.md) | ✅ **DONE** | 2026-06-27 | P1 Implementation — 15+ задач: backend core tests, playbook/RCA, frontend architecture, features. |
| 5 | [`map-iframe-plan.md`](./map-iframe-plan.md) | ✅ **DONE** | 2026-06-27 | iframe-карта вместо `window.open` для OpenStreetMap. MapModal компонент + интеграция. |

---

## Статус выполнения

```
✅ DONE: 5/5 планов выполнено
```

Все планы, документированные в этой директории, полностью реализованы. Финальное подтверждение:

- Коммит [`d937dea`](https://github.com/viruz2701/cctv-monitoring/commit/d937dea) — **61/61 ✅ ALL DONE**
- Коммит [`13eb693`](https://github.com/viruz2701/cctv-monitoring/commit/13eb693) — **TODO.md mark ALL sections DONE**

---

## Структура файлов

```
plans/
├── README.md                          ← этот файл (индекс)
├── code-review-epic1-2026-06-24.md    ← code review 61 finding
├── 2026-06-25-SEC02-fix-panics.md     ← SEC-02: panic→error
├── 2026-06-25-UX01-mobile-completion-wizard.md  ← UX-01: wizard
├── 2026-06-27-p1-implementation.md    ← P1: 15+ задач
└── map-iframe-plan.md                 ← iframe-карта
```

---

## Связанные артефакты

- [`TODO.md`](../TODO.md) — главный трекер задач проекта (61/61 ✅)
- [`backend/`](../backend/) — Go-бэкенд
- [`frontend/`](../frontend/) — React-фронтенд
- [`mobile/`](../mobile/) — React Native мобильное приложение
