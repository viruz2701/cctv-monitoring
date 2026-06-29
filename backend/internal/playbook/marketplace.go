// Package playbook — Marketplace pre-built playbooks.
//
// P1-MARKET: Публичный marketplace для pre-built playbooks
// (Hikvision, Dahua, Axis, Uniview) с rating/review,
// one-click install, vendor verification, private sharing.
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — RBAC в handler)
//   - ISO 27001 A.12.4 (Audit trail — created_at, trace_id логирование)
//   - OWASP ASVS V1 (Input validation — whitelist подход)
//   - OWASP ASVS V6 (Cryptographic storage — UUID PK)
package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// MarketplacePlaybook — публичный плейбук в marketplace.
type MarketplacePlaybook struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Vendor       string          `json:"vendor"` // hikvision, dahua, axis, uniview, generic
	Version      string          `json:"version"`
	CompatMatrix []string        `json:"compat_matrix"` // supported device models
	AvgRating    float64         `json:"avg_rating"`
	ReviewCount  int             `json:"review_count"`
	InstallCount int             `json:"install_count"`
	Verified     bool            `json:"verified"` // vendor-verified badge
	TenantID     string          `json:"tenant_id"`
	PlaybookData json.RawMessage `json:"playbook_data,omitempty"` // полные данные при установке
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// MarketplaceFilter — фильтры для списка плейбуков.
type MarketplaceFilter struct {
	Vendor    string  `json:"vendor,omitempty"`
	MinRating float64 `json:"min_rating,omitempty"`
	Search    string  `json:"search,omitempty"`
	Verified  *bool   `json:"verified,omitempty"`
	TenantID  string  `json:"tenant_id,omitempty"` // для private sharing
	Limit     int     `json:"limit,omitempty"`
	Offset    int     `json:"offset,omitempty"`
}

// MarketplaceRating — рейтинг/отзыв пользователя.
type MarketplaceRating struct {
	ID         string    `json:"id"`
	PlaybookID string    `json:"playbook_id"`
	UserID     string    `json:"user_id"`
	Score      int       `json:"score"`
	Review     string    `json:"review,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// InstallRecord — запись об установке плейбука.
type InstallRecord struct {
	ID          string    `json:"id"`
	PlaybookID  string    `json:"playbook_id"`
	TenantID    string    `json:"tenant_id"`
	InstalledAt time.Time `json:"installed_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Valid vendor whitelist (OWASP ASVS V5.1)
// ═══════════════════════════════════════════════════════════════════════

var validVendors = map[string]bool{
	"hikvision": true,
	"dahua":     true,
	"axis":      true,
	"uniview":   true,
	"generic":   true,
}

// ═══════════════════════════════════════════════════════════════════════
// MarketplaceService
// ═══════════════════════════════════════════════════════════════════════

// MarketplaceService — сервис для работы с marketplace playbook'ов.
type MarketplaceService struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewMarketplaceService создаёт новый MarketplaceService.
func NewMarketplaceService(db *pgxpool.Pool, logger *slog.Logger) *MarketplaceService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MarketplaceService{
		db:     db,
		logger: logger.With("component", "playbook-marketplace"),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Public Queries
// ═══════════════════════════════════════════════════════════════════════

// List возвращает список плейбуков с фильтрацией и пагинацией.
//
// Поддерживаемые фильтры:
//   - Vendor (whitelist: hikvision, dahua, axis, uniview, generic)
//   - MinRating (>= значение)
//   - Search (full-text по name + description)
//   - Verified (bool)
//   - TenantID (private sharing)
//
// Возвращает список и общее количество (для пагинации).
func (ms *MarketplaceService) List(ctx context.Context, filter MarketplaceFilter) ([]MarketplacePlaybook, int, error) {
	// Валидация vendor (OWASP ASVS V5.1 — whitelist)
	if filter.Vendor != "" && !validVendors[filter.Vendor] {
		return nil, 0, fmt.Errorf("marketplace: invalid vendor %q", filter.Vendor)
	}

	// Значения по умолчанию
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Строим динамический запрос
	where := " WHERE 1=1"
	args := make([]interface{}, 0)
	argIdx := 1

	if filter.Vendor != "" {
		where += fmt.Sprintf(" AND mp.vendor = $%d", argIdx)
		args = append(args, filter.Vendor)
		argIdx++
	}
	if filter.MinRating > 0 {
		where += fmt.Sprintf(" AND mp.avg_rating >= $%d", argIdx)
		args = append(args, filter.MinRating)
		argIdx++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND to_tsvector('english', mp.name || ' ' || COALESCE(mp.description, '')) @@ plainto_tsquery('english', $%d)", argIdx)
		args = append(args, filter.Search)
		argIdx++
	}
	if filter.Verified != nil {
		where += fmt.Sprintf(" AND mp.verified = $%d", argIdx)
		args = append(args, *filter.Verified)
		argIdx++
	}
	if filter.TenantID != "" {
		// Private sharing: плейбуки, опубликованные этим tenant'ом
		// или расшаренные с ним
		where += fmt.Sprintf(` AND (
			mp.tenant_id = $%d
			OR mp.id IN (SELECT playbook_id FROM playbook_shares WHERE target_tenant = $%d)
		)`, argIdx, argIdx)
		args = append(args, filter.TenantID)
		argIdx++
	}

	// Общее количество
	var total int
	countQuery := "SELECT COUNT(*) FROM playbook_marketplace mp" + where
	if err := ms.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("marketplace: count: %w", err)
	}

	// Данные
	query := `SELECT
		mp.id, mp.name, mp.description, mp.vendor, mp.version,
		mp.compat_matrix, mp.avg_rating, mp.review_count,
		mp.install_count, mp.verified, mp.tenant_id,
		mp.created_at, mp.updated_at
	FROM playbook_marketplace mp` + where +
		" ORDER BY mp.avg_rating DESC, mp.install_count DESC" +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := ms.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list: %w", err)
	}
	defer rows.Close()

	var playbooks []MarketplacePlaybook
	for rows.Next() {
		var p MarketplacePlaybook
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Vendor, &p.Version,
			&p.CompatMatrix, &p.AvgRating, &p.ReviewCount,
			&p.InstallCount, &p.Verified, &p.TenantID,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("marketplace: scan: %w", err)
		}
		playbooks = append(playbooks, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("marketplace: rows: %w", err)
	}

	return playbooks, total, nil
}

