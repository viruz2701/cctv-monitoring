// Package agent — human-in-the-loop approval workflow для критичных действий self-healing.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ApprovalStatus — статус запроса на подтверждение.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ApprovalRequest — запрос на подтверждение действия.
type ApprovalRequest struct {
	ID           string
	DeviceID     string
	DeviceName   string
	Action       string
	Reason       string
	Decision     Decision
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Status       ApprovalStatus
	ApprovedBy   string
	ApprovedAt   *time.Time
	RejectReason string
}

// ApprovalResult — результат ожидания подтверждения.
type ApprovalResult struct {
	Approved bool
	By       string
	Reason   string
}

// ApprovalManager управляет workflow подтверждений.
type ApprovalManager struct {
	mu       sync.RWMutex
	requests map[string]*ApprovalRequest
	logger   *slog.Logger

	// Каналы для уведомлений о решении
	decisionCh map[string]chan ApprovalResult

	// Callbacks для внешних уведомлений
	OnTelegramNotify func(ctx context.Context, req ApprovalRequest) error
	OnMobilePush     func(ctx context.Context, req ApprovalRequest) error
}

// NewApprovalManager создаёт новый менеджер подтверждений.
func NewApprovalManager(logger *slog.Logger) *ApprovalManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &ApprovalManager{
		requests:   make(map[string]*ApprovalRequest),
		decisionCh: make(map[string]chan ApprovalResult),
		logger:     logger,
	}
}

// RequestApproval создаёт запрос на подтверждение и отправляет уведомления.
func (am *ApprovalManager) RequestApproval(ctx context.Context, deviceID, deviceName, action, reason string, decision Decision, ttl time.Duration) (*ApprovalRequest, error) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	req := &ApprovalRequest{
		ID:         fmt.Sprintf("approval_%s_%d", deviceID, time.Now().UnixNano()),
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Action:     action,
		Reason:     reason,
		Decision:   decision,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		Status:     ApprovalPending,
	}

	am.mu.Lock()
	am.requests[req.ID] = req
	am.decisionCh[req.ID] = make(chan ApprovalResult, 1)
	am.mu.Unlock()

	am.logger.Info("approval requested",
		"id", req.ID,
		"device", deviceID,
		"action", action,
		"ttl", ttl,
	)

	// Отправляем уведомления
	if am.OnTelegramNotify != nil {
		go func() {
			if err := am.OnTelegramNotify(ctx, *req); err != nil {
				am.logger.Warn("telegram notify failed", "id", req.ID, "error", err)
			}
		}()
	}
	if am.OnMobilePush != nil {
		go func() {
			if err := am.OnMobilePush(ctx, *req); err != nil {
				am.logger.Warn("mobile push failed", "id", req.ID, "error", err)
			}
		}()
	}

	return req, nil
}

// WaitApproval ожидает решения по запросу (блокирующий вызов с таймаутом).
func (am *ApprovalManager) WaitApproval(ctx context.Context, reqID string) ApprovalResult {
	am.mu.RLock()
	ch, ok := am.decisionCh[reqID]
	am.mu.RUnlock()

	if !ok {
		return ApprovalResult{Approved: false, Reason: "request not found"}
	}

	req := am.getRequest(reqID)
	if req == nil {
		return ApprovalResult{Approved: false, Reason: "request not found"}
	}

	timeout := time.Until(req.ExpiresAt)
	if timeout <= 0 {
		am.expireRequest(reqID)
		return ApprovalResult{Approved: false, Reason: "request already expired"}
	}

	select {
	case <-ctx.Done():
		return ApprovalResult{Approved: false, Reason: ctx.Err().Error()}
	case <-time.After(timeout):
		am.expireRequest(reqID)
		return ApprovalResult{Approved: false, Reason: "timeout expired"}
	case result := <-ch:
		return result
	}
}

