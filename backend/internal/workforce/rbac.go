// Package workforce — Matrix RBAC (WM-8.1.2).
//
// Role × Permission × Entity matrix-based access control.
//
// Роли: admin, manager, technician, viewer, support
// Разрешения: create, read, update, delete, assign, approve, export
// Сущности: work_order, device, site, spare_part, schedule, user, team, report, settings
//
// Compliance:
//   - ISO 27001 A.9.1 (Access control policy)
//   - ISO 27001 A.9.2 (User access provisioning)
//   - OWASP ASVS V2 (Authentication verification)
//   - IEC 62443 SR 2.1 (Account management)
package workforce

import "fmt"

// ═══════════════════════════════════════════════════════════════════════
// WM-8.1.2: Matrix RBAC
// ═══════════════════════════════════════════════════════════════════════

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleManager   Role = "manager"
	RoleTechnician Role = "technician"
	RoleViewer    Role = "viewer"
	RoleSupport   Role = "support"
)

type Permission string

const (
	PermCreate  Permission = "create"
	PermRead    Permission = "read"
	PermUpdate  Permission = "update"
	PermDelete  Permission = "delete"
	PermAssign  Permission = "assign"
	PermApprove Permission = "approve"
	PermExport  Permission = "export"
)

type Entity string

const (
	EntityWorkOrder Entity = "work_order"
	EntityDevice    Entity = "device"
	EntitySite      Entity = "site"
	EntitySparePart Entity = "spare_part"
	EntitySchedule  Entity = "schedule"
	EntityUser      Entity = "user"
	EntityTeam      Entity = "team"
	EntityReport    Entity = "report"
	EntitySettings  Entity = "settings"
)

// RBACMatrix — матрица Role × Entity → разрешённые Permission.
type RBACMatrix map[Role]map[Entity][]Permission

// DefaultRBACMatrix возвращает матрицу RBAC по умолчанию.
func DefaultRBACMatrix() RBACMatrix {
	return RBACMatrix{
		RoleAdmin: {
			EntityWorkOrder: {PermCreate, PermRead, PermUpdate, PermDelete, PermAssign, PermApprove, PermExport},
			EntityDevice:    {PermCreate, PermRead, PermUpdate, PermDelete, PermExport},
			EntitySite:      {PermCreate, PermRead, PermUpdate, PermDelete},
			EntitySparePart: {PermCreate, PermRead, PermUpdate, PermDelete, PermExport},
			EntitySchedule:  {PermCreate, PermRead, PermUpdate, PermDelete},
			EntityUser:      {PermCreate, PermRead, PermUpdate, PermDelete, PermAssign},
			EntityTeam:      {PermCreate, PermRead, PermUpdate, PermDelete, PermAssign},
			EntityReport:    {PermRead, PermExport},
			EntitySettings:  {PermRead, PermUpdate},
		},
		RoleManager: {
			EntityWorkOrder: {PermCreate, PermRead, PermUpdate, PermAssign, PermApprove, PermExport},
			EntityDevice:    {PermRead, PermUpdate, PermExport},
			EntitySite:      {PermRead, PermUpdate},
			EntitySparePart: {PermCreate, PermRead, PermUpdate, PermExport},
			EntitySchedule:  {PermCreate, PermRead, PermUpdate},
			EntityUser:      {PermRead},
			EntityTeam:      {PermRead, PermUpdate},
			EntityReport:    {PermRead, PermExport},
		},
		RoleTechnician: {
			EntityWorkOrder: {PermRead, PermUpdate, PermExport},
			EntityDevice:    {PermRead},
			EntitySparePart: {PermRead},
			EntitySchedule:  {PermRead},
			EntityReport:    {PermRead},
		},
		RoleViewer: {
			EntityWorkOrder: {PermRead},
			EntityDevice:    {PermRead},
			EntitySite:      {PermRead},
			EntitySparePart: {PermRead},
			EntitySchedule:  {PermRead},
			EntityReport:    {PermRead},
		},
		RoleSupport: {
			EntityWorkOrder: {PermRead, PermUpdate},
			EntityDevice:    {PermRead},
			EntityUser:      {PermRead},
			EntityReport:    {PermRead},
		},
	}
}

