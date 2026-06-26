// Package multiregion provides multi-region geo-redundancy primitives.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-1: Multi-Region Geo-Redundancy
//
// Содержит:
//   - RegionManager — управление tenant-to-region mapping
//   - FailoverService — semi-auto failover логика
//   - NATSMirrorConfig — программатор NATS stream mirror
//
// Compliance:
//   - ISO 27001 A.17.1 (Business continuity — DR)
//   - IEC 62443 SR 7.1 (Resource availability — multi-region)
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//
// ═══════════════════════════════════════════════════════════════════════════
package multiregion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

// ValidRegions — список поддерживаемых регионов.
var ValidRegions = []string{"eu-central", "cis-east", "mena-gulf", "sea-hub"}

const (
	StatusActive    = "active"
	StatusFailover  = "failover"
	StatusMigrating = "migrating"
)

// ────────────────────────────────────────────────────────────────────────────
// TenantRegion
// ────────────────────────────────────────────────────────────────────────────

// TenantRegion представляет привязку тенанта к региону.
type TenantRegion struct {
	TenantID       string     `json:"tenant_id"`
	PrimaryRegion  string     `json:"primary_region"`
	FailoverRegion string     `json:"failover_region,omitempty"`
	Status         string     `json:"status"`
	FailoverCount  int        `json:"failover_count"`
	LastFailoverAt *time.Time `json:"last_failover_at,omitempty"`
	PinnedAt       time.Time  `json:"pinned_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// RegionStore
// ────────────────────────────────────────────────────────────────────────────

// RegionStore — интерфейс для работы с tenant_regions.
type RegionStore interface {
	GetTenantRegion(ctx context.Context, tenantID string) (*TenantRegion, error)
	SetTenantRegion(ctx context.Context, tr *TenantRegion) error
	UpdateTenantStatus(ctx context.Context, tenantID, status string) error
	RecordFailover(ctx context.Context, tenantID, failoverRegion string) error
	ListByRegion(ctx context.Context, region string) ([]TenantRegion, error)
	ListAll(ctx context.Context) ([]TenantRegion, error)
}

// PGTenantRegionStore — PostgreSQL реализация RegionStore.
type PGTenantRegionStore struct {
	pool *pgxpool.Pool
}

// NewPGTenantRegionStore создаёт новый PGTenantRegionStore.
func NewPGTenantRegionStore(pool *pgxpool.Pool) *PGTenantRegionStore {
	return &PGTenantRegionStore{pool: pool}
}

func (s *PGTenantRegionStore) GetTenantRegion(ctx context.Context, tenantID string) (*TenantRegion, error) {
	var tr TenantRegion
	err := s.pool.QueryRow(ctx, `
		SELECT tenant_id, primary_region, COALESCE(failover_region, ''),
		       status, failover_count, last_failover_at, pinned_at, created_at, updated_at
		FROM tenant_regions
		WHERE tenant_id = $1
	`, tenantID).Scan(&tr.TenantID, &tr.PrimaryRegion, &tr.FailoverRegion,
		&tr.Status, &tr.FailoverCount, &tr.LastFailoverAt,
		&tr.PinnedAt, &tr.CreatedAt, &tr.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant region %s: %w", tenantID, err)
	}
	return &tr, nil
}

func (s *PGTenantRegionStore) SetTenantRegion(ctx context.Context, tr *TenantRegion) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_regions (tenant_id, primary_region, failover_region, status, pinned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id) DO UPDATE SET
			primary_region = EXCLUDED.primary_region,
			failover_region = EXCLUDED.failover_region,
			status = EXCLUDED.status
	`, tr.TenantID, tr.PrimaryRegion, tr.FailoverRegion, tr.Status, time.Now())
	if err != nil {
		return fmt.Errorf("set tenant region %s: %w", tr.TenantID, err)
	}
	return nil
}

func (s *PGTenantRegionStore) UpdateTenantStatus(ctx context.Context, tenantID, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenant_regions SET status = $1 WHERE tenant_id = $2
	`, status, tenantID)
	if err != nil {
		return fmt.Errorf("update tenant %s status: %w", tenantID, err)
	}
	return nil
}

func (s *PGTenantRegionStore) RecordFailover(ctx context.Context, tenantID, failoverRegion string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenant_regions SET
			status = 'failover',
			failover_region = $2,
			failover_count = failover_count + 1,
			last_failover_at = NOW()
		WHERE tenant_id = $1
	`, tenantID, failoverRegion)
	if err != nil {
		return fmt.Errorf("record failover %s: %w", tenantID, err)
	}
	return nil
}

