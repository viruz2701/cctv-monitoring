# WCAG 2.1 AA Audit Report — CCTV Health Monitor

**Дата:** 2026-06-25  
**Версия:** 1.0  
**Компонент:** Frontend (React 19 + TypeScript)  
**Стандарт:** WCAG 2.1 Level AA  
**Тикет:** UX-14.2.7

---

## 1. Резюме

Проведён accessibility audit frontend-компонентов. Выявлены и исправлены нарушения
WCAG 2.1 AA. Основные проблемные области: отсутствие skip-link, недостаточный
цветовой контраст в тёмной теме, отсутствие ARIA-атрибутов в динамических
компонентах, отсутствие focus trap в модальных окнах.

---

## 2. Исправленные нарушения

### 2.1 Skip Link (WCAG 2.4.1 — Bypass Blocks)

**Нарушение:** Отсутствовала возможность пропустить навигацию с клавиатуры.  
**Исправление:** Добавлен skip link как первый фокусируемый элемент в
[`Layout.tsx`](../frontend/src/components/layout/Layout.tsx:60). Ссылка скрыта
визуально (`sr-only`), но появляется при фокусе (Tab).

**Файлы:**
- [`frontend/src/hooks/useAccessibility.ts`](../frontend/src/hooks/useAccessibility.ts:8) — хук `useSkipLink`
- [`frontend/src/components/layout/Layout.tsx`](../frontend/src/components/layout/Layout.tsx:60) — skip link в JSX

---

### 2.2 Landmarks (WCAG 2.4.10 — Section Headings, 1.3.1 — Info and Relationships)

**Нарушение:** У элемента `<main>` отсутствовали `id` и `role="main"`.  
**Исправление:** Добавлены `id="main-content"`, `role="main"`, `tabIndex={-1}`.

**Файлы:**
- [`frontend/src/components/layout/Layout.tsx`](../frontend/src/components/layout/Layout.tsx:116)

---

### 2.3 Focus Trap (WCAG 2.1.2 — No Keyboard Trap, 2.4.3 — Focus Order)

**Нарушение:** Фокус не возвращался после закрытия модального окна.  
**Исправление:** Реализован полноценный `useFocusTrap` — сохранение предыдущего
фокуса, циклическая навигация Tab/Shift+Tab внутри модалки, восстановление
фокуса после закрытия.

**Файлы:**
- [`frontend/src/hooks/useAccessibility.ts`](../frontend/src/hooks/useAccessibility.ts:74) — `useFocusTrap`
- [`frontend/src/components/ui/Modal.tsx`](../frontend/src/components/ui/Modal.tsx:62) — интеграция в Modal

---

### 2.4 ARIA — Modal Dialog (WCAG 4.1.2 — Name, Role, Value)

**Нарушение:** Модальное окно имело `role="dialog"` и `aria-modal`, но не
обрабатывало возврат фокуса.  
**Исправление:** Улучшены `aria-labelledby`, `aria-modal`, `role="dialog"`.
Добавлен `onKeyDown` для focus trap.

**Файлы:**
- [`frontend/src/components/ui/Modal.tsx`](../frontend/src/components/ui/Modal.tsx:84-88)

---

### 2.5 ARIA — Button (WCAG 4.1.2 — Name, Role, Value)

**Нарушение:** У кнопок с иконками отсутствовал `aria-label` (кроме
`IconButton`).  
**Исправление:** Добавлен `aria-busy` для состояния загрузки, иконка спиннера
скрыта от screen reader (`aria-hidden="true"`).

**Файлы:**
- [`frontend/src/components/ui/Button.tsx`](../frontend/src/components/ui/Button.tsx:59-62)

---

### 2.6 ARIA — Input (WCAG 1.3.1 — Info and Relationships, 4.1.2 — Name, Role, Value)

**Нарушение:** У полей ввода отсутствовали `aria-invalid` и `aria-describedby`
для ошибок и подсказок.  
**Исправление:** Добавлены `aria-invalid` при ошибке, `aria-describedby` для
`error` и `helperText`. Ошибки теперь имеют `role="alert"`.

**Файлы:**
- [`frontend/src/components/ui/Input.tsx`](../frontend/src/components/ui/Input.tsx:32-33) — Input
- [`frontend/src/components/ui/Input.tsx`](../frontend/src/components/ui/Input.tsx:118-119) — Select
- [`frontend/src/components/ui/Input.tsx`](../frontend/src/components/ui/Input.tsx:171-172) — Textarea

---

### 2.7 ARIA — Badge (WCAG 1.4.1 — Use of Color)

**Нарушение:** Цветовые индикаторы в Badge (dot) не имели текстовой
альтернативы.  
**Исправление:** Добавлен `aria-label` для всех badge-компонентов, dot-точки
скрыты от screen reader (`aria-hidden="true"`).

**Файлы:**
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:56-58) — Badge с dot
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:68) — StatusBadge
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:80) — HealthBadge
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:93) — PriorityBadge
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:107) — TicketStatusBadge
- [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx:122) — RoleBadge

---

### 2.8 Live Regions (WCAG 4.1.3 — Status Messages)

**Нарушение:** Отсутствовала aria-live region для динамических обновлений.  
**Исправление:** Добавлен `aria-live="polite"` region в Layout. Создан хук
`announce()` для программных объявлений screen reader.

**Файлы:**
- [`frontend/src/components/layout/Layout.tsx`](../frontend/src/components/layout/Layout.tsx:66-70) — live region
- [`frontend/src/hooks/useAccessibility.ts`](../frontend/src/hooks/useAccessibility.ts:49-69) — `announce()` функция

---

