# ADR-005: State Management Strategy

## Status

Accepted

## Context

При разработке CCTV Health Monitor накопилось несколько подходов к управлению состоянием:

1. **React Context** — использовался для server state (DevicesSites, Alerts, Tickets, WorkOrders, SpareParts, Maintenance, Notifications, Users)
2. **React Query** — постепенно внедрялся для server state (`useApiQuery.ts`), но Context'ы оставались как bridge-слой
3. **Zustand** — для UI state (theme, alert filters, command palette, onboarding, workspace, filters)
4. **Local useState + Zod** — для форм

Проблемы текущего подхода:
- Context'ы вызывают cascading re-renders при любом изменении данных
- Дублирование кода — каждый Context дублирует API вызовы, маппинг и типы
- Усложнение дерева компонентов — 10+ вложенных Provider'ов в `App.tsx`
- Нет чёткого разделения ответственности между Context и React Query

## Decision

### 1. Серверное состояние (Server State) → React Query

Все данные, получаемые с API, управляются через React Query:

| Данные | Хук React Query | Источник |
|--------|----------------|----------|
| Devices | `useDevices()` | `api.getDevices()` |
| Sites | `useSites()` | `api.getSites()` |
| Tickets | `useTickets()` | `api.getTickets()` |
| Alarms | `useAlarms()` | `api.getAlarms()` |
| Work Orders | `useWorkOrders()` | `workOrdersApi.getWorkOrders()` |
| Spare Parts | `useSpareParts()` | `sparePartsApi.getSpareParts()` |
| Maintenance | `useMaintenanceSchedules()` | `maintenanceApi.getSchedules()` |
| Notifications | `useNotifications()` | `api.getNotifications()` |
| Users | `useUsers()` | `api.getUsers()` |
| Reports (list) | `useReports()` | `api.getReports()` |

Преимущества:
- Автоматический кэш, stale-while-revalidate, refetchOnFocus, retry
- Изоляция данных — каждый компонент подписывается только на свои query keys
- Мутации с автоматической инвалидацией кэша
- Prefetching для навигации

### 2. UI-состояние (Client State) → Zustand

Состояние, которое не требует серверной синхронизации:

| Состояние | Store | Описание |
|-----------|-------|----------|
| Тема | `themeStore` | theme, isDark, accentColor |
| Алерты/фильтры | `alertStore` | filter, selected |
| Палитра команд | `commandPaletteStore` | isOpen, query |
| Онбординг | `onboardingStore` | currentStep, dismissed |
| Рабочее пространство | `workspaceStore` | activeWorkspace, layouts |
| Фильтры | `filterStore` | active filters |

### 3. Сессионное состояние (Session State) → Context

Только для критического сессионного состояния:

| Состояние | Provider | Описание |
|-----------|----------|----------|
| Auth | `AuthProvider` | user, token, login/logout |
| Тема | `ThemeProvider` | bridge to Zustand (backward compat) |
| Настройки | `SettingsProvider` | app settings + services (mixed state) |

### 4. Формы (Form State) → local useState + Zod

Каждая форма использует локальное состояние с валидацией через Zod:
- `useFormValidation.ts` — централизованный хук
- Zod schemas в `lib/validations.ts`

## Consequences

### Positive
- ✅ Устранение cascading re-renders от Context'ов
- ✅ Чёткое разделение server/client state
- ✅ Меньше boilerplate — React Query mutations с автоматической инвалидацией
- ✅ Оптимизация производительности — select-подписки в Zustand
- ✅ Предсказуемый lifecycle данных

### Negative
- ❌ Необходимость миграции существующих страниц
- ❌ Временная путаница — два подхода в кодебейсе

### Mitigation
- Миграция проводится в рамках P1-5
- После миграции — удаление всех bridge-Context'ов
- ESLint правило: запрет импорта из `context/` (кроме Theme, Settings, DataContext)

## Compliance Notes

- **IEC 62443 SR 7.1** — Resource availability через async data fetching (React Query)
- **OWASP ASVS V1.8** — Stateless design для server state
- **ISO 27001 A.12.4** — Audit trail через query keys (traceable data access)

## References

- [ADR-001: Headless CMMS](./ADR-001-headless-cmms.md)
- [React Query documentation](https://tanstack.com/query/latest)
- [Zustand documentation](https://github.com/pmndrs/zustand)
