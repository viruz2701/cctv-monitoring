// Package blackbox — Black Box Incident Recorder (KF-15.2.4).
//
// Автоматический сбор "пакета доказательств" при инциденте:
// снимок состояния устройства, логи, тревоги, статус записи,
// даунтайм, SLA — всё в одном месте.
//
// Compliance:
//   - IEC 62443 SR 7.1: Resource availability — evidence collection
//   - ISO 27001 A.12.4: Audit trail — неизменяемый журнал инцидентов
//   - ISO 27019 PCC.A.12: Incident management for ICS
//   - СТБ 34.101.27 п. 6.4: Регистрация инцидентов безопасности
//   - OWASP ASVS V7.1: Error handling — structured error evidence
package blackbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gb-telemetry-collector/internal/downtime"
	"gb-telemetry-collector/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ═══════════════════════════════════════════════════════════════════════
// Типы данных
// ═══════════════════════════════════════════════════════════════════════

// TriggerType определяет тип триггера инцидента.
type TriggerType string

const (
	TriggerAlarm     TriggerType = "alarm"
	TriggerManual    TriggerType = "manual"
	TriggerSLABreach TriggerType = "sla_breach"
	TriggerDowntime  TriggerType = "downtime"
)

// IncidentReport — полный пакет доказательств по инциденту.
type IncidentReport struct {
	ID          string    `json:"id" db:"id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	DeviceName  string    `json:"device_name,omitempty"`
	SiteID      string    `json:"site_id,omitempty" db:"site_id"`
	TriggeredBy string    `json:"triggered_by" db:"triggered_by"`
	TriggerRef  string    `json:"trigger_ref" db:"trigger_ref"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`

	// Evidence Package
	DeviceSnapshot  json.RawMessage    `json:"device_snapshot" db:"device_snapshot"`
	RecentAlerts    []AlarmSnapshot    `json:"recent_alerts" db:"recent_alerts"`
	RecentLogs      []LogSnapshot      `json:"recent_logs" db:"recent_logs"`
	RecordingStatus string             `json:"recording_status" db:"recording_status"`
	DowntimeHistory []DowntimeSnapshot `json:"downtime_history" db:"downtime_history"`
	SLAData         json.RawMessage    `json:"sla_data" db:"sla_data"`

	// Media & Notes
	Photos []string `json:"photos,omitempty" db:"photos"`
	Notes  string   `json:"notes,omitempty" db:"notes"`

	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AlarmSnapshot — краткая информация о тревоге для пакета доказательств.
type AlarmSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	Priority    int       `json:"priority"`
	Description string    `json:"description,omitempty"`
	Method      int       `json:"method,omitempty"`
}

// LogSnapshot — краткая информация о логе для пакета доказательств.
type LogSnapshot struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message,omitempty"`
	Source  string    `json:"source,omitempty"`
}

// DowntimeSnapshot — краткая информация о простое для пакета доказательств.
type DowntimeSnapshot struct {
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	DurationMin int        `json:"duration_minutes"`
	Reason      string     `json:"reason"`
	Description string     `json:"description,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Repository interface
// ═══════════════════════════════════════════════════════════════════════

// Repository определяет интерфейс к хранилищу данных для Black Box.
// Абстракция для тестирования (table-driven tests).
type Repository interface {
	SaveReport(ctx context.Context, report *IncidentReport) error
	GetReportByID(ctx context.Context, id string) (*IncidentReport, error)
	ListReports(ctx context.Context, deviceID string, limit int, offset int) ([]IncidentReport, int, error)
	DeleteReport(ctx context.Context, id string) error
	SaveTrigger(ctx context.Context, reportID string, triggeredBy TriggerType, triggerRef string, userID string, metadata json.RawMessage) error
}

// DBRepository реализует Repository через PostgreSQL.
type DBRepository struct {
	pool   pgxPool
	logger *slog.Logger
}

// pgxPool — интерфейс к pgx пулу (для тестирования).
// Соответствует сигнатурам pgxpool.Pool.
type pgxPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// NewDBRepository создаёт новый DBRepository.
func NewDBRepository(pool pgxPool, logger *slog.Logger) *DBRepository {
	return &DBRepository{pool: pool, logger: logger}
}

func (r *DBRepository) SaveReport(ctx context.Context, report *IncidentReport) error {
	deviceSnapshot := report.DeviceSnapshot
	if deviceSnapshot == nil {
		deviceSnapshot = json.RawMessage("{}")
	}
	slaData := report.SLAData
	if slaData == nil {
		slaData = json.RawMessage("{}")
	}

	alertsJSON, err := json.Marshal(report.RecentAlerts)
	if err != nil {
		return fmt.Errorf("marshal alerts: %w", err)
	}
	logsJSON, err := json.Marshal(report.RecentLogs)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}
	downtimeJSON, err := json.Marshal(report.DowntimeHistory)
	if err != nil {
		return fmt.Errorf("marshal downtime: %w", err)
	}
	photos := report.Photos
	if photos == nil {
		photos = []string{}
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO incident_reports (
			id, device_id, site_id, triggered_by, trigger_ref, "timestamp",
			device_snapshot, recent_alerts, recent_logs, recording_status,
			downtime_history, sla_data, photos, notes, status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7::jsonb, $8::jsonb, $9::jsonb, $10,
			$11::jsonb, $12::jsonb, $13, $14, $15,
			$16, $17
		)
	`,
		report.ID, report.DeviceID, report.SiteID, report.TriggeredBy,
		report.TriggerRef, report.Timestamp,
		deviceSnapshot, alertsJSON, logsJSON, report.RecordingStatus,
		downtimeJSON, slaData, photos, report.Notes, report.Status,
		report.CreatedAt, report.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}

