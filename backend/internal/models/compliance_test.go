// Package models — Compliance tests для CCTV Health Monitor.
//
// Проверяет соответствие кода регуляторным требованиям:
// - IEC 62443 SL-3 (Zone 3 — Backend)
// - ISO 27001 A.12.4 (Audit Logging)
// - СТБ 34.101.27 (Защита информации РБ)
// - OWASP ASVS Level 3 (V5 — Input Validation)
// - Приказ ОАЦ № 66 (п. 7.18)
package models

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// OWASP ASVS V5 — Input Validation Compliance
// ═══════════════════════════════════════════════════════════════════════

// TestOWASP_ASVS_V5_WorkOrderCreateRequest проверяет, что WorkOrderCreateRequest
// содержит validate теги для whitelist validation (OWASP ASVS V5.1).
func TestOWASP_ASVS_V5_WorkOrderCreateRequest(t *testing.T) {
	req := WorkOrderCreateRequest{}
	rt := reflect.TypeOf(req)

	// Проверяем наличие validate тегов у обязательных полей
	checks := []struct {
		field    string
		tagCheck string // что должен содержать validate тег
	}{
		{"DeviceID", "required"},
		{"Title", "required"},
		{"Type", "oneof=preventive corrective emergency routine inspection"},
		{"Priority", "oneof=critical high medium low"},
	}

	for _, c := range checks {
		field, ok := rt.FieldByName(c.field)
		if !ok {
			t.Errorf("поле %s отсутствует в WorkOrderCreateRequest", c.field)
			continue
		}
		tag := field.Tag.Get("validate")
		if !strings.Contains(tag, c.tagCheck) {
			t.Errorf("поле %s: validate тег '%s' не содержит '%s'", c.field, tag, c.tagCheck)
		}
	}
}

