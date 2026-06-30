// Package api — PG-реализация VersionStore.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-API: API Versioning Strategy — PostgreSQL Store
//
// Соответствует:
//   - IEC 62443-3-3 SL-3 (Zone 3 — Backend): Data integrity
//   - ISO 27001 A.12.4.1: Audit trail for version changes
//   - СТБ 34.101.27 п. 6.3: Контроль целостности данных
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGVersionStore — PostgreSQL реализация VersionStore.
type PGVersionStore struct {
	pool *pgxpool.Pool
}

// NewPGVersionStore создаёт PGVersionStore.
func NewPGVersionStore(pool *pgxpool.Pool) *PGVersionStore {
	return &PGVersionStore{pool: pool}
}

// ListVersions возвращает список всех зарегистрированных версий API.
func (s *PGVersionStore) ListVersions() ([]VersionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT version, released_at, deprecated_at, sunset_at, changelog
		FROM api_versions
		ORDER BY released_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []VersionInfo
	for rows.Next() {
		var (
			v            VersionInfo
			releasedAt   time.Time
			deprecatedAt *time.Time
			sunsetAt     *time.Time
		)
		if err := rows.Scan(&v.Version, &releasedAt, &deprecatedAt, &sunsetAt, &v.Changelog); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		v.ReleasedAt = releasedAt.Format(time.RFC3339)
		v.Deprecated = deprecatedAt != nil
		if deprecatedAt != nil {
			v.DeprecatedAt = deprecatedAt.Format(time.RFC3339)
		}
		if sunsetAt != nil {
			v.Sunset = sunsetAt.Format(time.RFC3339)
		}
		versions = append(versions, v)
	}

	if versions == nil {
		versions = []VersionInfo{}
	}
	return versions, nil
}

// GetVersion возвращает метаданные указанной версии.
func (s *PGVersionStore) GetVersion(version APIVersion) (*VersionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		v            VersionInfo
		releasedAt   time.Time
		deprecatedAt *time.Time
		sunsetAt     *time.Time
	)

	err := s.pool.QueryRow(ctx, `
		SELECT version, released_at, deprecated_at, sunset_at, changelog
		FROM api_versions
		WHERE version = $1
	`, string(version)).Scan(&v.Version, &releasedAt, &deprecatedAt, &sunsetAt, &v.Changelog)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get version %s: %w", version, err)
	}

	v.ReleasedAt = releasedAt.Format(time.RFC3339)
	v.Deprecated = deprecatedAt != nil
	if deprecatedAt != nil {
		v.DeprecatedAt = deprecatedAt.Format(time.RFC3339)
	}
	if sunsetAt != nil {
		v.Sunset = sunsetAt.Format(time.RFC3339)
	}

	return &v, nil
}

// CreateVersion регистрирует новую версию API.
func (s *PGVersionStore) CreateVersion(version APIVersion, changelog string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO api_versions (version, changelog)
		VALUES ($1, $2)
		ON CONFLICT (version) DO UPDATE
		SET changelog = EXCLUDED.changelog,
		    updated_at = NOW()
	`, string(version), changelog)
	if err != nil {
		return fmt.Errorf("create version %s: %w", version, err)
	}
	return nil
}

// UpdateVersion обновляет метаданные версии API.
func (s *PGVersionStore) UpdateVersion(version APIVersion, info VersionInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var deprecatedAt, sunsetAt *time.Time
	if info.DeprecatedAt != "" {
		t, err := time.Parse(time.RFC3339, info.DeprecatedAt)
		if err == nil {
			deprecatedAt = &t
		}
	}
	if info.Sunset != "" {
		t, err := time.Parse(time.RFC3339, info.Sunset)
		if err == nil {
			sunsetAt = &t
		}
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE api_versions
		SET deprecated_at = $2,
		    sunset_at     = $3,
		    changelog     = $4,
		    updated_at    = NOW()
		WHERE version = $1
	`, string(version), deprecatedAt, sunsetAt, info.Changelog)
	if err != nil {
		return fmt.Errorf("update version %s: %w", version, err)
	}
	return nil
}