// CheckPermission проверяет, имеет ли роль разрешение на действие с сущностью.
func (m RBACMatrix) CheckPermission(role Role, entity Entity, perm Permission) bool {
	entities, ok := m[role]
	if !ok {
		return false
	}
	perms, ok := entities[entity]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPermissions возвращает все разрешения для роли на сущность.
func (m RBACMatrix) GetPermissions(role Role, entity Entity) []Permission {
	entities, ok := m[role]
	if !ok {
		return nil
	}
	return entities[entity]
}

// GetRoleWeight возвращает "вес" роли (для иерархии).
func GetRoleWeight(role Role) int {
	switch role {
	case RoleAdmin:
		return 100
	case RoleManager:
		return 80
	case RoleSupport:
		return 60
	case RoleTechnician:
		return 40
	case RoleViewer:
		return 20
	default:
		return 0
	}
}

// HasHigherRoleThan проверяет, имеет ли пользователь более высокую роль.
func HasHigherRoleThan(current, other Role) bool {
	return GetRoleWeight(current) > GetRoleWeight(other)
}

// ═══════════════════════════════════════════════════════════════════════
// Validators
// ═══════════════════════════════════════════════════════════════════════

func ValidateRole(r string) bool {
	switch Role(r) {
	case RoleAdmin, RoleManager, RoleTechnician, RoleViewer, RoleSupport:
		return true
	}
	return false
}

func ValidateEntity(e string) bool {
	switch Entity(e) {
	case EntityWorkOrder, EntityDevice, EntitySite, EntitySparePart,
		EntitySchedule, EntityUser, EntityTeam, EntityReport, EntitySettings:
		return true
	}
	return false
}

func ValidatePermission(p string) bool {
	switch Permission(p) {
	case PermCreate, PermRead, PermUpdate, PermDelete,
		PermAssign, PermApprove, PermExport:
		return true
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════
// Workload Analytics (WM-8.3.1)
// ═══════════════════════════════════════════════════════════════════════

type TechnicianWorkload struct {
	UserID          string  `json:"user_id"`
	UserName        string  `json:"user_name"`
	TeamID          string  `json:"team_id,omitempty"`
	TeamName        string  `json:"team_name,omitempty"`
	Role            Role    `json:"role"`

	// Current load
	ActiveWO        int     `json:"active_wo"`
	PendingWO       int     `json:"pending_wo"`
	MaxWorkload     int     `json:"max_workload"` // default 5
	UtilizationPct  float64 `json:"utilization_percent"`

	// Today
	CompletedToday  int     `json:"completed_today"`
	OnTimeToday     int     `json:"on_time_today"`
	TotalTimeMin    int     `json:"total_time_minutes"`

	// Skills & certifications
	Skills          []string `json:"skills,omitempty"`
	Certifications  []string `json:"certifications,omitempty"`
	ExpiringCerts   int      `json:"expiring_certs_count"` // certs expiring in 30 days

	// Location
	BaseLocation    string   `json:"base_location,omitempty"`
	AssignedSites   []string `json:"assigned_sites,omitempty"`
}

// WorkloadSummary — сводка по нагрузке команды.
type WorkloadSummary struct {
	TotalTechnicians  int     `json:"total_technicians"`
	ActiveTechnicians int     `json:"active_technicians"`
	AvailableTechs    int     `json:"available_technicians"`
	TotalActiveWO     int     `json:"total_active_wo"`
	AvgUtilization    float64 `json:"avg_utilization_percent"`
	OverloadedTechs   int     `json:"overloaded_technicians"`
	OverdueWO         int     `json:"overdue_wo"`
}

// IsAvailable проверяет, может ли техник взять новый наряд.
func (tw *TechnicianWorkload) IsAvailable() bool {
	return tw.ActiveWO < tw.MaxWorkload
}

// Utilization возвращает процент загрузки.
func (tw *TechnicianWorkload) Utilization() float64 {
	if tw.MaxWorkload <= 0 {
		return 0
	}
	return float64(tw.ActiveWO) / float64(tw.MaxWorkload) * 100
}

// String возвращает строковое представление workload.
func (tw *TechnicianWorkload) String() string {
	return fmt.Sprintf("%s (%s): %d/%d active, %d completed today",
		tw.UserName, tw.Role, tw.ActiveWO, tw.MaxWorkload, tw.CompletedToday)
}
