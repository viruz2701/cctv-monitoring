// Package descriptor — Protocol Registry (Backend).
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-03: Protocol Registry (Backend)
//
// Хранит Protocol Descriptor'ы в PostgreSQL (JSONB) с кэшированием в памяти.
// Используется Backend API для отдачи дескрипторов Edge-агентам.
//
// Compliance:
//   - ISO 27001 A.12.4.1: Audit logging
//   - IEC 62443-3-3 SL-3: Zone separation
//
// ═══════════════════════════════════════════════════════════════════════════
package descriptor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DescriptorRegistry управляет ProtocolDescriptor'ами в БД и кэше.
type DescriptorRegistry struct {
	pool    *pgxpool.Pool
	cache   map[string]*ProtocolDescriptor // vendor → descriptor
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewDescriptorRegistry создаёт новый DescriptorRegistry.
func NewDescriptorRegistry(pool *pgxpool.Pool, logger *slog.Logger) *DescriptorRegistry {
	return &DescriptorRegistry{
		pool:   pool,
		cache:  make(map[string]*ProtocolDescriptor),
		logger: logger.With("component", "descriptor_registry"),
	}
}

// GetDescriptor возвращает дескриптор для вендора (с кэшем).
func (r *DescriptorRegistry) GetDescriptor(ctx context.Context, vendor string) (*ProtocolDescriptor, error) {
	// 1. Проверяем кэш
	r.mu.RLock()
	if d, ok := r.cache[vendor]; ok {
		r.mu.RUnlock()
		return d.Clone(), nil
	}
	r.mu.RUnlock()

	// 2. Загружаем из БД
	descriptor, err := r.loadFromDB(ctx, vendor)
	if err != nil {
		return nil, err
	}

	// 3. Сохраняем в кэш
	r.mu.Lock()
	r.cache[vendor] = descriptor
	r.mu.Unlock()

	return descriptor.Clone(), nil
}

// SaveDescriptor сохраняет дескриптор в БД и обновляет кэш.
func (r *DescriptorRegistry) SaveDescriptor(ctx context.Context, descriptor *ProtocolDescriptor) error {
	// Валидация
	if err := descriptor.Validate(); err != nil {
		return fmt.Errorf("validate descriptor: %w", err)
	}

	// Сериализуем в JSON
	data, err := json.Marshal(descriptor)
	if err != nil {
		return fmt.Errorf("marshal descriptor: %w", err)
	}

	query := `
		INSERT INTO protocol_descriptors (vendor, version, descriptor)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (vendor)
		DO UPDATE SET version = $2, descriptor = $3::jsonb, updated_at = NOW()`

	_, err = r.pool.Exec(ctx, query, descriptor.Vendor, descriptor.Version, data)
	if err != nil {
		return fmt.Errorf("save descriptor: %w", err)
	}

	// Обновляем кэш
	descriptor.RawJSON = data
	r.mu.Lock()
	r.cache[descriptor.Vendor] = descriptor
	r.mu.Unlock()

	r.logger.Info("descriptor saved", "vendor", descriptor.Vendor, "version", descriptor.Version)
	return nil
}

// DeleteDescriptor удаляет дескриптор вендора.
func (r *DescriptorRegistry) DeleteDescriptor(ctx context.Context, vendor string) error {
	query := `DELETE FROM protocol_descriptors WHERE vendor = $1`
	tag, err := r.pool.Exec(ctx, query, vendor)
	if err != nil {
		return fmt.Errorf("delete descriptor: %w", err)
	}

	r.mu.Lock()
	delete(r.cache, vendor)
	r.mu.Unlock()

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("descriptor not found: %s", vendor)
	}

	r.logger.Info("descriptor deleted", "vendor", vendor)
	return nil
}

// ListVendors возвращает список всех зарегистрированных вендоров.
func (r *DescriptorRegistry) ListVendors(ctx context.Context) ([]string, error) {
	query := `SELECT vendor FROM protocol_descriptors ORDER BY vendor`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}
	defer rows.Close()

	var vendors []string
	for rows.Next() {
		var vendor string
		if err := rows.Scan(&vendor); err != nil {
			return nil, err
		}
		vendors = append(vendors, vendor)
	}

	return vendors, rows.Err()
}

// InvalidateCache очищает кэш для указанного вендора.
func (r *DescriptorRegistry) InvalidateCache(vendor string) {
	r.mu.Lock()
	delete(r.cache, vendor)
	r.mu.Unlock()
}

// WarmupCache загружает все дескрипторы в кэш при старте.
func (r *DescriptorRegistry) WarmupCache(ctx context.Context) error {
	query := `SELECT vendor, version, descriptor FROM protocol_descriptors`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("warmup cache: %w", err)
	}
	defer rows.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	for rows.Next() {
		var vendor, version string
		var data []byte
		if err := rows.Scan(&vendor, &version, &data); err != nil {
			return err
		}

		var descriptor ProtocolDescriptor
		if err := json.Unmarshal(data, &descriptor); err != nil {
			r.logger.Warn("failed to parse descriptor", "vendor", vendor, "error", err)
			continue
		}
		descriptor.RawJSON = data
		r.cache[vendor] = &descriptor
	}

	r.logger.Info("descriptor cache warmed up", "count", len(r.cache))
	return rows.Err()
}

// loadFromDB загружает дескриптор из БД.
func (r *DescriptorRegistry) loadFromDB(ctx context.Context, vendor string) (*ProtocolDescriptor, error) {
	query := `SELECT vendor, version, descriptor FROM protocol_descriptors WHERE vendor = $1`

	var vendorName, version string
	var data []byte
	err := r.pool.QueryRow(ctx, query, vendor).Scan(&vendorName, &version, &data)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("descriptor not found: %s", vendor)
		}
		return nil, fmt.Errorf("load descriptor: %w", err)
	}

	var descriptor ProtocolDescriptor
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor: %w", err)
	}
	descriptor.RawJSON = data

	return &descriptor, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Cache Stats
// ────────────────────────────────────────────────────────────────────────────

// CacheStats содержит статистику кэша дескрипторов.
type CacheStats struct {
	Size      int            `json:"size"`
	Vendors   []string       `json:"vendors"`
	Timestamp time.Time      `json:"timestamp"`
}

// Stats возвращает статистику кэша.
func (r *DescriptorRegistry) Stats() *CacheStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vendors := make([]string, 0, len(r.cache))
	for v := range r.cache {
		vendors = append(vendors, v)
	}

	return &CacheStats{
		Size:      len(r.cache),
		Vendors:   vendors,
		Timestamp: time.Now(),
	}
}
