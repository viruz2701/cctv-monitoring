package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gb-telemetry-collector/internal/integrations/calendar"
)

// ── PostgreSQL SyncStore ──────────────────────────────────────────────

// CalendarStore — PostgreSQL реализация calendar.SyncStore.
type CalendarStore struct {
	pool *pgxpool.Pool
}

// NewCalendarStore создаёт новый CalendarStore.
func NewCalendarStore(pool *pgxpool.Pool) *CalendarStore {
	return &CalendarStore{pool: pool}
}

// ── Connections ───────────────────────────────────────────────────────

func (s *CalendarStore) ListConnections(ctx context.Context) ([]calendar.Connection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, provider, access_token, refresh_token,
		       token_expiry, calendar_id, enabled, tenant_id, created_at, updated_at
		FROM calendar_connections
		WHERE enabled = true`)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var conns []calendar.Connection
	for rows.Next() {
		var c calendar.Connection
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Provider, &c.AccessToken, &c.RefreshToken,
			&c.TokenExpiry, &c.CalendarID, &c.Enabled, &c.TenantID, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		conns = append(conns, c)
	}
	return conns, nil
}

func (s *CalendarStore) GetConnection(ctx context.Context, provider, userID string) (*calendar.Connection, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, access_token, refresh_token,
		       token_expiry, calendar_id, enabled, tenant_id, created_at, updated_at
		FROM calendar_connections
		WHERE provider = $1 AND user_id = $2`, provider, userID)

	var c calendar.Connection
	err := row.Scan(
		&c.ID, &c.UserID, &c.Provider, &c.AccessToken, &c.RefreshToken,
		&c.TokenExpiry, &c.CalendarID, &c.Enabled, &c.TenantID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	return &c, nil
}

func (s *CalendarStore) SaveConnection(ctx context.Context, conn *calendar.Connection) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_connections (user_id, provider, access_token, refresh_token,
		                                   token_expiry, calendar_id, enabled, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expiry  = EXCLUDED.token_expiry,
			calendar_id   = EXCLUDED.calendar_id,
			enabled       = EXCLUDED.enabled,
			updated_at    = NOW()`,
		conn.UserID, conn.Provider, conn.AccessToken, conn.RefreshToken,
		conn.TokenExpiry, conn.CalendarID, conn.Enabled, conn.TenantID,
	)
	if err != nil {
		return fmt.Errorf("save connection: %w", err)
	}
	return nil
}

func (s *CalendarStore) DeleteConnection(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM calendar_connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	return nil
}

// ── Event Mappings ────────────────────────────────────────────────────

func (s *CalendarStore) GetEventMapping(ctx context.Context, woID, provider string) (*calendar.EventMapping, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, wo_id, provider, external_id, event_url, status, last_synced
		FROM calendar_events
		WHERE wo_id = $1 AND provider = $2`, woID, provider)

	var m calendar.EventMapping
	err := row.Scan(&m.ID, &m.WOID, &m.Provider, &m.ExternalID, &m.EventURL, &m.Status, &m.LastSynced)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get event mapping: %w", err)
	}
	return &m, nil
}

func (s *CalendarStore) SaveEventMapping(ctx context.Context, mapping *calendar.EventMapping) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_events (wo_id, provider, external_id, event_url, status, last_synced)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (wo_id, provider) DO UPDATE SET
			external_id = EXCLUDED.external_id,
			event_url   = EXCLUDED.event_url,
			status      = EXCLUDED.status,
			last_synced = EXCLUDED.last_synced`,
		mapping.WOID, mapping.Provider, mapping.ExternalID,
		mapping.EventURL, mapping.Status, mapping.LastSynced,
	)
	if err != nil {
		return fmt.Errorf("save event mapping: %w", err)
	}
	return nil
}

func (s *CalendarStore) DeleteEventMapping(ctx context.Context, woID, provider string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM calendar_events WHERE wo_id = $1 AND provider = $2`,
		woID, provider)
	if err != nil {
		return fmt.Errorf("delete event mapping: %w", err)
	}
	return nil
}

func (s *CalendarStore) ListEventMappingsByProvider(ctx context.Context, provider string) ([]calendar.EventMapping, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, wo_id, provider, external_id, event_url, status, last_synced
		FROM calendar_events
		WHERE provider = $1`, provider)
	if err != nil {
		return nil, fmt.Errorf("list event mappings: %w", err)
	}
	defer rows.Close()

	var mappings []calendar.EventMapping
	for rows.Next() {
		var m calendar.EventMapping
		if err := rows.Scan(&m.ID, &m.WOID, &m.Provider, &m.ExternalID, &m.EventURL, &m.Status, &m.LastSynced); err != nil {
			return nil, fmt.Errorf("scan event mapping: %w", err)
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

// ── Sync Log ──────────────────────────────────────────────────────────

func (s *CalendarStore) LogSync(ctx context.Context, entry *calendar.SyncLogEntry) error {
	details, _ := json.Marshal(map[string]string{
		"details": entry.Details,
	})

	_, err := s.pool.Exec(ctx, `
		INSERT INTO calendar_sync_log (wo_id, provider, direction, event_type, external_id, details, status, error_message, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entry.WOID, entry.Provider, entry.Direction, entry.EventType,
		entry.ExternalID, details, entry.Status, entry.ErrorMsg, entry.TenantID,
	)
	if err != nil {
		return fmt.Errorf("log sync: %w", err)
	}
	return nil
}

// compile-time interface check
var _ calendar.SyncStore = (*CalendarStore)(nil)
