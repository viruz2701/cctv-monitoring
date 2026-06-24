// Package models — State Machine для WorkOrder lifecycle.
//
// Compliance: IEC 62443 SL-3 (Zone 3 — Backend),
// ISO 27001 A.12.4 (Audit Logging),
// СТБ 34.101.27 (Защита информации — контроль целостности)
//
// Ref: com.grash.model.WorkOrderStatus enum (Grash CMMS — 12 статусов)
package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/looplab/fsm"
)

// ═══════════════════════════════════════════════════════════════════════
// WorkOrder State Machine (12 статусов по Grash CMMS)
// ═══════════════════════════════════════════════════════════════════════

// WorkOrderStateMachine управляет переходами между статусами WorkOrder.
// Использует looplab/fsm для валидации и выполнения переходов.
type WorkOrderStateMachine struct {
	fsm *fsm.FSM
}

// NewWorkOrderFSM создаёт новый экземпляр FSM для WorkOrder.
// Начальный статус: REQUESTED
func NewWorkOrderFSM(initialStatus WorkOrderStatus, callbacks fsm.Callbacks) *WorkOrderStateMachine {
	if initialStatus == "" {
		initialStatus = StatusRequested
	}

	events := []fsm.EventDesc{
		// REQUESTED → (approve → APPROVED, reject → REJECTED)
		{Name: "approve", Src: []string{string(StatusRequested)}, Dst: string(StatusApproved)},
		{Name: "reject", Src: []string{string(StatusRequested)}, Dst: string(StatusRejected)},

		// APPROVED → (open → OPEN, reject → REJECTED)
		{Name: "open", Src: []string{string(StatusApproved)}, Dst: string(StatusOpen)},
		{Name: "reject_from_approved", Src: []string{string(StatusApproved)}, Dst: string(StatusRejected)},

		// OPEN → (start → IN_PROGRESS, reject → REJECTED)
		{Name: "start", Src: []string{string(StatusOpen)}, Dst: string(StatusInProgress)},
		{Name: "reject_from_open", Src: []string{string(StatusOpen)}, Dst: string(StatusRejected)},

		// IN_PROGRESS → (complete → COMPLETED, hold → ON_HOLD,
		//                await_parts → AWAITING_PARTS, await_vendor → AWAITING_VENDOR,
		//                await_client → AWAITING_CLIENT)
		{Name: "complete", Src: []string{string(StatusInProgress)}, Dst: string(StatusCompleted)},
		{Name: "hold", Src: []string{string(StatusInProgress)}, Dst: string(StatusOnHold)},
		{Name: "await_parts", Src: []string{string(StatusInProgress)}, Dst: string(StatusAwaitingParts)},
		{Name: "await_vendor", Src: []string{string(StatusInProgress)}, Dst: string(StatusAwaitingVendor)},
		{Name: "await_client", Src: []string{string(StatusInProgress)}, Dst: string(StatusAwaitingClient)},

		// ON_HOLD → (resume → IN_PROGRESS)
		{Name: "resume", Src: []string{string(StatusOnHold)}, Dst: string(StatusInProgress)},

		// AWAITING_PARTS → (parts_received → IN_PROGRESS)
		{Name: "parts_received", Src: []string{string(StatusAwaitingParts)}, Dst: string(StatusInProgress)},

		// AWAITING_VENDOR → (vendor_resolved → IN_PROGRESS)
		{Name: "vendor_resolved", Src: []string{string(StatusAwaitingVendor)}, Dst: string(StatusInProgress)},

		// AWAITING_CLIENT → (client_responded → IN_PROGRESS)
		{Name: "client_responded", Src: []string{string(StatusAwaitingClient)}, Dst: string(StatusInProgress)},

		// COMPLETED → (verify → VERIFIED, reopen → IN_PROGRESS)
		{Name: "verify", Src: []string{string(StatusCompleted)}, Dst: string(StatusVerified)},
		{Name: "reopen_from_completed", Src: []string{string(StatusCompleted)}, Dst: string(StatusInProgress)},

		// VERIFIED → (close → CLOSED, reopen → IN_PROGRESS)
		{Name: "close", Src: []string{string(StatusVerified)}, Dst: string(StatusClosed)},
		{Name: "reopen_from_verified", Src: []string{string(StatusVerified)}, Dst: string(StatusInProgress)},

		// REJECTED → (reopen → REQUESTED)
		{Name: "reopen", Src: []string{string(StatusRejected)}, Dst: string(StatusRequested)},
	}

	if callbacks == nil {
		callbacks = make(fsm.Callbacks)
	}

	sm := fsm.NewFSM(
		string(initialStatus),
		events,
		callbacks,
	)

	return &WorkOrderStateMachine{fsm: sm}
}