func (s *PGTenantRegionStore) ListByRegion(ctx context.Context, region string) ([]TenantRegion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id, primary_region, COALESCE(failover_region, ''),
		       status, failover_count, last_failover_at, pinned_at, created_at, updated_at
		FROM tenant_regions
		WHERE primary_region = $1 OR failover_region = $1
		ORDER BY tenant_id
	`, region)
	if err != nil {
		return nil, fmt.Errorf("list by region %s: %w", region, err)
	}
	defer rows.Close()
	return scanTenantRegions(rows)
}

func (s *PGTenantRegionStore) ListAll(ctx context.Context) ([]TenantRegion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id, primary_region, COALESCE(failover_region, ''),
		       status, failover_count, last_failover_at, pinned_at, created_at, updated_at
		FROM tenant_regions
		ORDER BY tenant_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list all tenant regions: %w", err)
	}
	defer rows.Close()
	return scanTenantRegions(rows)
}

func scanTenantRegions(rows pgx.Rows) ([]TenantRegion, error) {
	var regions []TenantRegion
	for rows.Next() {
		var tr TenantRegion
		if err := rows.Scan(&tr.TenantID, &tr.PrimaryRegion, &tr.FailoverRegion,
			&tr.Status, &tr.FailoverCount, &tr.LastFailoverAt,
			&tr.PinnedAt, &tr.CreatedAt, &tr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant region: %w", err)
		}
		regions = append(regions, tr)
	}
	if regions == nil {
		regions = []TenantRegion{}
	}
	return regions, rows.Err()
}

// ────────────────────────────────────────────────────────────────────────────
// FailoverService
// ────────────────────────────────────────────────────────────────────────────

// FailoverConfig — конфигурация failover.
type FailoverConfig struct {
	// NATSMirrorDomain — домен для NATS mirror (например, "nats-dr.cis-east.example.com")
	NATSMirrorDomain string
	// DBDNSuffix — суффикс DNS для DR PostgreSQL (например, "-dr.cis-east.example.com")
	DBDNSuffix string
}

// FailoverService реализует semi-auto failover для тенанта.
type FailoverService struct {
	store  RegionStore
	nc     *nats.Conn
	config FailoverConfig
	logger *slog.Logger
}

// NewFailoverService создаёт новый FailoverService.
func NewFailoverService(store RegionStore, nc *nats.Conn, config FailoverConfig, logger *slog.Logger) *FailoverService {
	if logger == nil {
		logger = slog.Default()
	}
	return &FailoverService{
		store:  store,
		nc:     nc,
		config: config,
		logger: logger.With("component", "failover-service"),
	}
}

// FailoverResult — результат операции failover.
type FailoverResult struct {
	TenantID        string `json:"tenant_id"`
	FromRegion      string `json:"from_region"`
	ToRegion        string `json:"to_region"`
	NATSPromoted    bool   `json:"nats_promoted"`
	DBPromoted      bool   `json:"db_promoted"`
	RoutingSwitched bool   `json:"routing_switched"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

// ExecuteFailover выполняет failover для указанного тенанта.
func (s *FailoverService) ExecuteFailover(ctx context.Context, tenantID string) (*FailoverResult, error) {
	tr, err := s.store.GetTenantRegion(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failover: get tenant %s: %w", tenantID, err)
	}
	if tr == nil {
		return nil, fmt.Errorf("failover: tenant %s not found", tenantID)
	}
	if tr.FailoverRegion == "" {
		return nil, fmt.Errorf("failover: tenant %s has no failover region configured", tenantID)
	}

	result := &FailoverResult{
		TenantID:   tenantID,
		FromRegion: tr.PrimaryRegion,
		ToRegion:   tr.FailoverRegion,
	}

	s.logger.Warn("executing failover",
		"tenant", tenantID,
		"from", tr.PrimaryRegion,
		"to", tr.FailoverRegion,
	)

	// Step 1: Promote NATS mirror → active (programmatic)
	if s.nc != nil {
		if err := s.promoteNATSMirror(ctx, tr.FailoverRegion); err != nil {
			result.Error = fmt.Sprintf("nats promotion failed: %v", err)
			result.Status = "failed"
			return result, nil
		}
		result.NATSPromoted = true
	}

	// Step 2: Promote DR PostgreSQL (signal to infrastructure)
	if err := s.promoteDatabase(ctx, tr.FailoverRegion); err != nil {
		result.Error = fmt.Sprintf("db promotion failed: %v", err)
		result.Status = "failed"
		return result, nil
	}
	result.DBPromoted = true

	// Step 3: Switch routing — записываем failover в БД
	if err := s.store.RecordFailover(ctx, tenantID, tr.FailoverRegion); err != nil {
		result.Error = fmt.Sprintf("routing switch failed: %v", err)
		result.Status = "failed"
		return result, nil
	}
	result.RoutingSwitched = true

	result.Status = "success"
	s.logger.Warn("failover completed",
		"tenant", tenantID,
		"to", tr.FailoverRegion,
	)

	return result, nil
}

// promoteNATSMirror публикует NATS сообщение для promotion DR mirror.
// В production это триггерит Helm-оператор, который переключает mirror → active.
func (s *FailoverService) promoteNATSMirror(ctx context.Context, region string) error {
	if s.nc == nil {
		s.logger.Warn("nats not connected, skipping nats mirror promotion")
		return nil
	}

	msg := fmt.Sprintf(`{"action":"promote_mirror","region":"%s","timestamp":"%s"}`,
		region, time.Now().UTC().Format(time.RFC3339))

	return s.nc.Publish("dr.nats.promote", []byte(msg))
}

// promoteDatabase публикует NATS сообщение для promotion DR PostgreSQL.
func (s *FailoverService) promoteDatabase(ctx context.Context, region string) error {
	if s.nc == nil {
		s.logger.Warn("nats not connected, db promotion requires manual execution")
		return nil
	}

	msg := fmt.Sprintf(`{"action":"promote_db","region":"%s","timestamp":"%s"}`,
		region, time.Now().UTC().Format(time.RFC3339))

	return s.nc.Publish("dr.postgres.promote", []byte(msg))
}

// ────────────────────────────────────────────────────────────────────────────
// NATSMirrorSetup — программатор NATS stream mirror
// ────────────────────────────────────────────────────────────────────────────

// NATSMirrorSetup настраивает NATS JetStream mirror streams для DR.
type NATSMirrorSetup struct {
	js     jetstream.JetStream
	logger *slog.Logger
}

// NewNATSMirrorSetup создаёт новый NATSMirrorSetup.
func NewNATSMirrorSetup(nc *nats.Conn, logger *slog.Logger) (*NATSMirrorSetup, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &NATSMirrorSetup{js: js, logger: logger.With("component", "nats-mirror-setup")}, nil
}

// MirrorConfig — конфигурация NATS stream mirror.
type MirrorConfig struct {
	SourceStream string // Имя исходного стрима
	MirrorName   string // Имя стрима-зеркала
	RemoteDomain string // NATS домен удалённого региона (например, "nats.eu-central.example.com")
}

// CreateMirrorStream создаёт mirror stream для DR.
func (s *NATSMirrorSetup) CreateMirrorStream(ctx context.Context, cfg MirrorConfig) (*jetstream.StreamInfo, error) {
	stream, err := s.js.CreateStream(ctx, jetstream.StreamConfig{
		Name: cfg.MirrorName,
		Mirror: &jetstream.StreamSource{
			Name: cfg.SourceStream,
		},
		Replicas: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("create mirror stream %s: %w", cfg.MirrorName, err)
	}

	s.logger.Info("mirror stream created",
		"name", cfg.MirrorName,
		"source", cfg.SourceStream,
	)

	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("mirror stream info: %w", err)
	}
	return info, nil
}
// SetupCrossRegionMirrors настраивает mirror для всех системных стримов.
func (s *NATSMirrorSetup) SetupCrossRegionMirrors(ctx context.Context, remoteDomain string) (int, error) {
	// Системные стримы, которые должны быть mirror'ированы
	systemStreams := []string{
		"events", "alarms", "telemetry",
		"work_orders", "sla_events",
	}

	count := 0
	for _, streamName := range systemStreams {
		_, err := s.CreateMirrorStream(ctx, MirrorConfig{
			SourceStream: streamName,
			MirrorName:   streamName + "_mirror",
			RemoteDomain: remoteDomain,
		})
		if err != nil {
			s.logger.Warn("failed to create mirror for stream",
				"stream", streamName, "error", err)
			continue
		}
		count++
	}

	return count, nil
}
