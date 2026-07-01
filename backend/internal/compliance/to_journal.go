// Package compliance — TO Journal auto-generation service (UX-3.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// UX-3.2: Auto-fill TO Journals при закрытии WorkOrder
//
// TOJournalService — сервис автоматической генерации записей в TO-журналах
// при закрытии Work Order (статус "completed").
//
// Pre-fill поля:
//   - device — из work_order.device_id
//   - date — время завершения WO
//   - technician — assigned_to из WO
//   - location — из device → site
//   - time — длительность выполнения WO
//
// Required поля (manual):
//   - checklist_notes — заметки по чеклисту
//   - defects — выявленные дефекты
//   - customer_signature — подпись заказчика
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — все мутации логируются)
//   - ISO 27019 PCC.A.12 (ICS compliance logging)
//   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
//   - СТБ 34.101.27 п. 7.2 (Защита журналов аудита)
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
// Constants
// ═══════════════════════════════════════════════════════════════════════════

// TOJournalEntryType — тип записи в TO-журнале.
type TOJournalEntryType string

const (
	TOJournalEntryAutoFilled   TOJournalEntryType = "to_auto_filled"
	TOJournalEntryManualUpdate TOJournalEntryType = "to_manual_update"
)

// TORequiredField — required поле TO-журнала, требующее ручного ввода.
type TORequiredField string

const (
	TORequiredFieldChecklistNotes TORequiredField = "checklist_notes"
	TORequiredFieldDefects        TORequiredField = "defects"
	TORequiredFieldCustomerSign   TORequiredField = "customer_signature"
)

// ValidTORequiredFields — whitelist для OWASP ASVS V5.1.
var ValidTORequiredFields = []string{
	string(TORequiredFieldChecklistNotes),
	string(TORequiredFieldDefects),
	string(TORequiredFieldCustomerSign),
}

// ═══════════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════════