// Get возвращает playbook по ID (с playbook_data).
func (ms *MarketplaceService) Get(ctx context.Context, id string) (*MarketplacePlaybook, error) {
	query := `SELECT
		mp.id, mp.name, mp.description, mp.vendor, mp.version,
		mp.compat_matrix, mp.playbook_data, mp.avg_rating,
		mp.review_count, mp.install_count, mp.verified,
		mp.tenant_id, mp.created_at, mp.updated_at
	FROM playbook_marketplace mp WHERE mp.id = $1`

	var p MarketplacePlaybook
	err := ms.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Vendor, &p.Version,
		&p.CompatMatrix, &p.PlaybookData, &p.AvgRating,
		&p.ReviewCount, &p.InstallCount, &p.Verified,
		&p.TenantID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("marketplace: get %s: %w", id, err)
	}

	return &p, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Mutations
// ═══════════════════════════════════════════════════════════════════════

// Install устанавливает плейбук в tenant.
// Создаёт запись в playbook_installs (триггер увеличит счётчик).
func (ms *MarketplaceService) Install(ctx context.Context, tenantID, playbookID string) error {
	// Проверяем, что плейбук существует
	var exists bool
	err := ms.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM playbook_marketplace WHERE id = $1)", playbookID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("marketplace: check exist: %w", err)
	}
	if !exists {
		return fmt.Errorf("marketplace: playbook %q not found", playbookID)
	}

	// Проверяем, что tenant ещё не устанавливал этот плейбук
	var alreadyInstalled bool
	err = ms.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM playbook_installs WHERE playbook_id = $1 AND tenant_id = $2)",
		playbookID, tenantID,
	).Scan(&alreadyInstalled)
	if err != nil {
		return fmt.Errorf("marketplace: check install: %w", err)
	}
	if alreadyInstalled {
		// Не ошибка — idempotent
		ms.logger.Debug("playbook already installed by tenant",
			"playbook_id", playbookID,
			"tenant_id", tenantID,
		)
		return nil
	}

	_, err = ms.db.Exec(ctx,
		"INSERT INTO playbook_installs (playbook_id, tenant_id) VALUES ($1, $2)",
		playbookID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("marketplace: install %s in tenant %s: %w", playbookID, tenantID, err)
	}

	ms.logger.Info("playbook installed",
		"playbook_id", playbookID,
		"tenant_id", tenantID,
	)
	return nil
}

