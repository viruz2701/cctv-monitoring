// Package compliance — Maintenance Compliance Engine (P0-REG.3-5).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-REG.3: Auto-generation WO из compliance schedules
//
// RegulatoryCron читает maintenance_regulations и создаёт WO по расписанию.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
//   - ISO 27001 A.12.4 (Logging and Monitoring)
//   - ISO 27019 PCC.A.12 (ICS compliance logging)
//   - СТБ 34.101.27 п. 7.2 (Защита журналов аудита)
//   - Приказ ОАЦ № 66 п. 7.18.3 (Tamper-evident logging)
//   - OWASP ASVS V7 (Log content and integrity)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ═══════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════

// DueRegulation представляет просроченный регламент ТО.
type DueRegulation struct {
	ID                  string     `json:"id"`
	RegionCode          string     `json:"region_code"`
	RegulationCode      string     `json:"regulation_code"`
	Name                string     `json:"name"`
	RegulationType      string     `json:"regulation_type"`
	IntervalMonths      int        `json:"interval_months"`
	EstimatedMinutes    int        `json:"estimated_minutes"`
	TotalItems          int        `json:"total_items"`
	ComplianceStandards []string   `json:"compliance_standards"`
	LicenseRequirements *string    `json:"license_requirements,omitempty"`
	DocsRequired        []byte     `json:"docs_required"`
	LastMaintenanceDate *time.Time `json:"last_maintenance_date,omitempty"`
	DaysOverdue         int        `json:"days_overdue"`
}

// WOProvider — интерфейс для создания work orders (абстракция над CMMS).
type WOProvider interface {
	CreateWO(ctx context.Context, regulation *DueRegulation) (string, error)
}

// RegulatoryAuditLogger — интерфейс для логирования в compliance_journal.
type RegulatoryAuditLogger interface {
	LogRegulatoryAction(ctx context.Context, action, regulationID, woID, regionCode string, details map[string]interface{}) error
}

// ═══════════════════════════════════════════════════════════════════════════
// RegulatoryCron
// ═══════════════════════════════════════════════════════════════════════════

// RegulatoryCron читает maintenance_regulations и создаёт WO по расписанию.
type RegulatoryCron struct {
	db          *pgxpool.Pool
	woProvider  WOProvider
	auditLogger RegulatoryAuditLogger
	logger      *slog.Logger
}

// NewRegulatoryCron создаёт новый RegulatoryCron.
func NewRegulatoryCron(
	db *pgxpool.Pool,
	woProvider WOProvider,
	auditLogger RegulatoryAuditLogger,
	logger *slog.Logger,
) *RegulatoryCron {
	if logger == nil {
		logger = slog.Default()
	}
	return &RegulatoryCron{
		db:          db,
		woProvider:  woProvider,
		auditLogger: auditLogger,
		logger:      logger.With("component", "compliance.regulatory_cron"),
	}
}