// Current возвращает текущий статус.
func (sm *WorkOrderStateMachine) Current() WorkOrderStatus {
	return WorkOrderStatus(sm.fsm.Current())
}

// AvailableTransitions возвращает список доступных переходов из текущего статуса.
func (sm *WorkOrderStateMachine) AvailableTransitions() []string {
	return sm.fsm.AvailableTransitions()
}

// CanTransition проверяет, возможен ли переход с указанным именем события.
func (sm *WorkOrderStateMachine) CanTransition(event string) bool {
	for _, t := range sm.fsm.AvailableTransitions() {
		if t == event {
			return true
		}
	}
	return false
}

// Transition выполняет переход по имени события.
// Возвращает error если переход недопустим.
func (sm *WorkOrderStateMachine) Transition(event string, args ...interface{}) error {
	if !sm.CanTransition(event) {
		return fmt.Errorf(
			"invalid transition: cannot %s from status %s (available: %s)",
			event, sm.Current(), strings.Join(sm.AvailableTransitions(), ", "),
		)
	}
	return sm.fsm.Event(context.Background(), event, args...)
}

// ═══════════════════════════════════════════════════════════════════════
// Утилиты для работы со статусами
// ═══════════════════════════════════════════════════════════════════════

// IsTerminal возвращает true если статус является терминальным (завершающим).
func IsTerminal(status WorkOrderStatus) bool {
	return status == StatusClosed || status == StatusRejected
}

// IsActive возвращает true если статус активный (требует действий).
func IsActive(status WorkOrderStatus) bool {
	switch status {
	case StatusClosed, StatusRejected, StatusCompleted, StatusVerified:
		return false
	default:
		return true
	}
}

// IsPaused возвращает true если статус приостанавливает SLA-таймер.
func IsPaused(status WorkOrderStatus) bool {
	switch status {
	case StatusOnHold, StatusAwaitingParts, StatusAwaitingVendor, StatusAwaitingClient:
		return true
	default:
		return false
	}
}

// StatusCategory возвращает категорию статуса для группировки в UI.
func StatusCategory(status WorkOrderStatus) string {
	switch status {
	case StatusRequested:
		return "requested"
	case StatusApproved:
		return "approved"
	case StatusOpen, StatusInProgress:
		return "active"
	case StatusOnHold, StatusAwaitingParts, StatusAwaitingVendor, StatusAwaitingClient:
		return "paused"
	case StatusCompleted:
		return "completed"
	case StatusVerified:
		return "verified"
	case StatusClosed:
		return "closed"
	case StatusRejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderStatus validation (OWASP ASVS V5 — whitelist)
// ═══════════════════════════════════════════════════════════════════════

// ValidateWorkOrderStatus проверяет, что статус входит в допустимый набор.
func ValidateWorkOrderStatus(status string) bool {
	for _, valid := range ValidWorkOrderStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// ValidateTransition проверяет легальность перехода между статусами.
// Без создания экземпляра FSM — для быстрой валидации на уровне API.
func ValidateTransition(from, to WorkOrderStatus) bool {
	sm := NewWorkOrderFSM(from, nil)
	for _, event := range sm.AvailableTransitions() {
		_ = sm.Transition(event)
		if sm.Current() == to {
			return true
		}
		// reset
		sm = NewWorkOrderFSM(from, nil)
	}
	return false
}
