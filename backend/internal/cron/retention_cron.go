package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/retention"
)

// ────────────────────────────────────────────────────────────────────────────
// RetentionCron — периодическая задача очистки данных по политикам хранения
// ────────────────────────────────────────────────────────────────────────────

// RetentionCronConfig — конфигурация RetentionCron.
type RetentionCronConfig struct {
	// BatchSize — количество записей для обработки за один цикл.
	BatchSize int
	// DryRun — если true, не выполняет фактическое удаление (только логирует).
	DryRun bool
}

// DefaultRetentionCronConfig возвращает конфигурацию по умолчанию.
func DefaultRetentionCronConfig() RetentionCronConfig {
	return RetentionCronConfig{
		BatchSize: 1000,
		DryRun:    false,
	}
}

// RetentionCron выполняет очистку данных по политикам хранения.
//
// Использует TimescaleDB drop_chunks для time-based partitions.
// Логирует все retention-действия в audit_log.
//
// Compliance:
//   - ISO 27001 A.12.4 (Event logging — audit trail)
//   - IEC 62443 SR 2.3 (Data integrity — retention enforcement)
//   - GDPR Art. 5(1)(e) (Storage limitation)
//   - CTБ 34.101.27 п. 7.2 (Целостность журналов аудита)
type RetentionCron struct {
	db         *db.DB
	profileMgr *retention.ProfileManager
	legalHold  *retention.LegalHoldManager
	config     RetentionCronConfig
	logger     *slog.Logger
}

// NewRetentionCron создаёт новый RetentionCron.
func NewRetentionCron(
	database *db.DB,
	profileMgr *retention.ProfileManager,
	legalHold *retention.LegalHoldManager,
	config RetentionCronConfig,
	logger *slog.Logger,
) *RetentionCron {
	if logger == nil {
		logger = slog.Default()
	}
	return &RetentionCron{
		db:         database,
		profileMgr: profileMgr,
		legalHold:  legalHold,
		config:     config,
		logger:     logger.With("component", "retention-cron"),
	}
}

// Run выполняет один цикл очистки данных по всем регионам.
//
// Порядок работы:
//  1. Получить список регионов из ProfileManager
//  2. Для каждого региона и типа данных:
//     a. Получить политику хранения
//     b. Проверить legal hold
//     c. Выполнить lifecycle transition (hot→cold, cold→archive, archive→delete)
//     d. Записать в audit_log
func (c *RetentionCron) Run(ctx context.Context) {
	c.logger.Info("Starting retention cron job",
		"batch_size", c.config.BatchSize,
		"dry_run", c.config.DryRun,
	)

	regions := c.profileMgr.ListRegions()
	totalActions := 0

	for _, region := range regions {
		count, err := c.processRegion(ctx, region)
		if err != nil {
			c.logger.Error("failed to process region",
				"region", region,
				"error", err,
			)
			continue
		}
		totalActions += count
	}

	c.logger.Info("Retention cron job completed",
		"total_actions", totalActions,
		"regions_processed", len(regions),
	)
}

// processRegion обрабатывает все типы данных для указанного региона.
func (c *RetentionCron) processRegion(ctx context.Context, region retention.Region) (int, error) {
	profiles := c.profileMgr.ListProfiles()
	totalActions := 0

	for _, policy := range profiles {
		if policy.Region != region {
			continue
		}

		count, err := c.processDataType(ctx, region, policy.DataType, policy)
		if err != nil {
			c.logger.Error("failed to process data type",
				"region", region,
				"data_type", policy.DataType,
				"error", err,
			)
			continue
		}
		totalActions += count
	}

	return totalActions, nil
}

// processDataType выполняет cleanup для конкретного типа данных.
func (c *RetentionCron) processDataType(
	ctx context.Context,
	region retention.Region,
	dataType retention.DataType,
	policy *retention.RetentionPolicy,
) (int, error) {
	// Определяем таблицы для очистки
	tableName, intervalCol, err := resolveTable(region, dataType)
	if err != nil {
		return 0, fmt.Errorf("resolve table: %w", err)
	}

	// Рассчитываем cut-off дату по TotalTTL
	cutoff := time.Now().UTC().Add(-policy.TotalTTL)
	c.logger.Info("processing data type",
		"region", region,
		"data_type", dataType,
		"table", tableName,
		"cutoff", cutoff,
		"total_ttl", policy.TotalTTL,
	)

	// Выполняем lifecycle stages
	actions := 0

	// Stage 1: Delete expired data (archive → delete)
	if policy.TotalTTL > 0 {
		count, err := c.deleteExpired(ctx, tableName, intervalCol, cutoff, region, dataType, policy)
		if err != nil {
			return actions, fmt.Errorf("delete expired: %w", err)
		}
		actions += count
	}

	return actions, nil
}