// Approve подтверждает запрос.
func (am *ApprovalManager) Approve(reqID, approvedBy string) error {
	return am.resolveRequest(reqID, ApprovalApproved, approvedBy, "")
}

// Reject отклоняет запрос.
func (am *ApprovalManager) Reject(reqID, rejectedBy, reason string) error {
	return am.resolveRequest(reqID, ApprovalRejected, rejectedBy, reason)
}

// resolveRequest изменяет статус и уведомляет ждущий канал.
func (am *ApprovalManager) resolveRequest(reqID string, status ApprovalStatus, by, reason string) error {
	am.mu.Lock()
	req, ok := am.requests[reqID]
	if !ok {
		am.mu.Unlock()
		return fmt.Errorf("approval request %s not found", reqID)
	}
	if req.Status != ApprovalPending {
		am.mu.Unlock()
		return fmt.Errorf("approval request %s already %s", reqID, req.Status)
	}

	now := time.Now()
	req.Status = status
	req.ApprovedBy = by
	req.ApprovedAt = &now
	if status == ApprovalRejected {
		req.RejectReason = reason
	}

	ch, hasCh := am.decisionCh[reqID]
	am.mu.Unlock()

	if hasCh {
		ch <- ApprovalResult{
			Approved: status == ApprovalApproved,
			By:       by,
			Reason:   reason,
		}
		am.mu.Lock()
		delete(am.decisionCh, reqID)
		am.mu.Unlock()
	}

	am.logger.Info("approval resolved", "id", reqID, "status", status, "by", by)
	return nil
}

// expireRequest истекает запрос по таймауту.
func (am *ApprovalManager) expireRequest(reqID string) {
	am.mu.Lock()
	req, ok := am.requests[reqID]
	if !ok {
		am.mu.Unlock()
		return
	}
	if req.Status != ApprovalPending {
		am.mu.Unlock()
		return
	}
	req.Status = ApprovalExpired

	ch, hasCh := am.decisionCh[reqID]
	am.mu.Unlock()

	if hasCh {
		ch <- ApprovalResult{
			Approved: false,
			Reason:   "timeout: fallback to manual",
		}
		am.mu.Lock()
		delete(am.decisionCh, reqID)
		am.mu.Unlock()
	}
	am.logger.Info("approval expired", "id", reqID)
}

// getRequest возвращает запрос по ID.
func (am *ApprovalManager) getRequest(reqID string) *ApprovalRequest {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.requests[reqID]
}

// GetPending возвращает список ожидающих запросов.
func (am *ApprovalManager) GetPending() []ApprovalRequest {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var pending []ApprovalRequest
	for _, req := range am.requests {
		if req.Status == ApprovalPending {
			pending = append(pending, *req)
		}
	}
	return pending
}

// CleanupExpired удаляет просроченные запросы старше maxAge.
func (am *ApprovalManager) CleanupExpired(maxAge time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0
	for id, req := range am.requests {
		if req.Status != ApprovalPending && req.CreatedAt.Before(cutoff) {
			delete(am.requests, id)
			delete(am.decisionCh, id)
			count++
		}
	}
	return count
}

// ── Telegram approval message builder ──────────────────────────────

// BuildApprovalMessage формирует текст для Telegram-сообщения.
func BuildApprovalMessage(req ApprovalRequest) string {
	return fmt.Sprintf(
		"🔔 *Self-Healing Agent Approval Required*\n\n"+
			"*Device:* %s (%s)\n"+
			"*Action:* %s\n"+
			"*Reason:* %s\n"+
			"*Auto-fix decision:* %s\n"+
			"*Expires:* %s\n\n"+
			"Approve: /approve %s\n"+
			"Reject: /reject %s <reason>",
		req.DeviceName, req.DeviceID,
		req.Action,
		req.Reason,
		req.Decision.Level,
		req.ExpiresAt.Format("15:04:05"),
		req.ID,
		req.ID,
	)
}
