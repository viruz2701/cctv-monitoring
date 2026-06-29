// Package compliance — Electronic Journal with HMAC-signed acts (P0-REG.4).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-REG.4: Electronic Journal + HMAC-signed acts
//
// ElectronicJournal — HMAC-signed журнал ТО.
// Обеспечивает tamper-evident audit trail для compliance-отчётности.
//
// Compliance:
//   - ISO 27001 A.12.4 (Logging and Monitoring — audit trail)
//   - ISO 27019 PCC.A.12 (ICS compliance logging)
//   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
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

	"gb-telemetry-collector/internal/audit"
)

// ═══════════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════════

// JournalEntryType — тип записи в журнале.
type JournalEntryType string

const (
	JournalEntryWOGenerated JournalEntryType = "wo_generated"
	JournalEntryActCreated  JournalEntryType = "act_created"
	JournalEntryActSigned   JournalEntryType = "act_signed"
	JournalEntryActVerified JournalEntryType = "act_verified"
	JournalEntryWOCompleted JournalEntryType = "wo_completed"
)

// ═══════════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════════

// JournalEntry представляет запись в compliance_journal.
type JournalEntry struct {
	ID            string     `json:"id"`
	RegulationID  *string    `json:"regulation_id,omitempty"`
	WoID          *string    `json:"wo_id,omitempty"`
	RegionCode    string     `json:"region_code"`
	ActData       []byte     `json:"act_data"`
	HMACSignature *string    `json:"hmac_signature,omitempty"`
	HMACSignedAt  *time.Time `json:"hmac_signed_at,omitempty"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	VerifiedBy    *string    `json:"verified_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ActData представляет данные акта ТО.
type ActData struct {
	Action          string                 `json:"action"`
	EntryType       JournalEntryType       `json:"entry_type"`
	RegulationCode  string                 `json:"regulation_code,omitempty"`
	RegulationName  string                 `json:"regulation_name,omitempty"`
	TechnicianID    string                 `json:"technician_id,omitempty"`
	TechnicianName  string                 `json:"technician_name,omitempty"`
	ChecklistResult []ChecklistItemResult  `json:"checklist_result,omitempty"`
	Notes           string                 `json:"notes,omitempty"`
	Attachments     []string               `json:"attachments,omitempty"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
	Timestamp       string                 `json:"timestamp"`
	TraceID         string                 `json:"trace_id,omitempty"`
}

// ChecklistItemResult представляет результат пункта чеклиста.
type ChecklistItemResult struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
	Status      string `json:"status"` // passed, failed, skipped
	Comment     string `json:"comment,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════
// ElectronicJournal — HMAC-signed журнал ТО.
// ═══════════════════════════════════════════════════════════════════════════

// ElectronicJournal — HMAC-signed журнал ТО.
type ElectronicJournal struct {
	db     *pgxpool.Pool
	signer *audit.Signer
	logger *slog.Logger
}

// NewElectronicJournal создаёт новый ElectronicJournal.
func NewElectronicJournal(db *pgxpool.Pool, signer *audit.Signer, logger *slog.Logger) *ElectronicJournal {
	if logger == nil {
		logger = slog.Default()
	}
	return &ElectronicJournal{
		db:     db,
		signer: signer,
		logger: logger.With("component", "compliance.electronic_journal"),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// SignAct — подписать акт HMAC
// ═══════════════════════════════════════════════════════════════════════════

// SignAct подписывает акт HMAC и возвращает верификационный код.
// Соответствует: ISO 27001 A.12.4, СТБ 34.101.27 п. 7.2
func (ej *ElectronicJournal) SignAct(ctx context.Context, entryID string) (string, error) {
	// Получаем запись из БД
	entry, err := ej.getEntry(ctx, entryID)
	if err != nil {
		return "", fmt.Errorf("sign act: get entry: %w", err)
	}

	// Если уже подписано — возвращаем существующую подпись
	if entry.HMACSignature != nil {
		return *entry.HMACSignature, nil
	}

	// Создаём данные для подписи: act_data + created_at
	signData := string(entry.ActData) + entry.CreatedAt.UTC().Format(time.RFC3339)

	// Вычисляем HMAC
	signature := ej.signer.Sign(signData)

	// Сохраняем подпись в БД
	now := time.Now().UTC()
	_, err = ej.db.Exec(ctx, `
		UPDATE compliance_journal
		SET hmac_signature = $1, hmac_signed_at = $2
		WHERE id = $3 AND hmac_signature IS NULL
	`, signature, now, entryID)
	if err != nil {
		return "", fmt.Errorf("sign act: update signature: %w", err)
	}

	ej.logger.Info("act signed",
		"entry_id", entryID,
		"region", entry.RegionCode,
	)

	return signature, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// VerifyAct — проверка HMAC подписи акта
// ═══════════════════════════════════════════════════════════════════════════

// VerifyAct проверяет HMAC подпись акта.
// Возвращает true если подпись валидна, false если нет.
// Соответствует: ISO 27001 A.12.4.2, СТБ 34.101.27 п. 7.2
func (ej *ElectronicJournal) VerifyAct(ctx context.Context, entryID string) (bool, error) {
	entry, err := ej.getEntry(ctx, entryID)
	if err != nil {
		return false, fmt.Errorf("verify act: get entry: %w", err)
	}

	if entry.HMACSignature == nil {
		return false, fmt.Errorf("verify act: entry %s has no signature", entryID)
	}

	// Восстанавливаем данные для подписи
	signData := string(entry.ActData) + entry.CreatedAt.UTC().Format(time.RFC3339)

	// Верифицируем
	valid := ej.signer.Verify(signData, *entry.HMACSignature)

	// Логируем результат верификации
	if valid {
		ej.logger.Info("act signature verified",
			"entry_id", entryID,
			"region", entry.RegionCode,
		)
	} else {
		ej.logger.Warn("act signature verification FAILED",
			"entry_id", entryID,
			"region", entry.RegionCode,
			"signature", *entry.HMACSignature,
		)
	}

	return valid, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// CreateEntry — создать запись в журнале
// ═══════════════════════════════════════════════════════════════════════════

// CreateEntry создаёт новую запись в compliance_journal.
func (ej *ElectronicJournal) CreateEntry(
	ctx context.Context,
	regulationID, woID, regionCode string,
	actData *ActData,
) (string, error) {
	if actData == nil {
		actData = &ActData{
			Action:    "created",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	if actData.Timestamp == "" {
		actData.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if actData.Action == "" {
		actData.Action = "created"
	}

	actDataJSON, err := json.Marshal(actData)
	if err != nil {
		return "", fmt.Errorf("create entry: marshal act data: %w", err)
	}

	var id string
	err = ej.db.QueryRow(ctx, `
		INSERT INTO compliance_journal (regulation_id, wo_id, region_code, act_data)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, regulationID, woID, regionCode, json.RawMessage(actDataJSON)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create entry: insert: %w", err)
	}

	ej.logger.Info("journal entry created",
		"entry_id", id,
		"region", regionCode,
		"regulation_id", regulationID,
		"wo_id", woID,
	)

	return id, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ListEntries — список записей журнала
// ═══════════════════════════════════════════════════════════════════════════

// ListEntries возвращает записи compliance_journal с фильтрацией.
func (ej *ElectronicJournal) ListEntries(
	ctx context.Context,
	regionCode string,
	from, to *time.Time,
	limit, offset int,
) ([]JournalEntry, int, error) {
	// Сначала считаем общее количество
	countQuery := `SELECT COUNT(*) FROM compliance_journal WHERE 1=1`
	countArgs := make([]interface{}, 0)
	argIdx := 1

	if regionCode != "" {
		countQuery += fmt.Sprintf(" AND region_code = $%d", argIdx)
		countArgs = append(countArgs, regionCode)
		argIdx++
	}
	if from != nil {
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		countArgs = append(countArgs, *from)
		argIdx++
	}
	if to != nil {
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		countArgs = append(countArgs, *to)
		argIdx++
	}

	var total int
	if err := ej.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("list entries: count: %w", err)
	}

	// Получаем данные
	query := `SELECT id, regulation_id, wo_id, region_code, act_data,
		hmac_signature, hmac_signed_at, verified_at, verified_by, created_at
		FROM compliance_journal WHERE 1=1`

	args := make([]interface{}, 0)
	argIdx = 1

	if regionCode != "" {
		query += fmt.Sprintf(" AND region_code = $%d", argIdx)
		args = append(args, regionCode)
		argIdx++
	}
	if from != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *to)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
		argIdx++
	}

	rows, err := ej.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list entries: query: %w", err)
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var e JournalEntry
		var regID, woID *string
		var hmacSig, verifiedBy *string
		var hmacSignedAt, verifiedAt *time.Time

		if err := rows.Scan(
			&e.ID, &regID, &woID, &e.RegionCode, &e.ActData,
			&hmacSig, &hmacSignedAt, &verifiedAt, &verifiedBy, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("list entries: scan: %w", err)
		}

		e.RegulationID = regID
		e.WoID = woID
		e.HMACSignature = hmacSig
		e.HMACSignedAt = hmacSignedAt
		e.VerifiedAt = verifiedAt
		e.VerifiedBy = verifiedBy

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list entries: rows: %w", err)
	}

	return entries, total, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// GetEntry — получить запись журнала по ID
// ═══════════════════════════════════════════════════════════════════════════

// getEntry возвращает запись compliance_journal по ID.
func (ej *ElectronicJournal) getEntry(ctx context.Context, id string) (*JournalEntry, error) {
	var e JournalEntry
	var regID, woID *string
	var hmacSig, verifiedBy *string
	var hmacSignedAt, verifiedAt *time.Time

	err := ej.db.QueryRow(ctx, `
		SELECT id, regulation_id, wo_id, region_code, act_data,
			hmac_signature, hmac_signed_at, verified_at, verified_by, created_at
		FROM compliance_journal
		WHERE id = $1
	`, id).Scan(
		&e.ID, &regID, &woID, &e.RegionCode, &e.ActData,
		&hmacSig, &hmacSignedAt, &verifiedAt, &verifiedBy, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get entry %s: %w", id, err)
	}

	e.RegulationID = regID
	e.WoID = woID
	e.HMACSignature = hmacSig
	e.HMACSignedAt = hmacSignedAt
	e.VerifiedAt = verifiedAt
	e.VerifiedBy = verifiedBy

	return &e, nil
}