// Run проверяет все active regulations и создаёт WO для просроченных/подходящих.
//
// Flow:
//  1. Выбрать все due regulations через get_due_regulations()
//  2. Для каждой: создать WO через woProvider
//  3. Логировать в compliance_journal через auditLogger
//  4. Обновить last_maintenance_date в compliance_journal
func (rc *RegulatoryCron) Run(ctx context.Context) error {
	rc.logger.Info("regulatory cron: checking due regulations")

	regulations, err := rc.getDueRegulations(ctx)
	if err != nil {
		return fmt.Errorf("regulatory cron: get due regulations: %w", err)
	}

	if len(regulations) == 0 {
		rc.logger.Info("regulatory cron: no due regulations found")
		return nil
	}

	rc.logger.Info("regulatory cron: found due regulations",
		"count", len(regulations),
	)

	created := 0
	errors := 0

	for _, reg := range regulations {
		select {
		case <-ctx.Done():
			return fmt.Errorf("regulatory cron: context cancelled: %w", ctx.Err())
		default:
		}

		// Создаём WO
		woID, err := rc.woProvider.CreateWO(ctx, &reg)
		if err != nil {
			rc.logger.Error("regulatory cron: failed to create WO",
				"regulation_id", reg.ID,
				"regulation_code", reg.RegulationCode,
				"error", err,
			)
			errors++
			continue
		}

		// Логируем в compliance_journal
		details := map[string]interface{}{
			"regulation_code": reg.RegulationCode,
			"regulation_name": reg.Name,
			"regulation_type": reg.RegulationType,
			"region_code":     reg.RegionCode,
			"interval_months": reg.IntervalMonths,
			"estimated_min":   reg.EstimatedMinutes,
			"days_overdue":    reg.DaysOverdue,
			"action":          "auto_generate_wo",
		}

		if err := rc.auditLogger.LogRegulatoryAction(
			ctx, "auto_generate_wo", reg.ID, woID, reg.RegionCode, details,
		); err != nil {
			rc.logger.Error("regulatory cron: failed to log audit",
				"regulation_id", reg.ID,
				"wo_id", woID,
				"error", err,
			)
			// Не фатально, WO уже создан
		}

		created++
		rc.logger.Info("regulatory cron: WO created",
			"regulation_id", reg.ID,
			"regulation_code", reg.RegulationCode,
			"wo_id", woID,
			"region", reg.RegionCode,
			"days_overdue", reg.DaysOverdue,
		)
	}

	rc.logger.Info("regulatory cron: completed",
		"total", len(regulations),
		"created", created,
		"errors", errors,
	)

	return nil
}

