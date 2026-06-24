// Package events — Cold Storage tier for Event Store.
//
// Реализует долговременное хранение событий в S3/MinIO (до 5 лет).
// Совместим с любым S3-compatible хранилищем (MinIO, AWS S3, DigitalOcean Spaces).
//
// Compliance:
//   - ISO 27001:2022 A.8.10 (Information disposal — retention 5 лет)
//   - ISO 27001:2022 A.12.4.1 (Event logging — retention)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity)
//   - Приказ ОАЦ №66 п. 7.18.3 (Audit trail retention для КИИ)
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

// ═══════════════════════════════════════════════════════════════════════
// ColdStorage — S3/MinIO based cold storage for events.
// ═══════════════════════════════════════════════════════════════════════

// ColdStorageConfig — конфигурация S3/MinIO cold storage.
type ColdStorageConfig struct {
	Endpoint  string        // S3 endpoint (e.g., "play.min.io:9000")
	Region    string        // region (default: "us-east-1")
	Bucket    string        // bucket name
	AccessKey string        // access key
	SecretKey string        // secret key
	UseTLS    bool          // использовать HTTPS
	Retention time.Duration // срок хранения (default: 1825 days = 5 years)
	Logger    *slog.Logger
}

// ColdListOptions — параметры для поиска событий в cold storage.
type ColdListOptions struct {
	Source      EventSource
	EventType   string
	AggregateID string
	Since       time.Time
	Until       time.Time
	Limit       int
}

// ColdStorage предоставляет методы для чтения/записи событий в S3/MinIO.
//
// Структура ключей в S3:
//
//	events/{source}/{year}/{month}/{day}/{event_id}.json
//
// Пример:
//
//	events/alarms/2026/06/24/01b8e3a0-1234-7abc-def0-123456789abc.json
type ColdStorage struct {
	client  *minio.Client
	bucket  string
	cfg     ColdStorageConfig
	logger  *slog.Logger
}

// NewColdStorage создаёт новый ColdStorage с подключением к S3/MinIO.
func NewColdStorage(cfg ColdStorageConfig) (*ColdStorage, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("cold storage: bucket name is required")
	}
	if cfg.Retention <= 0 {
		cfg.Retention = 1825 * 24 * time.Hour // 5 years
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseTLS,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("cold storage minio client: %w", err)
	}

	// Проверяем/создаём bucket
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("cold storage check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		}); err != nil {
			return nil, fmt.Errorf("cold storage create bucket: %w", err)
		}
		cfg.Logger.Info("cold storage bucket created", "bucket", cfg.Bucket)

		// Устанавливаем lifecycle policy для автоматического удаления
		if err := setLifecyclePolicy(ctx, client, cfg.Bucket, cfg.Retention); err != nil {
			cfg.Logger.Warn("cold storage lifecycle policy not set", "error", err)
		}
	}

	return &ColdStorage{
		client: client,
		bucket: cfg.Bucket,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// ── Core operations ──────────────────────────────────────────────────

// Save сохраняет событие в S3/MinIO.
//
// Путь: events/{source}/{year}/{month}/{day}/{event_id}.json
//
// Compliance:
//   - ISO 27001 A.8.10 (Information disposal — retention через lifecycle policy)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity through immutability)
func (cs *ColdStorage) Save(ctx context.Context, record *EventRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("cold storage marshal: %w", err)
	}

	key := cs.objectKey(record)

	// Upload with context timeout
	uploadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = cs.client.PutObject(uploadCtx, cs.bucket, key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType:  "application/json",
			UserMetadata: map[string]string{
				"source":       string(record.Source),
				"event_type":   record.EventType,
				"aggregate_id": record.AggregateID,
				"timestamp":    record.Timestamp.Format(time.RFC3339),
				"schema_ver":   string(record.SchemaVersion),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("cold storage put object: %w", err)
	}

	return nil
}

// Get загружает событие из S3 по ID.
func (cs *ColdStorage) Get(ctx context.Context, eventID string) (*EventRecord, error) {
	// Поиск по всем source-префиксам
	sources := []EventSource{SourceAlarms, SourceCMMS, SourcePredictions, SourceTelemetry, SourceAudit}

	for _, source := range sources {
		// Ищем по всем возможным датам (последние 5 лет)
		prefix := fmt.Sprintf("events/%s/", source)
		record, err := cs.findByID(ctx, prefix, eventID)
		if err == nil && record != nil {
			return record, nil
		}
	}

	return nil, fmt.Errorf("cold storage: event %s not found", eventID)
}

// List возвращает список событий из S3 по заданным фильтрам.
func (cs *ColdStorage) List(ctx context.Context, opts ColdListOptions) ([]*EventRecord, error) {
	prefix := fmt.Sprintf("events/%s/", opts.Source)

	records := make([]*EventRecord, 0)

	// Создаём контекст с пагинацией
	listCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	objCh := cs.client.ListObjects(listCtx, cs.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
		MaxKeys:   1000,
	})

	for obj := range objCh {
		if obj.Err != nil {
			cs.logger.Error("cold storage list error", "prefix", prefix, "error", obj.Err)
			continue
		}

		// Проверяем контекст
		select {
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}

		// Фильтрация по метаданным (быстрая, без загрузки тела)
		if !cs.matchesMetadataFilter(obj, opts) {
			continue
		}

		// Загружаем полный объект
		record, err := cs.getObject(ctx, obj.Key)
		if err != nil {
			cs.logger.Warn("cold storage get failed", "key", obj.Key, "error", err)
			continue
		}

		// Дополнительная фильтрация по полям записи
		if !matchesFilter(record, RetrieveOptions{
			Source:      opts.Source,
			EventType:   opts.EventType,
			AggregateID: opts.AggregateID,
			Since:       opts.Since,
			Until:       opts.Until,
		}) {
			continue
		}

		records = append(records, record)

		// Лимит
		if opts.Limit > 0 && len(records) >= opts.Limit {
			break
		}
	}

	// Сортировка по timestamp
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	return records, nil
}

