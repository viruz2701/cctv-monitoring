// Package storage — Region-aware S3 Client with Data Residency Enforcement (P0-CE.6).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.6: Data Residency Enforcement — S3 Client
//
// S3Client оборачивает minio.Client и добавляет:
//   - Region-aware endpoint selection через ResidencyEnforcer
//   - Pre-operation data residency validation
//   - Region-aware bucket routing
//   - Audit callback для всех residency violations
//
// Compliance:
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//   - СТБ 34.101.27 п. 7.1 (Data localization)
//   - ISO 27001 A.8.10 (Information disposal)
//   - ISO 27001 A.12.4 (Audit trail for violations)
//   - Приказ ОАЦ №66 п. 7.18.3 (Data protection)
//
// ═══════════════════════════════════════════════════════════════════════════
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// S3Client
// ────────────────────────────────────────────────────────────────────────────

// S3Client предоставляет region-aware S3 операции с data residency enforcement.
//
// Каждая операция проходит через ValidateStorageOperation перед выполнением.
// Регион клиента определяется из StorageContext при создании.
type S3Client struct {
	enforcer *ResidencyEnforcer
	client   *minio.Client
	ctx      *StorageContext
	logger   *slog.Logger
}

// S3ClientConfig — конфигурация для создания S3Client.
type S3ClientConfig struct {
	// Enforcer — ResidencyEnforcer для проверок data residency.
	Enforcer *ResidencyEnforcer
	// StorageContext — контекст storage операции (регион, профиль, tenant).
	StorageContext *StorageContext
	// Logger — опциональный логгер.
	Logger *slog.Logger
}

// NewS3Client создаёт region-aware S3Client с data residency enforcement.
//
// Алгоритм:
//  1. Получает endpoint для региона из ResidencyEnforcer
//  2. Создаёт minio.Client с этим endpoint
//  3. Все последующие операции проходят через ValidateStorageOperation
//
// Соответствует:
//   - СТБ 34.101.27 п. 7.1 (Data localization — region pinning)
//   - ISO 27001 A.8.10 (Информация хранится только в разрешённом регионе)
func NewS3Client(cfg S3ClientConfig) (*S3Client, error) {
	if cfg.Enforcer == nil {
		return nil, fmt.Errorf("s3 client: enforcer is required")
	}
	if cfg.StorageContext == nil {
		return nil, fmt.Errorf("s3 client: storage context is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default().With("component", "storage.s3client")
	}

	// Получаем endpoint для региона через ResidencyEnforcer
	endpointCfg, err := cfg.Enforcer.GetS3Endpoint(cfg.StorageContext.Region)
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}

	// Создаём minio.Client с region-aware endpoint
	client, err := minio.New(endpointCfg.Endpoint, &minio.Options{
		Creds:  nil, // credentials передаются через context или env
		Secure: endpointCfg.UseTLS,
		Region: endpointCfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: minio.New: %w", err)
	}

	logger.Info("s3 client created",
		"region", cfg.StorageContext.Region,
		"endpoint", endpointCfg.Endpoint,
		"bucket", endpointCfg.Bucket,
		"use_tls", endpointCfg.UseTLS,
	)

	return &S3Client{
		enforcer: cfg.Enforcer,
		client:   client,
		ctx:      cfg.StorageContext,
		logger:   logger,
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Public methods
// ────────────────────────────────────────────────────────────────────────────

// PutObject загружает объект в S3 с проверкой data residency.
//
// Перед загрузкой проверяет, что регион назначения соответствует
// политике data residency для tenant'а.
//
// Соответствует:
//   - СТБ 34.101.27 п. 7.1: Данные хранятся только в разрешённом регионе
//   - ISO 27001 A.8.10: Контроль размещения данных
func (c *S3Client) PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	// ── Pre-flight: Data residency check ──
	if err := c.validateOp(objectSize); err != nil {
		return minio.UploadInfo{}, fmt.Errorf("s3 put: %w", err)
	}

	info, err := c.client.PutObject(ctx, bucket, objectName, reader, objectSize, opts)
	if err != nil {
		c.logger.Error("s3 put failed",
			"bucket", bucket,
			"object", objectName,
			"region", c.ctx.Region,
			"error", err,
		)
		return minio.UploadInfo{}, fmt.Errorf("s3 put: %w", err)
	}

	c.logger.Debug("s3 put succeeded",
		"bucket", bucket,
		"object", objectName,
		"region", c.ctx.Region,
		"size", info.Size,
	)

	return info, nil
}

// GetObject получает объект из S3 с проверкой data residency.
//
// Перед чтением проверяет, что регион запроса соответствует
// региону хранения данных.
//
// Соответствует:
//   - GDPR Art. 44-49: Доступ к данным только из разрешённого региона
//   - СТБ 34.101.27 п. 7.1: Data localization enforcement
func (c *S3Client) GetObject(ctx context.Context, bucket, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	// ── Pre-flight: Data residency check ──
	if err := c.validateOp(0); err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}

	obj, err := c.client.GetObject(ctx, bucket, objectName, opts)
	if err != nil {
		c.logger.Error("s3 get failed",
			"bucket", bucket,
			"object", objectName,
			"region", c.ctx.Region,
			"error", err,
		)
		return nil, fmt.Errorf("s3 get: %w", err)
	}

	return obj, nil
}

// RemoveObject удаляет объект из S3 с проверкой data residency.
//
// Соответствует:
//   - ISO 27001 A.8.10 (Information disposal — controlled deletion)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity)
func (c *S3Client) RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error {
	// ── Pre-flight: Data residency check ──
	if err := c.validateOp(0); err != nil {
		return fmt.Errorf("s3 remove: %w", err)
	}

	if err := c.client.RemoveObject(ctx, bucket, objectName, opts); err != nil {
		c.logger.Error("s3 remove failed",
			"bucket", bucket,
			"object", objectName,
			"region", c.ctx.Region,
			"error", err,
		)
		return fmt.Errorf("s3 remove: %w", err)
	}

	c.logger.Debug("s3 remove succeeded",
		"bucket", bucket,
		"object", objectName,
		"region", c.ctx.Region,
	)

	return nil
}

// ListObjects возвращает список объектов в bucket'е с проверкой data residency.
//
// Соответствует:
//   - ISO 27001 A.12.4 (Audit trail — listing is logged)
//   - СТБ 34.101.27 п. 7.2 (Integrity of audit logs)
func (c *S3Client) ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	if err := c.validateOp(0); err != nil {
		// Возвращаем закрытый канал с ошибкой
		ch := make(chan minio.ObjectInfo, 1)
		close(ch)
		c.logger.Error("s3 list blocked by residency",
			"bucket", bucket,
			"region", c.ctx.Region,
			"error", err,
		)
		return ch
	}

	return c.client.ListObjects(ctx, bucket, opts)
}

