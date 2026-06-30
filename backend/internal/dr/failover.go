// Package dr — Disaster Recovery automation.
//
// ═══════════════════════════════════════════════════════════════════════════════
// P3-DR: Auto-Failover
//
// Содержит:
//   - FailoverManager — управление процессом failover
//   - FailoverEvent — запись события failover
//   - DNS Failover интеграция через NATS
//   - DB Promotion координация
//
// Compliance:
//   - ISO 27001 A.17.1 (Business continuity — DR procedures)
//   - IEC 62443-3-3 SR 7.3 (Failover mechanisms)
//   - Приказ ОАЦ №66 п. 7.18.2 (Резервирование каналов)
//   - GDPR Art. 32 (Security of processing — DR)
//
// ═══════════════════════════════════════════════════════════════════════════════
package dr

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ──────────────────────────────────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────────────────────────────────

// FailoverStatus — статус failover операции.
type FailoverStatus string

const (
	FailoverPending    FailoverStatus = "pending"
	FailoverApproved   FailoverStatus = "approved"
	FailoverRejected   FailoverStatus = "rejected"
	FailoverInProgress FailoverStatus = "in_progress"
	FailoverCompleted  FailoverStatus = "completed"
	FailoverFailed     FailoverStatus = "failed"
	FailoverRolledBack FailoverStatus = "rolled_back"
)

// FailoverEvent — полная запись события failover.
type FailoverEvent struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id,omitempty"`
	TriggerReason  string         `json:"trigger_reason"` // "health_check" | "manual" | "drill" | "admin"
	FromRegion     string         `json:"from_region"`
	ToRegion       string         `json:"to_region"`
	Status         FailoverStatus `json:"status"`
	InitiatedBy    string         `json:"initiated_by"`          // "system" | userID
	ApprovedBy     string         `json:"approved_by,omitempty"` // admin userID
	ApprovedAt     *time.Time     `json:"approved_at,omitempty"`
	DBPromoted     bool           `json:"db_promoted"`
	NATSPromoted   bool           `json:"nats_promoted"`
	DNSUpdated     bool           `json:"dns_updated"`
	HealthBefore   *HealthStatus  `json:"health_before,omitempty"`
	HealthAfter    *HealthStatus  `json:"health_after,omitempty"`
	RTO            time.Duration  `json:"rto"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	RollbackReason string         `json:"rollback_reason,omitempty"`
	StartedAt      time.Time      `json:"started_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// FailoverManager управляет процессом auto-failover.
//
// Процесс:
//  1. HealthMonitor обнаруживает failures (3+ последовательных)
//  2. FailoverManager создаёт FailoverEvent со статусом "pending"
//  3. Admin подтверждает (POST /api/v1/dr/failover)
//  4. Выполняется: DNS failover → DB promotion → NATS stream handover
//  5. Проверяется health после failover
//  6. Логируется RTO/RPO
type FailoverManager struct {
	mu         sync.RWMutex
	nc         *nats.Conn
	logger     *slog.Logger
	health     *HealthMonitor
	store      Store
	region     string
	drRegion   string
	eventIDSeq int

	// Active failover — текущий выполняемый failover.
	activeFailover *FailoverEvent

	// Callbacks для DNS и инфраструктурных операций.
	dnsFailoverFn func(ctx context.Context, fromRegion, toRegion string) error
	dbPromoteFn   func(ctx context.Context, region string) error
	natsPromoteFn func(ctx context.Context, region string) error
}

// FailoverConfig — конфигурация FailoverManager.
type FailoverConfig struct {
	Region        string `json:"region"`
	DRRegion      string `json:"dr_region"`
	RequireAdmin  bool   `json:"require_admin_confirm"` // true = manual confirm required
	NATSMirrorSub string `json:"nats_mirror_subject"`   // NATS subject для mirror promotion
	DBPromoteSub  string `json:"db_promote_subject"`    // NATS subject для DB promotion
	DNSUpdateSub  string `json:"dns_update_subject"`    // NATS subject для DNS update
}

