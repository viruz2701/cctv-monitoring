# План: iframe-карта вместо нового окна

## Проблема
При нажатии "View on map" на [`Sites.tsx:600`](../frontend/src/pages/Sites.tsx:600) открывается OpenStreetMap в новом окне браузера (`window.open(..., '_blank')`). Пользователь хочет видеть карту как всплывающий iframe внутри приложения.

## Аналогичные вызовы window.open
- [`Sites.tsx:600`](../frontend/src/pages/Sites.tsx:600) — `window.open(...)` карта
- [`DeviceDetail.tsx:399`](../frontend/src/pages/DeviceDetail.tsx:399) — `window.open(...)` QR-заявка
- [`BlackBox.tsx:298`](../frontend/src/pages/BlackBox.tsx:298) — экспорт отчёта
- [`NotificationsSettings.tsx:344`](../frontend/src/pages/NotificationsSettings.tsx:344) — Telegram
- [`SecuritySettings.tsx:362`](../frontend/src/pages/SecuritySettings.tsx:362) — Telegram

Пока исправляем только карту. Остальные — при необходимости.

## Компоненты для создания

### 1. `MapModal` — переиспользуемый модал с iframe карты

```
frontend/src/components/ui/MapModal.tsx
```

**Props:**
```typescript
interface MapModalProps {
  isOpen: boolean;
  onClose: () => void;
  latitude: number;
  longitude: number;
  title?: string;
}
```

**Реализация:**
- Использует существующий [`Modal`](../frontend/src/components/ui/Modal.tsx) компонент
- Внутри — `<iframe>` на `openstreetmap.org/export/embed.html`
- Кнопка "Open in new tab" для открытия в отдельном окне (опционально)
- Размер: `w-[90vw] h-[80vh]` или `max-w-4xl`

**URL для iframe:**
```
https://www.openstreetmap.org/export/embed.html?bbox={lon-0.01},{lat-0.01},{lon+0.01},{lat+0.01}&layer=mapnik&marker={lat},{lon}
```

### 2. Интеграция в Sites.tsx

**Изменения:**
- Добавить `import { MapModal } from '../components/ui/MapModal'`
- Добавить `const [showMap, setShowMap] = useState(false)`
- Заменить `window.open(...)` на `setShowMap(true)`
- Рендерить `<MapModal isOpen={showMap} onClose={() => setShowMap(false)} latitude={...} longitude={...} />`

## Схема работы

```
[Sites.tsx Form]
    │
    ├── Пользователь вводит координаты
    │
    ├── [Кнопка "View on map"]
    │       │ (onClick → setShowMap(true))
    │       ▼
    └── [MapModal] (всплывающий слой)
            │
            ├── <iframe src="openstreetmap.org/export/embed.html?bbox=...&marker=...">
            │       └── Карта с маркером
            │
            ├── [Кнопка "Закрыть"]
            │       └── setShowMap(false)
            │
            └── [Ссылка "Open in new tab"]
                    └── window.open(...) как fallback
```

## Проверки и комплаенс

- `iframe` загружается только когда `isOpen === true` (ленивая загрузка)
- CSP уже разрешает `img-src 'self' data: https:` и `connect-src 'self' https://nominatim.openstreetmap.org` (см. [`vite.config.ts`](../frontend/vite.config.ts:13))
- Для OSM embed нужно добавить `frame-src https://www.openstreetmap.org` в CSP
- fallback `window.open` сохраняется для пользователей с блокировщиками iframe

## Файлы для изменения

| Файл | Действие |
|------|----------|
| `frontend/src/components/ui/MapModal.tsx` | **Создать** — компонент модала с iframe |
| `frontend/src/pages/Sites.tsx` | **Изменить** — заменить `window.open` на `MapModal` |
| `frontend/vite.config.ts` | **Изменить** — добавить `frame-src` в CSP |