// getDueRegulations получает просроченные регламенты через SQL функцию.
func (rc *RegulatoryCron) getDueRegulations(ctx context.Context) ([]DueRegulation, error) {
	rows, err := rc.db.Query(ctx, `SELECT * FROM get_due_regulations()`)
	if err != nil {
		return nil, fmt.Errorf("query due regulations: %w", err)
	}
	defer rows.Close()

	var regulations []DueRegulation
	for rows.Next() {
		var reg DueRegulation
		var docsReq []byte

		if err := rows.Scan(
			&reg.ID,
			&reg.RegionCode,
			&reg.RegulationCode,
			&reg.Name,
			&reg.RegulationType,
			&reg.IntervalMonths,
			&reg.EstimatedMinutes,
			&reg.TotalItems,
			&reg.ComplianceStandards,
			&reg.LicenseRequirements,
			&docsReq,
			&reg.LastMaintenanceDate,
			&reg.DaysOverdue,
		); err != nil {
			return nil, fmt.Errorf("scan due regulation: %w", err)
		}

		reg.DocsRequired = docsReq
		regulations = append(regulations, reg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return regulations, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// DefaultWOProvider — реализация WOProvider через прямое SQL
// ═══════════════════════════════════════════════════════════════════════════

// DefaultWOProvider создаёт work orders напрямую через SQL.
type DefaultWOProvider struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewDefaultWOProvider создаёт новый DefaultWOProvider.
func NewDefaultWOProvider(db *pgxpool.Pool, logger *slog.Logger) *DefaultWOProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultWOProvider{
		db:     db,
		logger: logger.With("component", "compliance.wo_provider"),
	}
}

// CreateWO создаёт work order для регламента.
func (p *DefaultWOProvider) CreateWO(ctx context.Context, reg *DueRegulation) (string, error) {
	// Формируем описание work order на основе данных регламента
	notes := fmt.Sprintf(
		"[Auto] %s (%s)\nРегламент: %s\nТип ТО: %s\nИнтервал: %d мес.\nПросрочено: %d дн.\nСтандарты: %v",
		reg.Name,
		reg.RegulationCode,
		reg.ID,
		reg.RegulationType,
		reg.IntervalMonths,
		reg.DaysOverdue,
		reg.ComplianceStandards,
	)

	checklist := p.buildChecklist(reg)

	// Сериализуем чеклист в JSON
	checklistJSON, err := json.Marshal(checklist)
	if err != nil {
		return "", fmt.Errorf("marshal checklist: %w", err)
	}

	var woID string
	err = p.db.QueryRow(ctx, `
		INSERT INTO work_orders (
			type, status, priority, title, description, checklist,
			estimated_minutes, notes, created_by, schedule_id
		) VALUES (
			'preventive', 'open', 'high',
			$1, $2, $3,
			$4, $5, 'system', $6
		)
		RETURNING id
	`,
		fmt.Sprintf("ТО: %s (%s)", reg.Name, reg.RegulationCode),
		fmt.Sprintf("Автоматически создано из регламента %s. Просрочено на %d дн.", reg.RegulationCode, reg.DaysOverdue),
		checklistJSON,
		reg.EstimatedMinutes,
		notes,
		reg.ID,
	).Scan(&woID)

	if err != nil {
		return "", fmt.Errorf("insert work order: %w", err)
	}

	p.logger.Info("WO created from regulation",
		"wo_id", woID,
		"regulation_code", reg.RegulationCode,
		"region", reg.RegionCode,
	)

	return woID, nil
}

// buildChecklist формирует чеклист из данных регламента.
func (p *DefaultWOProvider) buildChecklist(reg *DueRegulation) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, reg.TotalItems)

	// Базовые пункты на основе типа ТО
	baseItems := []string{
		"Проверка работоспособности оборудования",
		"Визуальный осмотр состояния",
		"Проверка целостности соединений",
		"Очистка оборудования",
	}

	for i, item := range baseItems {
		items = append(items, map[string]interface{}{
			"order":       i + 1,
			"description": item,
			"category":    "inspection",
			"is_required": true,
			"status":      "pending",
		})
	}

	// Добавляем compliance-специфичные пункты
	complianceItems := []string{
		"Проверка соответствия требованиям " + reg.RegulationCode,
		"Аудит логов безопасности",
		"Проверка целостности данных",
	}

	for i, item := range complianceItems {
		items = append(items, map[string]interface{}{
			"order":       len(baseItems) + i + 1,
			"description": item,
			"category":    "compliance",
			"is_required": true,
			"status":      "pending",
		})
	}

	// Если есть docs_required — добавляем пункты документации
	if len(reg.DocsRequired) > 0 {
		var docs []string
		if err := json.Unmarshal(reg.DocsRequired, &docs); err == nil {
			for i, doc := range docs {
				items = append(items, map[string]interface{}{
					"order":       len(baseItems) + len(complianceItems) + i + 1,
					"description": "Оформление: " + doc,
					"category":    "documentation",
					"is_required": true,
					"status":      "pending",
				})
			}
		}
	}

	return items
}

// ═══════════════════════════════════════════════════════════════════════════
// AuditLogger — реализация RegulatoryAuditLogger
// ═══════════════════════════════════════════════════════════════════════════

// AuditLogger логирует compliance действия в compliance_journal.
type AuditLogger struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewRegulatoryAuditLogger создаёт новый AuditLogger.
func NewRegulatoryAuditLogger(db *pgxpool.Pool, logger *slog.Logger) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditLogger{
		db:     db,
		logger: logger.With("component", "compliance.audit_logger"),
	}
}

// LogRegulatoryAction записывает действие в compliance_journal.
func (l *AuditLogger) LogRegulatoryAction(
	ctx context.Context,
	action, regulationID, woID, regionCode string,
	details map[string]interface{},
) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	_, err = l.db.Exec(ctx, `
		INSERT INTO compliance_journal (regulation_id, wo_id, region_code, act_data)
		VALUES ($1, $2, $3, $4)
	`,
		regulationID,
		woID,
		regionCode,
		json.RawMessage(detailsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert compliance journal: %w", err)
	}

	l.logger.Info("regulatory action logged",
		"action", action,
		"regulation_id", regulationID,
		"wo_id", woID,
		"region", regionCode,
	)

	return nil
}
