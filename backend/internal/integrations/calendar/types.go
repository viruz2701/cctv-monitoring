// Package calendar — External Calendar Sync (Google + Microsoft Outlook).
//
// ═══════════════════════════════════════════════════════════════════════
// P1-CALENDAR: External Calendar Sync
//
// Обеспечивает bi-directional синхронизацию Work Orders с Google Calendar
// и Microsoft Outlook через OAuth2.
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — sync_log)
//   - IEC 62443-3-3 SL-3 (Zone 3 — application data integrity)
//   - OWASP ASVS V6.2 (Encrypted tokens at rest)
//
// ═══════════════════════════════════════════════════════════════════════
package calendar

import (
	"context"
	"time"
)

// ── Domain Types ──────────────────────────────────────────────────────

// WorkOrderEvent — DTO для синхронизации Work Order с событием календаря.
type WorkOrderEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
	AssignedTo  string    `json:"assigned_to"`
	Status      string    `json:"status"` // scheduled, in_progress, completed, cancelled
}

// CalendarChange — единица изменения из внешнего календаря (pull).
type CalendarChange struct {
	EventID    string    `json:"event_id"`
	Type       string    `json:"type"` // created, updated, deleted
	ExternalID string    `json:"external_id"`
	Provider   string    `json:"provider"` // google, outlook
	ChangedAt  time.Time `json:"changed_at"`
}

// ── Provider Interface ────────────────────────────────────────────────

// CalendarProvider — унифицированный интерфейс для Google/Outlook.
type CalendarProvider interface {
	// CreateEvent создаёт событие во внешнем календаре.
	// Возвращает external_id созданного события.
	CreateEvent(ctx context.Context, wo WorkOrderEvent) (string, error)

	// UpdateEvent обновляет существующее событие при изменении WO.
	UpdateEvent(ctx context.Context, eventID string, wo WorkOrderEvent) error

	// DeleteEvent удаляет событие при отмене WO.
	DeleteEvent(ctx context.Context, eventID string) error

	// SyncChanges получает изменения из внешнего календаря с момента since.
	SyncChanges(ctx context.Context, since time.Time) ([]CalendarChange, error)
}

// ── Sync Store Interface ──────────────────────────────────────────────

// Connection — OAuth2-подключение календаря из БД.
type Connection struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"`
	AccessToken  string    `json:"-"` // encrypted
	RefreshToken string    `json:"-"` // encrypted
	TokenExpiry  time.Time `json:"token_expiry"`
	CalendarID   string    `json:"calendar_id"`
	Enabled      bool      `json:"enabled"`
	TenantID     string    `json:"tenant_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EventMapping — маппинг WO ↔ внешнее событие.
type EventMapping struct {
	ID         string    `json:"id"`
	WOID       string    `json:"wo_id"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	EventURL   string    `json:"event_url"`
	Status     string    `json:"status"`
	LastSynced time.Time `json:"last_synced"`
}

// SyncStore — интерфейс для доступа к БД для SyncEngine.
type SyncStore interface {
	// ListConnections возвращает активные подключения календарей.
	ListConnections(ctx context.Context) ([]Connection, error)

	// GetConnection возвращает подключение по провайдеру и user_id.
	GetConnection(ctx context.Context, provider, userID string) (*Connection, error)

	// SaveConnection создаёт или обновляет подключение.
	SaveConnection(ctx context.Context, conn *Connection) error

	// DeleteConnection удаляет подключение календаря.
	DeleteConnection(ctx context.Context, id string) error

	// GetEventMapping возвращает маппинг WO → внешнее событие.
	GetEventMapping(ctx context.Context, woID, provider string) (*EventMapping, error)

	// SaveEventMapping создаёт или обновляет маппинг.
	SaveEventMapping(ctx context.Context, mapping *EventMapping) error

	// DeleteEventMapping удаляет маппинг.
	DeleteEventMapping(ctx context.Context, woID, provider string) error

	// ListEventMappingsByProvider возвращает все маппинги для провайдера.
	ListEventMappingsByProvider(ctx context.Context, provider string) ([]EventMapping, error)

	// LogSync записывает запись аудита синхронизации.
	LogSync(ctx context.Context, entry *SyncLogEntry) error
}

// SyncLogEntry — запись аудита синхронизации (ISO 27001 A.12.4).
type SyncLogEntry struct {
	WOID           string `json:"wo_id"`
	Provider       string `json:"provider"`
	Direction      string `json:"direction"`  // push, pull
	EventType      string `json:"event_type"` // created, updated, deleted, skipped, conflict
	ExternalID     string `json:"external_id"`
	Details        string `json:"details"`
	Status         string `json:"status"` // success, error, conflict
	ErrorMsg       string `json:"error_message"`
	TenantID       string `json:"tenant_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"` // P1-HI-09: UUID для dedup
}

// ── Sync Config ───────────────────────────────────────────────────────

// Config — конфигурация SyncEngine.
type Config struct {
	// SyncInterval — интервал автоматической синхронизации (default: 5m).
	SyncInterval time.Duration

	// ConflictStrategy — стратегия при конфликте: "wo_wins" | "calendar_wins" | "manual".
	ConflictStrategy string

	// SyncWindow — окно синхронизации в прошлое (default: 30d).
	SyncWindow time.Duration

	// DryRun — если true, не выполняет мутации (только аудит).
	DryRun bool
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		SyncInterval:     5 * time.Minute,
		ConflictStrategy: "wo_wins",
		SyncWindow:       30 * 24 * time.Hour,
		DryRun:           false,
	}
}