// Rate добавляет или обновляет рейтинг пользователя.
//
// OWASP ASVS V1: score валидируется (1-5).
// UNIQUE(playbook_id, user_id) — upsert.
func (ms *MarketplaceService) Rate(ctx context.Context, playbookID, userID string, score int, review string) error {
	if score < 1 || score > 5 {
		return fmt.Errorf("marketplace: score %d out of range [1-5]", score)
	}

	// Upsert через INSERT ... ON CONFLICT DO UPDATE
	query := `INSERT INTO playbook_ratings (playbook_id, user_id, score, review)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (playbook_id, user_id)
		DO UPDATE SET score = EXCLUDED.score, review = EXCLUDED.review, updated_at = NOW()`

	_, err := ms.db.Exec(ctx, query, playbookID, userID, score, review)
	if err != nil {
		return fmt.Errorf("marketplace: rate playbook %s by user %s: %w", playbookID, userID, err)
	}

	ms.logger.Info("playbook rated",
		"playbook_id", playbookID,
		"user_id", userID,
		"score", score,
	)
	return nil
}

// Share приватно расшаривает плейбук между tenant'ами.
func (ms *MarketplaceService) Share(ctx context.Context, playbookID, sourceTenant, targetTenant string) error {
	// Проверяем, что плейбук принадлежит sourceTenant
	var ownerTenant string
	err := ms.db.QueryRow(ctx,
		"SELECT tenant_id FROM playbook_marketplace WHERE id = $1", playbookID,
	).Scan(&ownerTenant)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("marketplace: playbook %q not found", playbookID)
		}
		return fmt.Errorf("marketplace: check owner: %w", err)
	}
	if ownerTenant != sourceTenant {
		return fmt.Errorf("marketplace: playbook %q does not belong to tenant %q", playbookID, sourceTenant)
	}

	_, err = ms.db.Exec(ctx,
		`INSERT INTO playbook_shares (playbook_id, source_tenant, target_tenant)
		VALUES ($1, $2, $3)
		ON CONFLICT (playbook_id, target_tenant) DO NOTHING`,
		playbookID, sourceTenant, targetTenant,
	)
	if err != nil {
		return fmt.Errorf("marketplace: share %s from %s to %s: %w",
			playbookID, sourceTenant, targetTenant, err)
	}

	ms.logger.Info("playbook shared",
		"playbook_id", playbookID,
		"from_tenant", sourceTenant,
		"to_tenant", targetTenant,
	)
	return nil
}

// GetRatingForUser возвращает рейтинг пользователя для конкретного плейбука.
func (ms *MarketplaceService) GetRatingForUser(ctx context.Context, playbookID, userID string) (*MarketplaceRating, error) {
	query := `SELECT id, playbook_id, user_id, score, COALESCE(review, ''), created_at
		FROM playbook_ratings WHERE playbook_id = $1 AND user_id = $2`

	var r MarketplaceRating
	err := ms.db.QueryRow(ctx, query, playbookID, userID).Scan(
		&r.ID, &r.PlaybookID, &r.UserID, &r.Score, &r.Review, &r.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("marketplace: get rating: %w", err)
	}
	return &r, nil
}

// GetInstalledPlaybooks возвращает установленные плейбуки для tenant'а.
func (ms *MarketplaceService) GetInstalledPlaybooks(ctx context.Context, tenantID string) ([]MarketplacePlaybook, error) {
	query := `SELECT
		mp.id, mp.name, mp.description, mp.vendor, mp.version,
		mp.compat_matrix, mp.avg_rating, mp.review_count,
		mp.install_count, mp.verified, mp.tenant_id,
		mp.created_at, mp.updated_at
	FROM playbook_marketplace mp
	INNER JOIN playbook_installs pi ON pi.playbook_id = mp.id
	WHERE pi.tenant_id = $1
	ORDER BY pi.installed_at DESC`

	rows, err := ms.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: installed list: %w", err)
	}
	defer rows.Close()

	var playbooks []MarketplacePlaybook
	for rows.Next() {
		var p MarketplacePlaybook
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Vendor, &p.Version,
			&p.CompatMatrix, &p.AvgRating, &p.ReviewCount,
			&p.InstallCount, &p.Verified, &p.TenantID,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("marketplace: scan installed: %w", err)
		}
		playbooks = append(playbooks, p)
	}
	return playbooks, rows.Err()
}