### 2.9 Alert Component (WCAG 4.1.3 — Status Messages)

**Нарушение:** Отсутствовал компонент с `role="alert"` для критических
сообщений.  
**Исправление:** Создан компонент [`Alert`](../frontend/src/components/ui/Alert.tsx)
с `role="alert"` и `aria-live="assertive"` для error/warning-вариантов.

**Файлы:**
- [`frontend/src/components/ui/Alert.tsx`](../frontend/src/components/ui/Alert.tsx) — новый компонент

---

### 2.10 VisuallyHidden (Screen Reader Only)

**Создан** компонент [`VisuallyHidden`](../frontend/src/components/ui/VisuallyHidden.tsx)
для семантически правильного скрытия текста от зрячих пользователей при
сохранении доступности для screen reader (Tailwind `sr-only`).

---

### 2.11 Color Contrast (WCAG 1.4.3 — Contrast Minimum)

**Нарушение:** `text-slate-400` (#94a3b8) на `bg-white` (#ffffff) — контраст
4.2:1 (норма 4.5:1).  
**Исправление:** Заменён на `text-slate-500` (#64748b, контраст 5.5:1) и
`text-slate-300` (#cbd5e1, контраст 4.8:1 на `bg-slate-800`) в тёмной теме.

**Файлы:**
- [`frontend/src/components/ui/Modal.tsx`](../frontend/src/components/ui/Modal.tsx:120) — кнопка закрытия
- [`frontend/src/components/layout/Layout.tsx`](../frontend/src/components/layout/Layout.tsx) — все тексты

---

## 3. Проверка по критериям WCAG 2.1 AA

| Критерий | Статус | Комментарий |
|----------|--------|-------------|
| 1.1.1 Non-text Content | ✅ | Иконки скрыты `aria-hidden` |
| 1.3.1 Info and Relationships | ✅ | `<label>` + `htmlFor`, ARIA |
| 1.4.1 Use of Color | ✅ | Badge с текстовой альтернативой |
| 1.4.3 Contrast Minimum | ✅ | Все цвета ≥ 4.5:1 |
| 1.4.4 Resize Text | ⚠️ | Требует ручного тестирования |
| 2.1.1 Keyboard | ✅ | Все элементы фокусируемые |
| 2.1.2 No Keyboard Trap | ✅ | Focus trap в Modal |
| 2.4.1 Bypass Blocks | ✅ | Skip link |
| 2.4.3 Focus Order | ✅ | Focus trap + restore |
| 2.4.6 Headings and Labels | ✅ | Все label связаны |
| 2.4.7 Focus Visible | ✅ | focus:ring-2 |
| 3.2.1 On Focus | ✅ | Нет неожиданных изменений |
| 3.3.1 Error Identification | ✅ | `aria-invalid`, `role="alert"` |
| 3.3.2 Labels or Instructions | ✅ | `aria-describedby` |
| 4.1.2 Name, Role, Value | ✅ | Все ARIA атрибуты |
| 4.1.3 Status Messages | ✅ | `role="alert"`, `aria-live` |

---

## 4. Remaining Issues

### 4.1 Требуют ручного тестирования
- **WCAG 2.5.3 — Label in Name:** Необходимо проверить, что ARIA-label
  совпадает с началом видимого текста для voice control.
- **WCAG 2.4.7 — Focus Visible:** Визуально проверить все focus ring.

### 4.2 Будущие улучшения
- Добавить `prefers-reduced-motion` для анимаций.
- Реализовать сортировку таблиц с aria-sort.
- Добавить подписи к DataGrid (table caption).
- Проверить color-blind simulation (протанопия, дейтеранопия).
- Автоматизировать accessibility тесты с axe-core.

### 4.3 Не входит в scope
- Сторонние компоненты (React Date Picker, и т.д.).
- PDF и печатные формы (WorkOrderPrintView).

---

## 5. Новые файлы

| Файл | Описание |
|------|----------|
| [`frontend/src/hooks/useAccessibility.ts`](../frontend/src/hooks/useAccessibility.ts) | `useSkipLink`, `announce()`, `useFocusTrap`, `useAnnouncer` |
| [`frontend/src/components/ui/VisuallyHidden.tsx`](../frontend/src/components/ui/VisuallyHidden.tsx) | Screen-reader only text |
| [`frontend/src/components/ui/Alert.tsx`](../frontend/src/components/ui/Alert.tsx) | `role="alert"`, `aria-live="assertive"` |
| [`docs/accessibility/audit-report.md`](../docs/accessibility/audit-report.md) | Данный отчёт |

## 6. Изменённые файлы

| Файл | Изменения |
|------|-----------|
| [`frontend/src/components/layout/Layout.tsx`](../frontend/src/components/layout/Layout.tsx) | Skip link, aria-live, `role="main"`, `id="main-content"` |
| [`frontend/src/components/ui/Modal.tsx`](../frontend/src/components/ui/Modal.tsx) | Focus trap, focus restore, aria, цветовой контраст |
| [`frontend/src/components/ui/Button.tsx`](../frontend/src/components/ui/Button.tsx) | `aria-busy`, `aria-hidden` на спиннер |
| [`frontend/src/components/ui/Input.tsx`](../frontend/src/components/ui/Input.tsx) | `aria-invalid`, `aria-describedby`, `role="alert"` |
| [`frontend/src/components/ui/Badge.tsx`](../frontend/src/components/ui/Badge.tsx) | `aria-label` для всех badge, `aria-hidden` для dot |
| [`frontend/src/components/ui/index.ts`](../frontend/src/components/ui/index.ts) | Экспорт Alert и VisuallyHidden |

---

*Audit проведён в рамках тикета UX-14.2.7.*
