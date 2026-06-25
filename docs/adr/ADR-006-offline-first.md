# ADR-006: Offline-First Mobile Strategy

## Status
ACCEPTED (2026-06-25)

## Context

Мобильное приложение CCTV Technician работает в условиях нестабильной связи — подвалы, удалённые объекты, зоны без покрытия. Текущая реализация использует AsyncStorage для очереди синхронизации (`syncStore`) и React Query для кэширования, но:

1. **Нет структурированного локального хранилища** — все данные загружаются заново при каждом открытии
2. **Нет offline-доступа к данным** — при отсутствии сети приложение показывает пустые экраны
3. **AsyncStorage не подходит для структурированных данных** — ограничение 6MB, нет индексов, нет транзакций
4. **Нет conflict resolution** — последняя операция всегда перезаписывает предыдущую

### Рассмотренные решения

| Критерий | WatermelonDB | expo-sqlite | RxDB |
|---|---|---|---|
| **Уже в deps** | ❌ Нет | ✅ Да | ❌ Нет |
| **Размер бандла** | ~200KB | ~100KB | ~150KB |
| **Observed queries** | ✅ Lazily evaluated | ❌ Нет | ✅ Reactive |
| **Sync protocol** | ✅ Pull/push | ❌ Вручную | ✅ Pull/push + replication |
| **ACID transactions** | ✅ SQLite | ✅ SQLite | ✅ PouchDB |
| **Сложность** | Средняя | Низкая | Высокая |
| **Подходит для текущих задач** | Overkill | ✅ Достаточно | Overkill |
| **Миграции схем** | ✅ Авто | ✅ Вручную (простые) | ✅ Авто |

### Decision

**expo-sqlite + ручная реализация sync**

Обоснование:
- Уже в зависимостях (`expo-sqlite@^56.0.5`) — не увеличиваем бандл
- Достаточно для текущих задач: хранение work_orders, devices, sites
- ACID транзакции через SQLite
- Низкая сложность поддержки
- Если потребуется reactive sync — мигрируем на RxDB

### Conflict Resolution Strategy

**Last-Write-Wins (LWW)** с manual merge для сложных случаев:

1. Каждая запись имеет `updated_at` timestamp
2. При конфликте побеждает запись с более поздним `updated_at`
3. Если конфликт возник при sync — побеждает серверная версия (server-authoritative для data integrity)
4. Пользователь уведомляется о конфликте через UI

```
Client mutation (t1) ──► Pending Queue
                              │
Server mutation (t2) ◄──── Sync (t1 > t2 → client wins, else server wins)
```

## Solution

### 1. SQLite Database (`offlineStorage.ts`)

| Таблица | Назначение | Индексы |
|---|---|---|
| `work_orders` | Кэш нарядов-заказов | `id`, `status`, `updated_at` |
| `devices` | Кэш устройств | `id`, `status` |
| `sites` | Кэш объектов | `id` |
| `pending_sync` | Очередь мутаций для синхронизации | `id`, `timestamp`, `entity_type` |

### 2. Sync Service (`syncService.ts`)

Pull/push sync flow:
```
┌──────────┐     ┌─────────────┐     ┌──────────┐
│  SQLite  │◄────│  SyncService │────►│   API    │
│ (Cache)  │     │              │     │ (Server) │
└──────────┘     │              │     └──────────┘
                 │ 1. Push      │
                 │    pending   │
                 │ 2. Pull      │
                 │    latest    │
                 └─────────────┘
```

### 3. UI Integration

- `OfflineIndicator` — показывает статус online/offline/syncing
- DashboardScreen — загружает WO из SQLite если offline
- WorkOrderDetailScreen — сохраняет изменения в SQLite + pending_sync

## Consequences

### Positive
- ✅ Полная offline-работа для просмотра и базовых операций
- ✅ Мгновенная загрузка данных из локального кэша
- ✅ ACID гарантии через SQLite транзакции
- ✅ Прозрачный sync при восстановлении соединения

### Negative
- ❌ Нужно поддерживать sync логику вручную
- ❌ LWW может потерять данные при конкурентных изменениях
- ❌ Дополнительный код для миграций схемы БД

### Mitigation
- Server-authoritative conflict resolution для business-critical данных
- Логирование всех sync конфликтов в `audit_log`
- Периодическая очистка `pending_sync` от expired записей (>7 дней)

## Compliance Notes

- **IEC 62443 SR 3.1** — Data integrity через `updated_at` tracking
- **ISO 27001 A.12.4** — Audit trail для sync конфликтов
- **Приказ ОАЦ №66 п.7.18** — Целостность данных на конечных узлах
- **OWASP ASVS V6** — Хранение данных: SQLite с параметризованными запросами

## References

- [ADR-005: State Management Strategy](./ADR-005-state-management.md)
- [expo-sqlite documentation](https://docs.expo.dev/versions/latest/sdk/sqlite/)
- [WatermelonDB](https://watermelondb.dev/)
- [RxDB](https://rxdb.info/)