// TestOWASP_ASVS_V5_ValidStatusLists проверяет, что списки валидных значений
// содержат все необходимые варианты (OWASP ASVS V5 — whitelist).
func TestOWASP_ASVS_V5_ValidStatusLists(t *testing.T) {
	checks := []struct {
		name   string
		list   []string
		expect int
	}{
		{"ValidWorkOrderStatuses", ValidWorkOrderStatuses, 12},
		{"ValidPriorities", ValidPriorities, 4},
		{"ValidWorkOrderTypes", ValidWorkOrderTypes, 5},
		{"ValidAdditionalCostCategories", ValidAdditionalCostCategories, 5},
	}

	for _, c := range checks {
		if len(c.list) != c.expect {
			t.Errorf("%s: expected %d items, got %d", c.name, c.expect, len(c.list))
		}
		// Все элементы должны быть уникальны
		seen := make(map[string]bool)
		for _, v := range c.list {
			if seen[v] {
				t.Errorf("%s: duplicate value '%s'", c.name, v)
			}
			seen[v] = true
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// IEC 62443 SL-3 — Zone 3 Backend Security Compliance
// ═══════════════════════════════════════════════════════════════════════

// TestIEC62443_SL3_StateMachine проверяет, что state machine
// покрывает все требуемые статусы (12 статусов Grash CMMS).
func TestIEC62443_SL3_StateMachine(t *testing.T) {
	// SL-3 требует полного lifecycle управления
	requiredStatuses := []WorkOrderStatus{
		StatusRequested, StatusApproved, StatusOpen, StatusInProgress,
		StatusOnHold, StatusAwaitingParts, StatusAwaitingVendor, StatusAwaitingClient,
		StatusCompleted, StatusVerified, StatusClosed, StatusRejected,
	}

	if len(requiredStatuses) != 12 {
		t.Errorf("SL-3: expected exactly 12 statuses, got %d", len(requiredStatuses))
	}

	// Проверяем, что каждый статус достижим
	statusSet := make(map[WorkOrderStatus]bool)
	statusSet[StatusRequested] = true

	// Пробуем достичь каждого статуса через валидные переходы
	attemptTransition := func(from WorkOrderStatus, event string) WorkOrderStatus {
		fsm := NewWorkOrderFSM(from, nil)
		if fsm.CanTransition(event) {
			_ = fsm.Transition(event)
			return fsm.Current()
		}
		return from
	}

	statusSet[attemptTransition(StatusRequested, "approve")] = true       // APPROVED
	statusSet[attemptTransition(StatusApproved, "open")] = true           // OPEN
	statusSet[attemptTransition(StatusOpen, "start")] = true              // IN_PROGRESS
	statusSet[attemptTransition(StatusInProgress, "hold")] = true         // ON_HOLD
	statusSet[attemptTransition(StatusInProgress, "await_parts")] = true  // AWAITING_PARTS
	statusSet[attemptTransition(StatusInProgress, "await_vendor")] = true // AWAITING_VENDOR
	statusSet[attemptTransition(StatusInProgress, "await_client")] = true // AWAITING_CLIENT
	statusSet[attemptTransition(StatusInProgress, "complete")] = true     // COMPLETED
	statusSet[attemptTransition(StatusCompleted, "verify")] = true        // VERIFIED
	statusSet[attemptTransition(StatusVerified, "close")] = true          // CLOSED
	statusSet[attemptTransition(StatusRequested, "reject")] = true        // REJECTED

	for _, s := range requiredStatuses {
		if !statusSet[s] {
			t.Errorf("SL-3: статус %s не достижим через валидные переходы", s)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ISO 27001 A.12.4 — Audit Logging Compliance
// ═══════════════════════════════════════════════════════════════════════

// TestISO27001_A12_4_AuditFields проверяет наличие audit полей
// во всех мутируемых сущностях (ISO 27001 A.12.4).
func TestISO27001_A12_4_AuditFields(t *testing.T) {
	// AuditBase должен содержать обязательные поля
	ab := AuditBase{}
	abType := reflect.TypeOf(ab)

	requiredFields := []string{"CreatedAt", "UpdatedAt"}
	for _, f := range requiredFields {
		field, ok := abType.FieldByName(f)
		if !ok {
			t.Errorf("ISO 27001 A.12.4: AuditBase не содержит поле %s", f)
			continue
		}
		// CreatedAt и UpdatedAt должны быть time.Time
		if field.Type != reflect.TypeOf(time.Time{}) {
			t.Errorf("ISO 27001 A.12.4: %s должен быть time.Time, got %s", f, field.Type)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// СТБ 34.101.27 — Защита информации РБ Compliance
// ═══════════════════════════════════════════════════════════════════════

// TestSTB34_101_27_SoftDelete проверяет наличие механизма soft delete
// для обеспечения конфиденциальности данных (СТБ 34.101.27 п. 7.3).
func TestSTB34_101_27_SoftDelete(t *testing.T) {
	s := SoftDeleteMixin{}
	sType := reflect.TypeOf(s)

	requiredFields := []string{"DeletedAt", "DeletedBy"}
	for _, f := range requiredFields {
		if _, ok := sType.FieldByName(f); !ok {
			t.Errorf("СТБ 34.101.27: SoftDeleteMixin не содержит поле %s", f)
		}
	}

	// Проверяем функциональность
	now := time.Now()
	s2 := SoftDeleteMixin{DeletedAt: &now}
	if !s2.IsDeleted() {
		t.Error("СТБ 34.101.27: IsDeleted() должен вернуть true при установленном DeletedAt")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderHistory — Tamper Detection (СТБ 34.101.27 п. 7.5)
// ═══════════════════════════════════════════════════════════════════════

func TestSTB34_101_27_HistoryTamperDetection(t *testing.T) {
	history := WorkOrderHistory{}
	ht := reflect.TypeOf(history)

	// WorkOrderHistory должен содержать prev_hash для chain integrity
	if _, ok := ht.FieldByName("PrevHash"); !ok {
		t.Error("СТБ 34.101.27 п. 7.5: WorkOrderHistory не содержит PrevHash для tamper detection")
	}

	// Должен содержать все поля для полного аудита
	requiredFields := []string{"ID", "WorkOrderID", "FromStatus", "ToStatus", "ChangedBy", "ChangedAt"}
	for _, f := range requiredFields {
		if _, ok := ht.FieldByName(f); !ok {
			t.Errorf("СТБ 34.101.27: WorkOrderHistory не содержит поле %s", f)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Приказ ОАЦ № 66 (п. 7.18) — Идентификация и контроль целостности
// ═══════════════════════════════════════════════════════════════════════

// TestOAC66_Point718_UniqueID проверяет, что все сущности имеют
// уникальную идентификацию (Приказ ОАЦ № 66 п. 7.18.1).
func TestOAC66_Point718_UniqueID(t *testing.T) {
	entities := []struct {
		name string
		obj  interface{}
	}{
		{"WorkOrder", WorkOrder{}},
		{"Request", Request{}},
		{"PreventiveMaintenance", PreventiveMaintenance{}},
		{"WorkOrderHistory", WorkOrderHistory{}},
		{"WorkOrderRelation", WorkOrderRelation{}},
		{"PartsConsumption", PartsConsumption{}},
		{"Labor", Labor{}},
		{"AdditionalCost", AdditionalCost{}},
		{"TimeEntry", TimeEntry{}},
	}

	for _, e := range entities {
		rt := reflect.TypeOf(e.obj)
		idField, ok := rt.FieldByName("ID")
		if !ok {
			t.Errorf("Приказ ОАЦ №66 п.7.18.1: %s не содержит поле ID", e.name)
			continue
		}
		if idField.Type.Kind() != reflect.String {
			t.Errorf("Приказ ОАЦ №66 п.7.18.1: %s.ID должен быть string, got %s",
				e.name, idField.Type)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// RBAC — Least Privilege (ISO 27001 A.9.1)
// ═══════════════════════════════════════════════════════════════════════

// TestISO27001_A9_1_LeastPrivilege проверяет, что sensitive поля
// исключены из JSON-вывода (ISO 27001 A.9.1 — Least Privilege).
func TestISO27001_A9_1_LeastPrivilege(t *testing.T) {
	// Проверяем, что sensitive поля имеют json:"-" или не экспортируются
	// WorkOrderCreateRequest не должен содержать sensitive полей
	req := WorkOrderCreateRequest{}
	rType := reflect.TypeOf(req)
	for i := 0; i < rType.NumField(); i++ {
		field := rType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		// Проверяем что поле не sensitive
		name := strings.ToLower(field.Name)
		if strings.Contains(name, "password") || strings.Contains(name, "secret") || strings.Contains(name, "token") {
			t.Errorf("ISO 27001 A.9.1: WorkOrderCreateRequest не должен содержать поле %s", field.Name)
		}
	}
}