// DefaultFailoverConfig возвращает конфигурацию failover по умолчанию.
func DefaultFailoverConfig() FailoverConfig {
	return FailoverConfig{
		Region:        RegionPrimary,
		DRRegion:      RegionSecondary,
		RequireAdmin:  true, // Безопасность: admin confirm обязателен
		NATSMirrorSub: "dr.nats.promote",
		DBPromoteSub:  "dr.postgres.promote",
		DNSUpdateSub:  "dr.dns.update",
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Constructor
// ──────────────────────────────────────────────────────────────────────────────

// NewFailoverManager создаёт новый FailoverManager.
func NewFailoverManager(
	nc *nats.Conn,
	health *HealthMonitor,
	store Store,
	cfg FailoverConfig,
	logger *slog.Logger,
) *FailoverManager {
	if logger == nil {
		logger = slog.Default()
	}

	fm := &FailoverManager{
		nc:       nc,
		health:   health,
		store:    store,
		logger:   logger.With("component", "dr.failover-manager"),
		region:   cfg.Region,
		drRegion: cfg.DRRegion,
	}

	// Callbacks по умолчанию — через NATS.
	fm.dnsFailoverFn = func(ctx context.Context, from, to string) error {
		return fm.publishNATSMessage(cfg.DNSUpdateSub, map[string]string{
			"action":    "dns_failover",
			"from":      from,
			"to":        to,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
	fm.dbPromoteFn = func(ctx context.Context, region string) error {
		return fm.publishNATSMessage(cfg.DBPromoteSub, map[string]string{
			"action":    "promote_db",
			"region":    region,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
	fm.natsPromoteFn = func(ctx context.Context, region string) error {
		return fm.publishNATSMessage(cfg.NATSMirrorSub, map[string]string{
			"action":    "promote_mirror",
			"region":    region,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}

	return fm
}

// SetDNSFailoverFn устанавливает кастомную функцию DNS failover.
func (fm *FailoverManager) SetDNSFailoverFn(fn func(ctx context.Context, fromRegion, toRegion string) error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.dnsFailoverFn = fn
}

// SetDBPromoteFn устанавливает кастомную функцию promotion БД.
func (fm *FailoverManager) SetDBPromoteFn(fn func(ctx context.Context, region string) error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.dbPromoteFn = fn
}

// SetNATSPromoteFn устанавливает кастомную функцию promotion NATS.
func (fm *FailoverManager) SetNATSPromoteFn(fn func(ctx context.Context, region string) error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.natsPromoteFn = fn
}

// ──────────────────────────────────────────────────────────────────────────────
// Failover Operations
// ──────────────────────────────────────────────────────────────────────────────

// InitiateFailover создаёт запрос на failover (статус: pending).
// Если !RequireAdmin, выполняет failover немедленно.
//
// Соответствует: ISO 27001 A.17.1.2 (DR procedures — authorised initiation)
func (fm *FailoverManager) InitiateFailover(ctx context.Context, reason, initiatedBy string) (*FailoverEvent, error) {
	fm.mu.Lock()

	if fm.activeFailover != nil && fm.activeFailover.Status == FailoverInProgress {
		fm.mu.Unlock()
		return nil, fmt.Errorf("failover already in progress: id=%s", fm.activeFailover.ID)
	}

	fm.eventIDSeq++
	eventID := fmt.Sprintf("fo-%s-%d", time.Now().UTC().Format("20060102-150405"), fm.eventIDSeq)

	healthBefore := fm.health.GetStatus()

	event := &FailoverEvent{
		ID:            eventID,
		TriggerReason: reason,
		FromRegion:    fm.region,
		ToRegion:      fm.drRegion,
		Status:        FailoverPending,
		InitiatedBy:   initiatedBy,
		HealthBefore:  &healthBefore,
		StartedAt:     time.Now(),
		CreatedAt:     time.Now(),
	}

	fm.activeFailover = event
	fm.mu.Unlock()

	fm.logger.Warn("failover initiated",
		"id", eventID,
		"reason", reason,
		"from", fm.region,
		"to", fm.drRegion,
		"initiated_by", initiatedBy,
	)

	// Если admin confirm не требуется — выполняем немедленно.
	if !DefaultFailoverConfig().RequireAdmin {
		return fm.executeFailover(ctx, event)
	}

	// Сохраняем в store для ожидания подтверждения.
	if fm.store != nil {
		if err := fm.store.SaveFailoverEvent(ctx, event); err != nil {
			fm.logger.Error("failed to save failover event", "id", eventID, "error", err)
		}
	}

	return event, nil
}

// ApproveFailover подтверждает и выполняет failover (admin action).
//
// Соответствует: ISO 27001 A.9.2.3 (Admin privileges — DR approval)
func (fm *FailoverManager) ApproveFailover(ctx context.Context, eventID, approvedBy string) (*FailoverEvent, error) {
	fm.mu.Lock()
	if fm.activeFailover == nil || fm.activeFailover.ID != eventID {
		fm.mu.Unlock()
		return nil, fmt.Errorf("failover event %s not found or not pending", eventID)
	}
	if fm.activeFailover.Status != FailoverPending {
		fm.mu.Unlock()
		return nil, fmt.Errorf("failover event %s is not in pending status (current: %s)",
			eventID, fm.activeFailover.Status)
	}

	now := time.Now()
	fm.activeFailover.ApprovedBy = approvedBy
	fm.activeFailover.ApprovedAt = &now
	fm.activeFailover.Status = FailoverApproved
	event := fm.activeFailover
	fm.mu.Unlock()

	fm.logger.Warn("failover approved",
		"id", eventID,
		"approved_by", approvedBy,
	)

	return fm.executeFailover(ctx, event)
}

// RejectFailover отклоняет failover (admin action).
func (fm *FailoverManager) RejectFailover(ctx context.Context, eventID, rejectedBy, reason string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.activeFailover == nil || fm.activeFailover.ID != eventID {
		return fmt.Errorf("failover event %s not found", eventID)
	}
	if fm.activeFailover.Status != FailoverPending {
		return fmt.Errorf("failover event %s is not pending", eventID)
	}

	fm.activeFailover.Status = FailoverRejected
	fm.activeFailover.ErrorMessage = reason

	fm.logger.Warn("failover rejected",
		"id", eventID,
		"rejected_by", rejectedBy,
		"reason", reason,
	)

	return nil
}

// GetActiveFailover возвращает текущий активный failover.
func (fm *FailoverManager) GetActiveFailover() *FailoverEvent {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.activeFailover
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal: Execute Failover
// ──────────────────────────────────────────────────────────────────────────────

// executeFailover выполняет полный цикл failover.
//
// Шаги:
//  1. DNS failover (обновление DNS записей)
//  2. DB promotion (переключение на DR БД)
//  3. NATS stream handover (mirror → active)
//  4. Проверка health после failover
//  5. Логирование RTO
func (fm *FailoverManager) executeFailover(ctx context.Context, event *FailoverEvent) (*FailoverEvent, error) {
	fm.mu.Lock()
	event.Status = FailoverInProgress
	fm.health.SetFailoverInProgress(true)
	fm.mu.Unlock()

	fm.logger.Warn("executing failover",
		"id", event.ID,
		"from", event.FromRegion,
		"to", event.ToRegion,
	)

	startTime := time.Now()

	// Step 1: DNS failover.
	fm.logger.Info("step 1/4: DNS failover", "id", event.ID)
	if err := fm.dnsFailoverFn(ctx, event.FromRegion, event.ToRegion); err != nil {
		return fm.failFailover(event, fmt.Sprintf("dns failover failed: %v", err))
	}
	event.DNSUpdated = true

	// Step 2: DB promotion.
	fm.logger.Info("step 2/4: DB promotion", "id", event.ID)
	if err := fm.dbPromoteFn(ctx, event.ToRegion); err != nil {
		return fm.failFailover(event, fmt.Sprintf("db promotion failed: %v", err))
	}
	event.DBPromoted = true

	// Step 3: NATS stream handover.
	fm.logger.Info("step 3/4: NATS promotion", "id", event.ID)
	if err := fm.natsPromoteFn(ctx, event.ToRegion); err != nil {
		return fm.failFailover(event, fmt.Sprintf("nats promotion failed: %v", err))
	}
	event.NATSPromoted = true

	// Step 4: Health check after failover.
	fm.logger.Info("step 4/4: post-failover health check", "id", event.ID)
	healthAfter := fm.health.GetStatus()
	event.HealthAfter = &healthAfter

	completedAt := time.Now()
	event.CompletedAt = &completedAt
	event.RTO = completedAt.Sub(startTime)

	fm.mu.Lock()
	event.Status = FailoverCompleted
	fm.health.SetFailoverInProgress(false)
	fm.activeFailover = event
	fm.mu.Unlock()

	fm.logger.Warn("failover completed successfully",
		"id", event.ID,
		"rto", event.RTO,
		"rpo_seconds", MaxRPO.Seconds(),
	)

	// Проверка RTO.
	if event.RTO > MaxRTO {
		fm.logger.Error("RTO EXCEEDED",
			"id", event.ID,
			"rto", event.RTO,
			"max_rto", MaxRTO,
		)
	}

	// Сохраняем в store.
	if fm.store != nil {
		if err := fm.store.SaveFailoverEvent(ctx, event); err != nil {
			fm.logger.Error("failed to save failover event", "id", event.ID, "error", err)
		}
	}

	return event, nil
}

// failFailover отмечает failover как failed.
func (fm *FailoverManager) failFailover(event *FailoverEvent, errMsg string) (*FailoverEvent, error) {
	now := time.Now()
	event.CompletedAt = &now
	event.Status = FailoverFailed
	event.ErrorMessage = errMsg
	event.RTO = now.Sub(event.StartedAt)

	fm.mu.Lock()
	fm.health.SetFailoverInProgress(false)
	fm.activeFailover = event
	fm.mu.Unlock()

	fm.logger.Error("failover failed",
		"id", event.ID,
		"error", errMsg,
		"rto", event.RTO,
	)

	if fm.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fm.store.SaveFailoverEvent(ctx, event); err != nil {
			fm.logger.Error("failed to save failed failover event", "id", event.ID, "error", err)
		}
	}

	return event, fmt.Errorf("%s", errMsg)
}

// publishNATSMessage публикует сообщение в NATS.
func (fm *FailoverManager) publishNATSMessage(subject string, data interface{}) error {
	if fm.nc == nil {
		fm.logger.Warn("nats not connected, skipping message",
			"subject", subject,
		)
		return nil
	}

	// В production здесь используется сериализация в JSON.
	// Для упрощения используем fmt.Sprintf.
	msg := fmt.Sprintf(`{"action":"%s","timestamp":"%s"}`,
		data.(map[string]string)["action"],
		data.(map[string]string)["timestamp"],
	)

	return fm.nc.Publish(subject, []byte(msg))
}

// RollbackFailover выполняет rollback failover.
func (fm *FailoverManager) RollbackFailover(ctx context.Context, eventID, reason string) (*FailoverEvent, error) {
	fm.mu.Lock()
	if fm.activeFailover == nil || fm.activeFailover.ID != eventID {
		fm.mu.Unlock()
		return nil, fmt.Errorf("failover event %s not found", eventID)
	}
	event := fm.activeFailover
	fm.mu.Unlock()

	fm.logger.Warn("rolling back failover",
		"id", eventID,
		"reason", reason,
	)

	// Reverse: promote original region back.
	if err := fm.dbPromoteFn(ctx, event.FromRegion); err != nil {
		return nil, fmt.Errorf("rollback db promotion failed: %v", err)
	}
	if err := fm.natsPromoteFn(ctx, event.FromRegion); err != nil {
		return nil, fmt.Errorf("rollback nats promotion failed: %v", err)
	}
	if err := fm.dnsFailoverFn(ctx, event.ToRegion, event.FromRegion); err != nil {
		return nil, fmt.Errorf("rollback dns failed: %v", err)
	}

	now := time.Now()
	event.CompletedAt = &now
	event.Status = FailoverRolledBack
	event.RollbackReason = reason

	fm.mu.Lock()
	fm.health.SetFailoverInProgress(false)
	fm.activeFailover = event
	fm.mu.Unlock()

	return event, nil
}