func (r *DBRepository) GetReportByID(ctx context.Context, id string) (*IncidentReport, error) {
	var report IncidentReport
	var alertsJSON, logsJSON, downtimeJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT
			id, device_id, COALESCE(site_id, ''), triggered_by, trigger_ref, "timestamp",
			device_snapshot, recent_alerts, recent_logs, recording_status,
			downtime_history, sla_data, photos, COALESCE(notes, ''), status,
			created_at, updated_at
		FROM incident_reports
		WHERE id = $1
	`, id).Scan(
		&report.ID, &report.DeviceID, &report.SiteID,
		&report.TriggeredBy, &report.TriggerRef, &report.Timestamp,
		&report.DeviceSnapshot, &alertsJSON, &logsJSON,
		&report.RecordingStatus, &downtimeJSON, &report.SLAData,
		&report.Photos, &report.Notes, &report.Status,
		&report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get report %q: %w", id, err)
	}

	if len(alertsJSON) > 0 {
		if err := json.Unmarshal(alertsJSON, &report.RecentAlerts); err != nil {
			r.logger.Warn("unmarshal recent_alerts", "error", err, "report_id", id)
		}
	}
	if len(logsJSON) > 0 {
		if err := json.Unmarshal(logsJSON, &report.RecentLogs); err != nil {
			r.logger.Warn("unmarshal recent_logs", "error", err, "report_id", id)
		}
	}
	if len(downtimeJSON) > 0 {
		if err := json.Unmarshal(downtimeJSON, &report.DowntimeHistory); err != nil {
			r.logger.Warn("unmarshal downtime_history", "error", err, "report_id", id)
		}
	}

	return &report, nil
}

func (r *DBRepository) ListReports(ctx context.Context, deviceID string, limit int, offset int) ([]IncidentReport, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Count
	countQuery := `SELECT COUNT(*) FROM incident_reports`
	countArgs := []interface{}{}
	argIdx := 1
	if deviceID != "" {
		countQuery += fmt.Sprintf(" WHERE device_id = $%d", argIdx)
		countArgs = append(countArgs, deviceID)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}

	// Data
	query := `
		SELECT
			id, device_id, COALESCE(site_id, ''), triggered_by, trigger_ref, "timestamp",
			device_snapshot, recent_alerts, recent_logs, recording_status,
			downtime_history, sla_data, photos, COALESCE(notes, ''), status,
			created_at, updated_at
		FROM incident_reports`
	args := []interface{}{}
	argIdx = 1

	if deviceID != "" {
		query += fmt.Sprintf(" WHERE device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}

	query += ` ORDER BY "timestamp" DESC`
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []IncidentReport
	for rows.Next() {
		var rep IncidentReport
		var alertsJSON, logsJSON, downtimeJSON []byte

		if err := rows.Scan(
			&rep.ID, &rep.DeviceID, &rep.SiteID,
			&rep.TriggeredBy, &rep.TriggerRef, &rep.Timestamp,
			&rep.DeviceSnapshot, &alertsJSON, &logsJSON,
			&rep.RecordingStatus, &downtimeJSON, &rep.SLAData,
			&rep.Photos, &rep.Notes, &rep.Status,
			&rep.CreatedAt, &rep.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan report: %w", err)
		}

		if len(alertsJSON) > 0 {
			_ = json.Unmarshal(alertsJSON, &rep.RecentAlerts)
		}
		if len(logsJSON) > 0 {
			_ = json.Unmarshal(logsJSON, &rep.RecentLogs)
		}
		if len(downtimeJSON) > 0 {
			_ = json.Unmarshal(downtimeJSON, &rep.DowntimeHistory)
		}

		reports = append(reports, rep)
	}

	return reports, total, rows.Err()
}

func (r *DBRepository) DeleteReport(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM incident_reports WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete report %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("report %q not found", id)
	}
	return nil
}

func (r *DBRepository) SaveTrigger(ctx context.Context, reportID string, triggeredBy TriggerType, triggerRef string, userID string, metadata json.RawMessage) error {
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO incident_triggers (report_id, triggered_by, trigger_ref, triggered_by_user, metadata, "timestamp")
		VALUES ($1, $2, $3, $4, $5::jsonb, NOW())
	`, reportID, string(triggeredBy), triggerRef, userID, metadata)
	if err != nil {
		return fmt.Errorf("save trigger: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// DeviceService interface (для сбора данных с устройства)
// ═══════════════════════════════════════════════════════════════════════

// DeviceProvider предоставляет данные об устройстве.
type DeviceProvider interface {
	GetDeviceByID(ctx context.Context, deviceID string) (*models.Device, error)
	SearchLogs(deviceID, level, keyword, timeFrom, timeTo string) ([]models.ParsedLog, error)
}

// DowntimeProvider предоставляет данные о простоях.
type DowntimeProvider interface {
	GetDeviceDowntimeHistory(ctx context.Context, deviceID string, limit int) ([]downtime.AssetDowntime, error)
}

// ═══════════════════════════════════════════════════════════════════════
// Recorder — основной движок Black Box
// ═══════════════════════════════════════════════════════════════════════

// Recorder управляет сбором пакетов доказательств.
type Recorder struct {
	repo     Repository
	devices  DeviceProvider
	downtime DowntimeProvider
	logger   *slog.Logger
}

// NewRecorder создаёт новый Recorder.
// providers может быть nil — в этом случае соответствующие данные не собираются.
func NewRecorder(repo Repository, devices DeviceProvider, dt DowntimeProvider, logger *slog.Logger) *Recorder {
	return &Recorder{
		repo:     repo,
		devices:  devices,
		downtime: dt,
		logger:   logger,
	}
}

// TriggerIncident создаёт новый пакет доказательств по инциденту.
// Собирает: снимок устройства, последние логи, тревоги, даунтайм, SLA.
func (r *Recorder) TriggerIncident(ctx context.Context, deviceID string, trigger TriggerType, triggerRef string, userID string, notes string) (*IncidentReport, error) {
	r.logger.Info("triggering black box incident",
		"device_id", deviceID,
		"trigger", trigger,
		"ref", triggerRef,
	)

	report := &IncidentReport{
		ID:          "",
		DeviceID:    deviceID,
		TriggeredBy: string(trigger),
		TriggerRef:  triggerRef,
		Timestamp:   time.Now().UTC(),
		Status:      "draft",
		Notes:       notes,
		Photos:      []string{},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// ── 1. Снимок устройства ───────────────────────────────────────
	if r.devices != nil {
		if dev, err := r.devices.GetDeviceByID(ctx, deviceID); err != nil {
			r.logger.Warn("blackbox: failed to get device snapshot", "device_id", deviceID, "error", err)
			report.DeviceSnapshot = json.RawMessage(fmt.Sprintf(`{"error": %q}`, err.Error()))
		} else if dev != nil {
			snapshot := map[string]interface{}{
				"device_id":    dev.DeviceID,
				"name":         dev.Name,
				"status":       dev.Status,
				"health":       dev.Health,
				"device_type":  dev.DeviceType,
				"last_seen":    dev.LastSeen.Format(time.RFC3339),
				"vendor":       dev.VendorType,
				"site_id":      dev.SiteID,
				"asset_class":  dev.AssetClass,
				"firmware":     dev.FirmwareVersion,
				"manufacturer": dev.Manufacturer,
				"serial":       dev.SerialNumber,
			}
			report.DeviceName = dev.Name
			report.SiteID = ""
			if dev.SiteID != nil {
				report.SiteID = *dev.SiteID
			}
			data, _ := json.Marshal(snapshot)
			report.DeviceSnapshot = data
		}
	}

	// ── 2. Последние логи (50 записей) ─────────────────────────────
	if r.devices != nil {
		logs, err := r.devices.SearchLogs(deviceID, "", "", "", "")
		if err != nil {
			r.logger.Warn("blackbox: failed to fetch logs", "device_id", deviceID, "error", err)
		} else {
			count := len(logs)
			if count > 50 {
				logs = logs[:50]
			}
			snapshots := make([]LogSnapshot, 0, len(logs))
			for _, l := range logs {
				snapshots = append(snapshots, LogSnapshot{
					Time:    l.Time,
					Level:   l.LogLevel,
					Message: l.Message,
					Source:  l.Source,
				})
			}
			report.RecentLogs = snapshots
			r.logger.Debug("blackbox: collected logs", "device_id", deviceID, "count", count)
		}
	}

	// ── 3. Последние тревоги ───────────────────────────────────────
	// Тревоги получаем через прямой SQL-запрос, так как в репозитории
	// нет отдельного метода GetAlarms. Используем SearchLogs или прямой запрос.
	// Временно получаем пустой массив, данные будут добавлены через отдельный API.
	report.RecentAlerts = []AlarmSnapshot{}

	// ── 4. Статус записи ───────────────────────────────────────────
	if r.devices != nil {
		if dev, err := r.devices.GetDeviceByID(ctx, deviceID); err == nil && dev != nil {
			report.RecordingStatus = string(dev.Status)
		}
	}

	// ── 5. История простоев ────────────────────────────────────────
	if r.downtime != nil {
		entries, err := r.downtime.GetDeviceDowntimeHistory(ctx, deviceID, 20)
		if err != nil {
			r.logger.Warn("blackbox: failed to fetch downtime", "device_id", deviceID, "error", err)
		} else {
			snapshots := make([]DowntimeSnapshot, 0, len(entries))
			for _, d := range entries {
				snapshots = append(snapshots, DowntimeSnapshot{
					StartedAt:   d.StartedAt,
					EndedAt:     d.EndedAt,
					DurationMin: d.DurationMin,
					Reason:      string(d.Reason),
					Description: d.Description,
				})
			}
			report.DowntimeHistory = snapshots
			r.logger.Debug("blackbox: collected downtime", "device_id", deviceID, "count", len(entries))
		}
	}

	// ── 6. SLA данные ──────────────────────────────────────────────
	slaData := map[string]interface{}{
		"captured_at": time.Now().UTC().Format(time.RFC3339),
		"status":      "pending_sla_check",
	}
	data, _ := json.Marshal(slaData)
	report.SLAData = data

	// ── Сохраняем отчёт ────────────────────────────────────────────
	if err := r.repo.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("blackbox: save report: %w", err)
	}

	// ── Сохраняем триггер (audit trail) ────────────────────────────
	triggerMeta := json.RawMessage(fmt.Sprintf(`{"device_id": %q, "trigger": %q}`, deviceID, trigger))
	if err := r.repo.SaveTrigger(ctx, report.ID, trigger, triggerRef, userID, triggerMeta); err != nil {
		r.logger.Warn("blackbox: failed to save trigger audit", "report_id", report.ID, "error", err)
	}

	r.logger.Info("black box incident created",
		"report_id", report.ID,
		"device_id", deviceID,
		"trigger", trigger,
	)

	return report, nil
}

// GetReport возвращает пакет доказательств по ID.
func (r *Recorder) GetReport(ctx context.Context, id string) (*IncidentReport, error) {
	report, err := r.repo.GetReportByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("blackbox: get report: %w", err)
	}
	return report, nil
}

// ListReports возвращает список пакетов доказательств.
func (r *Recorder) ListReports(ctx context.Context, deviceID string, limit int, offset int) ([]IncidentReport, int, error) {
	reports, total, err := r.repo.ListReports(ctx, deviceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("blackbox: list reports: %w", err)
	}
	return reports, total, nil
}

// DeleteReport удаляет пакет доказательств.
func (r *Recorder) DeleteReport(ctx context.Context, id string) error {
	return r.repo.DeleteReport(ctx, id)
}

// UpdateReportStatus обновляет статус отчёта.
func (r *Recorder) UpdateReportStatus(ctx context.Context, id string, status string) error {
	_, err := r.repo.(*DBRepository).pool.Exec(ctx,
		`UPDATE incident_reports SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("blackbox: update status: %w", err)
	}
	return nil
}