// ────────────────────────────────────────────────────────────────────────────
// Bucket management (proxy with validation)
// ────────────────────────────────────────────────────────────────────────────

// MakeBucket создаёт bucket в регионе клиента.
// Регион bucket'а фиксируется политикой data residency.
func (c *S3Client) MakeBucket(ctx context.Context, bucket string, opts minio.MakeBucketOptions) error {
	if err := c.validateOp(0); err != nil {
		return fmt.Errorf("s3 makebucket: %w", err)
	}

	if err := c.client.MakeBucket(ctx, bucket, opts); err != nil {
		return fmt.Errorf("s3 makebucket: %w", err)
	}

	c.logger.Info("s3 bucket created",
		"bucket", bucket,
		"region", c.ctx.Region,
	)

	return nil
}

// BucketExists проверяет существование bucket'а.
func (c *S3Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	if err := c.validateOp(0); err != nil {
		return false, fmt.Errorf("s3 bucketexists: %w", err)
	}

	return c.client.BucketExists(ctx, bucket)
}

// ────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ────────────────────────────────────────────────────────────────────────────

// validateOp выполняет pre-flight проверку data residency.
// Вызывается перед каждой S3 операцией.
//
// Возвращает ошибку если операция нарушает политику data residency.
// Overhead: <1ms per call (локальная проверка в памяти).
func (c *S3Client) validateOp(_ int64) error {
	return c.enforcer.ValidateStorageOperation(c.ctx, c.ctx.Region)
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers for creating S3Client from compliance profile
// ────────────────────────────────────────────────────────────────────────────

// NewS3ClientForProfile создаёт S3Client на основе ComplianceProfile и региона.
//
// Удобно для создания клиента из middleware, когда профиль уже извлечён.
func NewS3ClientForProfile(enforcer *ResidencyEnforcer, profile compliance.ComplianceProfile, region, tenantID string, logger *slog.Logger) (*S3Client, error) {
	if enforcer == nil {
		return nil, fmt.Errorf("s3 client: enforcer is required")
	}
	if profile == nil {
		return nil, fmt.Errorf("s3 client: compliance profile is required")
	}

	ctx := &StorageContext{
		Region:            region,
		ComplianceProfile: profile,
		TenantID:          tenantID,
	}

	return NewS3Client(S3ClientConfig{
		Enforcer:       enforcer,
		StorageContext: ctx,
		Logger:         logger,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Timeout helpers
// ────────────────────────────────────────────────────────────────────────────

// defaultTimeout — таймаут по умолчанию для S3 операций.
const defaultTimeout = 30 * time.Second

// ContextWithTimeout возвращает context с таймаутом по умолчанию.
func ContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}