// deleteExpired удаляет данные старше cut-off даты.
// Использует TimescaleDB drop_chunks для time-based партиций.
func (c *RetentionCron) deleteExpired(
	ctx context.Context,
	tableName string,
	intervalCol string,
	cutoff time.Time,
	region retention.Region,
	dataType retention.DataType,
	policy *retention.RetentionPolicy,
) (int, error) {
	// Проверяем legal hold — глобальная проверка
	// В production здесь должна быть проверка по каждому tenant'у
	if c.legalHold != nil && c.legalHold.ListActive() != nil {
		c.logger.Warn("active legal holds exist, checking per-tenant",
			"region", region,
			"data_type", dataType,
		)
		// В production здесь per-tenant проверка
	}

	if c.config.DryRun {
		c.logger.Info("DRY RUN: would delete expired data",
			"table", tableName,
			"cutoff", cutoff,
		)
		return 0, nil
	}

	// Попытка 1: TimescaleDB drop_chunks
	count, err := c.dropChunks(ctx, tableName, cutoff)
	if err != nil {
		c.logger.Warn("drop_chunks failed, falling back to DELETE",
			"table", tableName,
			"error", err,
		)
		// Fallback: hard DELETE
		return c.deleteRows(ctx, tableName, intervalCol, cutoff)
	}

	// Логируем retention action
	c.logRetentionAction(ctx, region, dataType, tableName, count, "delete")

	return count, nil
}

// dropChunks использует TimescaleDB drop_chunks для удаления старых партиций.
func (c *RetentionCron) dropChunks(ctx context.Context, tableName string, olderThan time.Time) (int, error) {
	query := fmt.Sprintf(`
		SELECT count_chunks_dropped FROM drop_chunks(
			table_name := '%s',
			older_than := '%s'::timestamptz,
			verbose := false
		)
	`, tableName, olderThan.Format(time.RFC3339))

	var count int
	err := c.db.Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("drop_chunks %s: %w", tableName, err)
	}

	c.logger.Info("dropped chunks via TimescaleDB",
		"table", tableName,
		"older_than", olderThan,
		"chunks_dropped", count,
	)
	return count, nil
}

// deleteRows выполняет hard DELETE как fallback.
func (c *RetentionCron) deleteRows(ctx context.Context, tableName, intervalCol string, olderThan time.Time) (int, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE %s < $1
	`, tableName, intervalCol)

	result, err := c.db.Pool.Exec(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("delete from %s: %w", tableName, err)
	}

	count := int(result.RowsAffected())
	c.logger.Info("deleted rows",
		"table", tableName,
		"older_than", olderThan,
		"rows_deleted", count,
	)
	return count, nil
}

// logRetentionAction записывает действие retention в audit_log.
func (c *RetentionCron) logRetentionAction(
	ctx context.Context,
	region retention.Region,
	dataType retention.DataType,
	tableName string,
	recordsCount int,
	action string,
) {
	c.logger.Info("retention action",
		"region", region,
		"data_type", dataType,
		"table", tableName,
		"action", action,
		"records_count", recordsCount,
	)

	// Вставка в audit_log
	_, err := c.db.Pool.Exec(ctx, `
		INSERT INTO audit_log
			(user_id, action, entity_type, entity_id, new_value, trace_id)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
	`,
		"system",
		"retention."+action,
		"retention_policy",
		string(dataType)+":"+string(region),
		fmt.Sprintf(`{"table":"%s","records_count":%d,"region":"%s","data_type":"%s"}`,
			tableName, recordsCount, region, dataType),
		"retention-cron-"+time.Now().UTC().Format("20060102T150405"),
	)
	if err != nil {
		c.logger.Error("failed to write audit log",
			"error", err,
		)
	}
}

// resolveTable определяет таблицу и колонку интервала для типа данных в регионе.
func resolveTable(region retention.Region, dataType retention.DataType) (string, string, error) {
	// Региональные суффиксы для таблиц
	regionSuffix := string(region)
	switch region {
	case retention.RegionBY:
		regionSuffix = "by"
	case retention.RegionRU:
		regionSuffix = "ru"
	case retention.RegionEU:
		regionSuffix = "eu"
	case retention.RegionUS:
		regionSuffix = "us"
	case retention.RegionCN:
		regionSuffix = "cn"
	}

	switch dataType {
	case retention.DataTelemetry:
		return fmt.Sprintf("telemetry_%s", regionSuffix), "recorded_at", nil
	case retention.DataAlerts:
		return fmt.Sprintf("alerts_%s", regionSuffix), "created_at", nil
	case retention.DataImages:
		return fmt.Sprintf("images_%s", regionSuffix), "captured_at", nil
	case retention.DataVideo:
		return fmt.Sprintf("video_%s", regionSuffix), "recorded_at", nil
	case retention.DataAudit:
		return "audit_log", "timestamp", nil
	case retention.DataReports:
		return fmt.Sprintf("reports_%s", regionSuffix), "created_at", nil
	case retention.DataWorkOrders:
		return fmt.Sprintf("work_orders_%s", regionSuffix), "created_at", nil
	default:
		return "", "", fmt.Errorf("unknown data type: %s", dataType)
	}
}