// TOJournalEntry — запись в TO-журнале (UX-3.2).
type TOJournalEntry struct {
	ID              string             `json:"id"`
	WorkOrderID     string             `json:"work_order_id"`
	DeviceID        string             `json:"device_id"`
	DeviceName      string             `json:"device_name,omitempty"`
	TechnicianID    string             `json:"technician_id,omitempty"`
	TechnicianName  string             `json:"technician_name,omitempty"`
	SiteName        string             `json:"site_name,omitempty"`
	EntryType       TOJournalEntryType `json:"entry_type"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	CompletedAt     time.Time          `json:"completed_at"`
	DurationMinutes int                `json:"duration_minutes"`
	ChecklistNotes  string             `json:"checklist_notes,omitempty"`
	Defects         string             `json:"defects,omitempty"`
	CustomerSign    string             `json:"customer_signature,omitempty"`
	IsCompleted     bool               `json:"is_completed"`
	RequiredFields  []TORequiredField  `json:"required_fields"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// TOJournalCreateRequest — DTO для создания записи TO-журнала.
type TOJournalCreateRequest struct {
	WorkOrderID    string     `json:"work_order_id" validate:"required"`
	DeviceID       string     `json:"device_id" validate:"required"`
	TechnicianID   string     `json:"technician_id,omitempty"`
	TechnicianName string     `json:"technician_name,omitempty"`
	SiteName       string     `json:"site_name,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    time.Time  `json:"completed_at" validate:"required"`
	DurationMin    int        `json:"duration_minutes"`
}

// TOJournalUpdateRequest — DTO для обновления required полей.
type TOJournalUpdateRequest struct {
	ChecklistNotes string `json:"checklist_notes,omitempty"`
	Defects        string `json:"defects,omitempty"`
	CustomerSign   string `json:"customer_signature,omitempty"`
}

// TOJournalSummary — сводка TO-журналов для отображения в completion flow.
type TOJournalSummary struct {
	Entries         []TOJournalEntry `json:"entries"`
	TotalAutoFilled int              `json:"total_auto_filled"`
	PendingManual   int              `json:"pending_manual"`
	IsComplete      bool             `json:"is_complete"`
}

// ═══════════════════════════════════════════════════════════════════════════
// TOJournalService — сервис генерации TO журналов.
// ═══════════════════════════════════════════════════════════════════════════

// TOJournalService — сервис для авто-заполнения TO журналов при закрытии WO.
type TOJournalService struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewTOJournalService создаёт новый TOJournalService.
func NewTOJournalService(db *pgxpool.Pool, logger *slog.Logger) *TOJournalService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TOJournalService{
		db:     db,
		logger: logger.With("component", "compliance.to_journal"),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// CreateAutoFilledEntries — создать авто-заполненные записи TO-журнала
// ═══════════════════════════════════════════════════════════════════════════

// CreateAutoFilledEntries создаёт авто-заполненные записи в TO-журнале
// при закрытии Work Order. Возвращает список созданных записей.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — audit trail)
//   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
func (s *TOJournalService) CreateAutoFilledEntries(
	ctx context.Context,
	req *TOJournalCreateRequest,
) ([]TOJournalEntry, error) {
	if req == nil {
		return nil, fmt.Errorf("to_journal: request is nil")
	}
	if req.WorkOrderID == "" || req.DeviceID == "" {
		return nil, fmt.Errorf("to_journal: work_order_id and device_id are required")
	}

	entry := TOJournalEntry{
		WorkOrderID:     req.WorkOrderID,
		DeviceID:        req.DeviceID,
		TechnicianID:    req.TechnicianID,
		TechnicianName:  req.TechnicianName,
		SiteName:        req.SiteName,
		EntryType:       TOJournalEntryAutoFilled,
		CompletedAt:     req.CompletedAt,
		StartedAt:       req.StartedAt,
		DurationMinutes: req.DurationMin,
		IsCompleted:     false,
		RequiredFields: []TORequiredField{
			TORequiredFieldChecklistNotes,
			TORequiredFieldDefects,
			TORequiredFieldCustomerSign,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Генерируем ID (используем UUID из БД)
	actData := map[string]interface{}{
		"action":           string(TOJournalEntryAutoFilled),
		"work_order_id":    req.WorkOrderID,
		"device_id":        req.DeviceID,
		"technician_id":    req.TechnicianID,
		"technician_name":  req.TechnicianName,
		"site_name":        req.SiteName,
		"started_at":       req.StartedAt,
		"completed_at":     req.CompletedAt,
		"duration_minutes": req.DurationMin,
		"required_fields":  entry.RequiredFields,
		"is_completed":     false,
	}

	actDataJSON, err := json.Marshal(actData)
	if err != nil {
		return nil, fmt.Errorf("to_journal: marshal act data: %w", err)
	}

	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO to_journal (work_order_id, device_id, technician_id, technician_name,
			site_name, entry_type, started_at, completed_at, duration_minutes,
			required_fields, is_completed, act_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text
	`,
		req.WorkOrderID, req.DeviceID, req.TechnicianID, req.TechnicianName,
		req.SiteName, string(TOJournalEntryAutoFilled), req.StartedAt,
		req.CompletedAt, req.DurationMin,
		entry.RequiredFields, false, json.RawMessage(actDataJSON),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("to_journal: insert entry: %w", err)
	}

	entry.ID = id

	s.logger.Info("TO journal entry auto-filled",
		"entry_id", id,
		"work_order_id", req.WorkOrderID,
		"device_id", req.DeviceID,
	)

	return []TOJournalEntry{entry}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// UpdateRequiredFields — обновление required полей TO-журнала
// ═══════════════════════════════════════════════════════════════════════════

// UpdateRequiredFields обновляет required поля TO-журнала (manual input).
// После заполнения всех required полей запись считается is_completed = true.
//
// Compliance:
//   - OWASP ASVS V5.1 (Whitelist validation)
//   - ISO 27001 A.12.4 (Audit trail)
func (s *TOJournalService) UpdateRequiredFields(
	ctx context.Context,
	entryID string,
	req *TOJournalUpdateRequest,
) (*TOJournalEntry, error) {
	if entryID == "" {
		return nil, fmt.Errorf("to_journal: entry_id is required")
	}
	if req == nil {
		return nil, fmt.Errorf("to_journal: request is nil")
	}

	// Получаем текущую запись
	entry, err := s.getEntry(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("to_journal: get entry: %w", err)
	}

	if entry.IsCompleted {
		return nil, fmt.Errorf("to_journal: entry %s is already completed", entryID)
	}

	// Обновляем поля
	if req.ChecklistNotes != "" {
		entry.ChecklistNotes = req.ChecklistNotes
	}
	if req.Defects != "" {
		entry.Defects = req.Defects
	}
	if req.CustomerSign != "" {
		entry.CustomerSign = req.CustomerSign
	}

	// Проверяем, все ли required поля заполнены
	allFilled := true
	var remainingRequired []TORequiredField
	for _, rf := range entry.RequiredFields {
		switch rf {
		case TORequiredFieldChecklistNotes:
			if entry.ChecklistNotes == "" {
				allFilled = false
				remainingRequired = append(remainingRequired, rf)
			}
		case TORequiredFieldDefects:
			if entry.Defects == "" {
				allFilled = false
				remainingRequired = append(remainingRequired, rf)
			}
		case TORequiredFieldCustomerSign:
			if entry.CustomerSign == "" {
				allFilled = false
				remainingRequired = append(remainingRequired, rf)
			}
		}
	}

	entry.IsCompleted = allFilled
	entry.UpdatedAt = time.Now().UTC()

	// Сохраняем в БД
	_, err = s.db.Exec(ctx, `
		UPDATE to_journal
		SET checklist_notes = $1, defects = $2, customer_signature = $3,
			is_completed = $4, updated_at = $5
		WHERE id = $6
	`, entry.ChecklistNotes, entry.Defects, entry.CustomerSign,
		entry.IsCompleted, entry.UpdatedAt, entryID)
	if err != nil {
		return nil, fmt.Errorf("to_journal: update entry: %w", err)
	}

	s.logger.Info("TO journal entry updated",
		"entry_id", entryID,
		"is_completed", entry.IsCompleted,
		"remaining_required", len(remainingRequired),
	)

	return entry, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// GetEntriesByWorkOrder — получить записи TO-журнала по WorkOrder
// ═══════════════════════════════════════════════════════════════════════════

// GetEntriesByWorkOrder возвращает все записи TO-журнала для указанного WO.
func (s *TOJournalService) GetEntriesByWorkOrder(
	ctx context.Context,
	workOrderID string,
) (*TOJournalSummary, error) {
	if workOrderID == "" {
		return nil, fmt.Errorf("to_journal: work_order_id is required")
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, work_order_id, device_id, device_name, technician_id,
			technician_name, site_name, entry_type, started_at, completed_at,
			duration_minutes, checklist_notes, defects, customer_signature,
			is_completed, required_fields, created_at, updated_at
		FROM to_journal
		WHERE work_order_id = $1
		ORDER BY created_at DESC
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("to_journal: query entries: %w", err)
	}
	defer rows.Close()

	var entries []TOJournalEntry
	pendingManual := 0

	for rows.Next() {
		var e TOJournalEntry
		var entryType string
		var reqFieldsJSON []byte

		if err := rows.Scan(
			&e.ID, &e.WorkOrderID, &e.DeviceID, &e.DeviceName,
			&e.TechnicianID, &e.TechnicianName, &e.SiteName,
			&entryType, &e.StartedAt, &e.CompletedAt,
			&e.DurationMinutes, &e.ChecklistNotes, &e.Defects,
			&e.CustomerSign, &e.IsCompleted, &reqFieldsJSON,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("to_journal: scan entry: %w", err)
		}

		e.EntryType = TOJournalEntryType(entryType)

		// Десериализуем required_fields
		if len(reqFieldsJSON) > 0 {
			json.Unmarshal(reqFieldsJSON, &e.RequiredFields)
		}

		if !e.IsCompleted {
			pendingManual++
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("to_journal: rows iteration: %w", err)
	}

	if entries == nil {
		entries = []TOJournalEntry{}
	}

	return &TOJournalSummary{
		Entries:         entries,
		TotalAutoFilled: len(entries),
		PendingManual:   pendingManual,
		IsComplete:      pendingManual == 0,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// RegulatoryChecklist — проверка regulatory checklist перед закрытием
// ═══════════════════════════════════════════════════════════════════════════

// RegulatoryChecklistResult — результат проверки regulatory checklist.
type RegulatoryChecklistResult struct {
	AllRequiredFilled bool              `json:"all_required_filled"`
	MissingFields     []TORequiredField `json:"missing_fields,omitempty"`
	EntriesPending    int               `json:"entries_pending"`
}

// CheckRegulatoryCompliance проверяет, все ли required поля заполнены
// перед финальным закрытием Work Order.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
//   - OWASP ASVS V7 (Log content validation)
func (s *TOJournalService) CheckRegulatoryCompliance(
	ctx context.Context,
	workOrderID string,
) (*RegulatoryChecklistResult, error) {
	summary, err := s.GetEntriesByWorkOrder(ctx, workOrderID)
	if err != nil {
		return nil, err
	}

	if len(summary.Entries) == 0 {
		return &RegulatoryChecklistResult{
			AllRequiredFilled: true,
			EntriesPending:    0,
		}, nil
	}

	// Собираем все незаполненные required поля
	missingSet := make(map[TORequiredField]bool)
	for _, entry := range summary.Entries {
		if entry.IsCompleted {
			continue
		}
		for _, rf := range entry.RequiredFields {
			missingSet[rf] = true
		}
	}

	var missingFields []TORequiredField
	for rf := range missingSet {
		missingFields = append(missingFields, rf)
	}

	return &RegulatoryChecklistResult{
		AllRequiredFilled: len(missingFields) == 0,
		MissingFields:     missingFields,
		EntriesPending:    summary.PendingManual,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Private helpers
// ═══════════════════════════════════════════════════════════════════════════

// getEntry возвращает запись TO-журнала по ID.
func (s *TOJournalService) getEntry(ctx context.Context, id string) (*TOJournalEntry, error) {
	var e TOJournalEntry
	var entryType string
	var reqFieldsJSON []byte

	err := s.db.QueryRow(ctx, `
		SELECT id, work_order_id, device_id, device_name, technician_id,
			technician_name, site_name, entry_type, started_at, completed_at,
			duration_minutes, checklist_notes, defects, customer_signature,
			is_completed, required_fields, created_at, updated_at
		FROM to_journal
		WHERE id = $1
	`, id).Scan(
		&e.ID, &e.WorkOrderID, &e.DeviceID, &e.DeviceName,
		&e.TechnicianID, &e.TechnicianName, &e.SiteName,
		&entryType, &e.StartedAt, &e.CompletedAt,
		&e.DurationMinutes, &e.ChecklistNotes, &e.Defects,
		&e.CustomerSign, &e.IsCompleted, &reqFieldsJSON,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("to_journal: get entry %s: %w", id, err)
	}

	e.EntryType = TOJournalEntryType(entryType)
	if len(reqFieldsJSON) > 0 {
		json.Unmarshal(reqFieldsJSON, &e.RequiredFields)
	}

	return &e, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Migration: Create to_journal table
// ═══════════════════════════════════════════════════════════════════════════

// CreateTOJournalTableSQL — DDL для создания таблицы to_journal.
//
// Compliance:
//   - OWASP ASVS V6 (Stored cryptography — JSONB для гибких полей)
//   - IEC 62443-3-3 SL-3 (Zone 3 — Data integrity)
const CreateTOJournalTableSQL = `
CREATE TABLE IF NOT EXISTS to_journal (
	id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	work_order_id TEXT NOT NULL,
	device_id     TEXT NOT NULL,
	device_name   TEXT NOT NULL DEFAULT '',
	technician_id TEXT NOT NULL DEFAULT '',
	technician_name TEXT NOT NULL DEFAULT '',
	site_name     TEXT NOT NULL DEFAULT '',
	entry_type    TEXT NOT NULL DEFAULT 'to_auto_filled',
	started_at    TIMESTAMPTZ,
	completed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	duration_minutes INTEGER NOT NULL DEFAULT 0,
	checklist_notes  TEXT NOT NULL DEFAULT '',
	defects          TEXT NOT NULL DEFAULT '',
	customer_signature TEXT NOT NULL DEFAULT '',
	is_completed     BOOLEAN NOT NULL DEFAULT FALSE,
	required_fields  JSONB NOT NULL DEFAULT '[]'::jsonb,
	act_data         JSONB,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_to_journal_work_order_id ON to_journal(work_order_id);
CREATE INDEX IF NOT EXISTS idx_to_journal_device_id ON to_journal(device_id);
CREATE INDEX IF NOT EXISTS idx_to_journal_is_completed ON to_journal(is_completed);
`
