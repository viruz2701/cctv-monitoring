// Package audit — HMAC chain for audit log tamper detection.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-2: Audit Trail Compliance (ISO 27001 A.12.4)
//
// ChainStore обеспечивает:
//   - prev_hash chaining (каждая запись содержит HMAC предыдущей)
//   - TraceID для сквозного трейсинга
//   - Функция VerifyChain для проверки целостности
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - ISO 27001 A.12.4.2 (Protection of log information — chain)
//   - СТБ 34.101.27 п. 7.2 (Целостность журналов аудита)
//   - IEC 62443 SR 3.1 (Communication integrity — audit chain)
//
// ═══════════════════════════════════════════════════════════════════════════
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ────────────────────────────────────────────────────────────────────────────
// ChainStore — хранилище audit_log с HMAC chain
// ────────────────────────────────────────────────────────────────────────────

// ChainStore предоставляет методы для работы с audit_log цепочкой.
type ChainStore struct {
	pool   *pgxpool.Pool
	signer *Signer
}

// NewChainStore создаёт новый ChainStore.
func NewChainStore(pool *pgxpool.Pool, signer *Signer) *ChainStore {
	return &ChainStore{pool: pool, signer: signer}
}

// AuditEntry — полная запись аудита для вставки.
type AuditEntry struct {
	UserID     string
	Action     string
	EntityType string
	EntityID   string
	OldValue   interface{}
	NewValue   interface{}
	IPAddress  string
	UserAgent  string
	TraceID    string
}

// InsertWithChain вставляет запись в audit_log с prev_hash и hmac_signature.
func (s *ChainStore) InsertWithChain(ctx context.Context, entry *AuditEntry) error {
	var oldJSON, newJSON []byte
	var err error

	if entry.OldValue != nil {
		oldJSON, err = json.Marshal(entry.OldValue)
		if err != nil {
			return fmt.Errorf("audit: marshal old_value: %w", err)
		}
	}
	if entry.NewValue != nil {
		newJSON, err = json.Marshal(entry.NewValue)
		if err != nil {
			return fmt.Errorf("audit: marshal new_value: %w", err)
		}
	}

	// Получаем prev_hash последней записи
	prevHash, err := s.getLastHMAC(ctx)
	if err != nil {
		return fmt.Errorf("audit: get last hmac: %w", err)
	}

	// Формируем данные для подписи (включая prev_hash для chain)
	signData := SignAuditEntry(entry.UserID, entry.Action, entry.EntityType, entry.EntityID, oldJSON, newJSON)
	if prevHash != "" {
		signData = prevHash + "|" + signData
	}

	hmacSig := s.signer.Sign(signData)

	// Вставляем запись с prev_hash
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_log
			(user_id, action, entity_type, entity_id, old_value, new_value,
			 ip_address, user_agent, hmac_signature, prev_hash, trace_id)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $10, $11)
	`, entry.UserID, entry.Action, entry.EntityType, entry.EntityID,
		oldJSON, newJSON, entry.IPAddress, entry.UserAgent,
		hmacSig, prevHash, entry.TraceID)

	if err != nil {
		return fmt.Errorf("audit: insert: %w", err)
	}

	return nil
}

// VerifyEntry проверяет HMAC подпись конкретной записи.
func (s *ChainStore) VerifyEntry(ctx context.Context, entryID int64) (bool, error) {
	var userID, action, entityType, entityID string
	var oldVal, newVal []byte
	var hmacSig, prevHash string

	err := s.pool.QueryRow(ctx, `
		SELECT user_id, action, entity_type, entity_id,
		       COALESCE(old_value::text, ''),
		       COALESCE(new_value::text, ''),
		       COALESCE(hmac_signature, ''),
		       COALESCE(prev_hash, '')
		FROM audit_log WHERE id = $1
	`, entryID).Scan(&userID, &action, &entityType, &entityID, &oldVal, &newVal, &hmacSig, &prevHash)

	if err != nil {
		return false, fmt.Errorf("audit: get entry %d: %w", entryID, err)
	}

	signData := SignAuditEntry(userID, action, entityType, entityID, oldVal, newVal)
	if prevHash != "" {
		signData = prevHash + "|" + signData
	}

	return s.signer.Verify(signData, hmacSig), nil
}

// GetComplianceReport возвращает сводку compliance.
func (s *ChainStore) GetComplianceReport(ctx context.Context) (*ComplianceReport, error) {
	report := &ComplianceReport{}

	// Общее количество записей
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&report.TotalEntries)
	if err != nil {
		return nil, fmt.Errorf("audit: count entries: %w", err)
	}

	// Количество за последние 30 дней
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_log
		WHERE timestamp >= NOW() - INTERVAL '30 days'
	`).Scan(&report.Last30Days)
	if err != nil {
		return nil, fmt.Errorf("audit: count 30d: %w", err)
	}

	// Проверка целостности цепочки
	err = s.pool.QueryRow(ctx, `SELECT * FROM verify_audit_chain()`).Scan(
		&report.ChainBroken, &report.FirstBrokenID, &report.BrokenCount,
	)
	if err != nil {
		// Функция может не существовать на старых БД
		report.ChainCheckError = err.Error()
	}

	// Распределение по action
	rows, err := s.pool.Query(ctx, `
		SELECT action, COUNT(*) as cnt
		FROM audit_log
		GROUP BY action
		ORDER BY cnt DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("audit: action stats: %w", err)
	}
	defer rows.Close()

	report.ActionStats = make(map[string]int64)
	for rows.Next() {
		var action string
		var count int64
		if err := rows.Scan(&action, &count); err != nil {
			return nil, fmt.Errorf("audit: scan action: %w", err)
		}
		report.ActionStats[action] = count
	}

	// Дата самой старой записи
	err = s.pool.QueryRow(ctx, `SELECT MIN(timestamp) FROM audit_log`).Scan(&report.OldestEntry)
	if err != nil {
		report.OldestEntry = time.Now()
	}

	// Дата самой новой записи
	err = s.pool.QueryRow(ctx, `SELECT MAX(timestamp) FROM audit_log`).Scan(&report.NewestEntry)
	if err != nil {
		report.NewestEntry = time.Now()
	}

	return report, nil
}

// ── Internal ──────────────────────────────────────────────────────────────

func (s *ChainStore) getLastHMAC(ctx context.Context) (string, error) {
	var hmacSig string
	err := s.pool.QueryRow(ctx, `SELECT get_last_audit_hmac()`).Scan(&hmacSig)
	if err != nil {
		return "", nil // Функция может не существовать
	}
	return hmacSig, nil
}

// ── Compliance Report ─────────────────────────────────────────────────────

// ComplianceReport — отчёт о состоянии audit trail.
type ComplianceReport struct {
	TotalEntries    int64            `json:"total_entries"`
	Last30Days      int64            `json:"last_30_days"`
	ChainBroken     bool             `json:"chain_broken"`
	FirstBrokenID   *int64           `json:"first_broken_id,omitempty"`
	BrokenCount     int64            `json:"broken_count"`
	ChainCheckError string           `json:"chain_check_error,omitempty"`
	ActionStats     map[string]int64 `json:"action_stats"`
	OldestEntry     time.Time        `json:"oldest_entry"`
	NewestEntry     time.Time        `json:"newest_entry"`
}