// DeleteOld удаляет события старше заданного периода.
// Используется вручную или через lifecycle policy.
func (cs *ColdStorage) DeleteOld(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	prefix := "events/"
	deleted := 0

	objCh := cs.client.ListObjects(ctx, cs.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			continue
		}

		if obj.LastModified.Before(cutoff) {
			if err := cs.client.RemoveObject(ctx, cs.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
				cs.logger.Warn("cold storage remove failed", "key", obj.Key, "error", err)
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

// ── Internal helpers ────────────────────────────────────────────────

// objectKey формирует S3 key для события.
//
// Формат: events/{source}/{year}/{month}/{day}/{event_id}.json
func (cs *ColdStorage) objectKey(record *EventRecord) string {
	return path.Join(
		"events",
		string(record.Source),
		fmt.Sprintf("%04d", record.Timestamp.Year()),
		fmt.Sprintf("%02d", record.Timestamp.Month()),
		fmt.Sprintf("%02d", record.Timestamp.Day()),
		record.ID+".json",
	)
}

// findByID ищет событие по ID внутри заданного префикса.
func (cs *ColdStorage) findByID(ctx context.Context, prefix, eventID string) (*EventRecord, error) {
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	objCh := cs.client.ListObjects(searchCtx, cs.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			continue
		}

		// Имя файла: {event_id}.json
		if strings.HasPrefix(obj.Key, prefix+eventID) {
			return cs.getObject(ctx, obj.Key)
		}
	}

	return nil, fmt.Errorf("event %s not found in %s", eventID, prefix)
}

// getObject загружает и десериализует объект из S3.
func (cs *ColdStorage) getObject(ctx context.Context, key string) (*EventRecord, error) {
	getCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	obj, err := cs.client.GetObject(getCtx, cs.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}

	var record EventRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", key, err)
	}

	return &record, nil
}

// matchesMetadataFilter проверяет соответствует ли объект фильтрам по метаданным.
func (cs *ColdStorage) matchesMetadataFilter(obj minio.ObjectInfo, opts ColdListOptions) bool {
	if opts.Source != "" {
		if src, ok := obj.UserMetadata["source"]; ok {
			if EventSource(src) != opts.Source {
				return false
			}
		}
	}

	if opts.EventType != "" {
		if et, ok := obj.UserMetadata["event_type"]; ok {
			if et != opts.EventType {
				return false
			}
		}
	}

	if opts.AggregateID != "" {
		if ag, ok := obj.UserMetadata["aggregate_id"]; ok {
			if ag != opts.AggregateID {
				return false
			}
		}
	}

	if !opts.Since.IsZero() {
		if obj.LastModified.Before(opts.Since) {
			return false
		}
	}

	if !opts.Until.IsZero() {
		if obj.LastModified.After(opts.Until) {
			return false
		}
	}

	return true
}

// ═══════════════════════════════════════════════════════════════════════
// Lifecycle Policy
// ═══════════════════════════════════════════════════════════════════════

// setLifecyclePolicy устанавливает lifecycle rule для автоматического удаления
// объектов старше указанного периода.
//
// Compliance: ISO 27001 A.8.10 (Information disposal)
func setLifecyclePolicy(ctx context.Context, client *minio.Client, bucket string, retention time.Duration) error {
	days := int(retention.Hours() / 24)

	config := lifecycle.NewConfiguration()
	config.Rules = []lifecycle.Rule{
		{
			ID:     "event-store-retention",
			Status: "Enabled",
			RuleFilter: lifecycle.Filter{
				Prefix: "events/",
			},
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(days),
			},
		},
	}

	return client.SetBucketLifecycle(ctx, bucket, config)
}

// ── Lifecycle ─────────────────────────────────────────────────────────

// Close закрывает соединение с S3 (no-op для minio.Client, но для интерфейса).
func (cs *ColdStorage) Close() error {
	cs.logger.Info("cold storage closed", "bucket", cs.bucket)
	return nil
}

// Stats возвращает статистику cold storage.
type ColdStorageStats struct {
	Bucket       string `json:"bucket"`
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	Retention    string `json:"retention"`
	ObjectCount  int    `json:"object_count,omitempty"`
	TotalSize    int64  `json:"total_size_bytes,omitempty"`
}

func (cs *ColdStorage) Stats(ctx context.Context) (*ColdStorageStats, error) {
	stats := &ColdStorageStats{
		Bucket:    cs.bucket,
		Endpoint:  cs.cfg.Endpoint,
		Region:    cs.cfg.Region,
		Retention: cs.cfg.Retention.String(),
	}

	objCh := cs.client.ListObjects(ctx, cs.bucket, minio.ListObjectsOptions{
		Prefix:    "events/",
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			continue
		}
		stats.ObjectCount++
		stats.TotalSize += obj.Size
	}

	return stats, nil
}
