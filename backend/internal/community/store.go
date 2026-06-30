// Package community — Community Protocol Registry (PROTO-07).
//
// Публичный реестр Protocol Descriptor'ов, где community может
// публиковать и обмениваться дескрипторами для вендоров CCTV.
//
// Compliance:
//   - OWASP ASVS V1 (Input validation — parameterized queries)
//   - OWASP ASVS V5 (Validation — whitelist approach)
//   - ISO 27001 A.12.4 (Audit — trace_id логирование)
//   - IEC 62443-3-3 SL-3 (Zone 3 — Backend)
package community

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Store — PostgreSQL хранилище для community дескрипторов.
// ═══════════════════════════════════════════════════════════════════════

type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewStore создаёт новый Store для community дескрипторов.
func NewStore(pool *pgxpool.Pool, logger *slog.Logger) *Store {
	return &Store{
		pool:   pool,
		logger: logger.With("component", "community.registry"),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// CRUD Operations
// ═══════════════════════════════════════════════════════════════════════

// List возвращает список community дескрипторов с фильтрацией и пагинацией.
func (s *Store) List(ctx context.Context, filter models.CommunityDescriptorFilter) (*models.CommunityDescriptorListResponse, error) {
	// Валидация пагинации (OWASP ASVS V5 — whitelist)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	// Валидация сортировки (OWASP ASVS V5.1 — whitelist)
	validSortFields := map[string]bool{
		"rating":     true,
		"downloads":  true,
		"created_at": true,
		"vendor":     true,
	}
	sortField := filter.SortBy
	if !validSortFields[sortField] {
		sortField = "rating"
	}

	sortDir := strings.ToUpper(filter.SortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "DESC"
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("vendor ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.MinRating > 0 {
		conditions = append(conditions, fmt.Sprintf("rating >= $%d", argIdx))
		args = append(args, filter.MinRating)
		argIdx++
	}

	if filter.Verified != nil {
		conditions = append(conditions, fmt.Sprintf("verified = $%d", argIdx))
		args = append(args, *filter.Verified)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM community_descriptors %s", whereClause)
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count descriptors: %w", err)
	}

	// Fetch page
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, vendor, version, rating, downloads, verified, created_at, updated_at
		FROM community_descriptors %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortField, sortDir, argIdx, argIdx+1,
	)
	args = append(args, filter.PageSize, offset)

	rows, err := s.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list descriptors: %w", err)
	}
	defer rows.Close()

	var descriptors []models.CommunityDescriptorSummary
	for rows.Next() {
		var d models.CommunityDescriptorSummary
		if err := rows.Scan(&d.ID, &d.Vendor, &d.Version, &d.Rating, &d.Downloads, &d.Verified, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan descriptor: %w", err)
		}
		descriptors = append(descriptors, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.PageSize)))

	return &models.CommunityDescriptorListResponse{
		Descriptors: descriptors,
		Total:       total,
		Page:        filter.Page,
		PageSize:    filter.PageSize,
		TotalPages:  totalPages,
	}, nil
}

// GetByVendor возвращает полный дескриптор по имени вендора.
func (s *Store) GetByVendor(ctx context.Context, vendor string) (*models.CommunityDescriptor, error) {
	query := `
		SELECT id, vendor, version, descriptor, author_id, rating, downloads, verified, created_at, updated_at
		FROM community_descriptors
		WHERE vendor = $1`

	var d models.CommunityDescriptor
	err := s.pool.QueryRow(ctx, query, vendor).Scan(
		&d.ID, &d.Vendor, &d.Version, &d.Descriptor, &d.AuthorID,
		&d.Rating, &d.Downloads, &d.Verified, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get descriptor: %w", err)
	}

	return &d, nil
}

// Publish создаёт новый community дескриптор.
func (s *Store) Publish(ctx context.Context, req models.PublishDescriptorRequest, authorID string) (*models.CommunityDescriptor, error) {
	// Проверяем, не существует ли уже дескриптор для этого вендора
	existing, err := s.GetByVendor(ctx, req.Vendor)
	if err != nil {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("descriptor for vendor %s already exists", req.Vendor)
	}

	data, err := json.Marshal(req.Descriptor)
	if err != nil {
		return nil, fmt.Errorf("marshal descriptor: %w", err)
	}

	query := `
		INSERT INTO community_descriptors (vendor, version, descriptor, author_id)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id, vendor, version, descriptor, author_id, rating, downloads, verified, created_at, updated_at`

	var d models.CommunityDescriptor
	err = s.pool.QueryRow(ctx, query, req.Vendor, req.Version, data, authorID).Scan(
		&d.ID, &d.Vendor, &d.Version, &d.Descriptor, &d.AuthorID,
		&d.Rating, &d.Downloads, &d.Verified, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("publish descriptor: %w", err)
	}

	s.logger.Info("community descriptor published",
		"vendor", req.Vendor,
		"version", req.Version,
		"author", authorID,
	)

	return &d, nil
}

// Rate устанавливает оценку для дескриптора (1-5).
// Обновляет средний рейтинг в таблице community_descriptors.
func (s *Store) Rate(ctx context.Context, descriptorID, userID string, score int) error {
	// Используем транзакцию для атомарности
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Вставляем или обновляем оценку
	upsertQuery := `
		INSERT INTO community_descriptor_ratings (descriptor_id, user_id, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (descriptor_id, user_id)
		DO UPDATE SET score = $3, created_at = NOW()`

	if _, err := tx.Exec(ctx, upsertQuery, descriptorID, userID, score); err != nil {
		return fmt.Errorf("upsert rating: %w", err)
	}

	// Обновляем средний рейтинг
	updateQuery := `
		UPDATE community_descriptors
		SET rating = (
			SELECT ROUND(AVG(score)::numeric, 2)
			FROM community_descriptor_ratings
			WHERE descriptor_id = $1
		)
		WHERE id = $1`

	if _, err := tx.Exec(ctx, updateQuery, descriptorID); err != nil {
		return fmt.Errorf("update avg rating: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	s.logger.Info("community descriptor rated",
		"descriptor_id", descriptorID,
		"user_id", userID,
		"score", score,
	)

	return nil
}

// IncrementDownload увеличивает счётчик скачиваний дескриптора.
func (s *Store) IncrementDownload(ctx context.Context, vendor string) error {
	query := `UPDATE community_descriptors SET downloads = downloads + 1 WHERE vendor = $1`

	tag, err := s.pool.Exec(ctx, query, vendor)
	if err != nil {
		return fmt.Errorf("increment download: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("descriptor not found: %s", vendor)
	}

	return nil
}

// GetDescriptorIDByVendor возвращает ID дескриптора по имени вендора.
func (s *Store) GetDescriptorIDByVendor(ctx context.Context, vendor string) (string, error) {
	query := `SELECT id FROM community_descriptors WHERE vendor = $1`

	var id string
	err := s.pool.QueryRow(ctx, query, vendor).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("descriptor not found: %s", vendor)
		}
		return "", fmt.Errorf("get descriptor id: %w", err)
	}

	return id, nil
}
