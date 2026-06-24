// Package workforce — Workforce Management (WM-8.x).
//
// WM-8.1.1 Team entity
// WM-8.1.2 Matrix RBAC (Role × Permission × Entity)
// WM-8.2.1 ShiftConfiguration
// WM-8.2.2 User ↔ Shift assignment
// WM-8.4.1 Skills matrix
// WM-8.4.2 Certifications
package workforce

import "time"

// ═══════════════════════════════════════════════════════════════════════
// WM-8.1.1: Team
// ═══════════════════════════════════════════════════════════════════════

type Team struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name" validate:"required,max=100"`
	Description string   `json:"description,omitempty" db:"description"`
	LeadID      *string  `json:"lead_id,omitempty" db:"lead_id"` // team lead user_id
	MemberIDs   []string `json:"member_ids,omitempty" db:"member_ids"`
	SiteIDs     []string `json:"site_ids,omitempty" db:"site_ids"` // assigned sites
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// WM-8.2.1: Shift Configuration
// ═══════════════════════════════════════════════════════════════════════

type ShiftType string

const (
	ShiftDay     ShiftType = "day"     // 08:00-17:00
	ShiftEvening ShiftType = "evening" // 16:00-01:00
	ShiftNight   ShiftType = "night"   // 00:00-08:00
	ShiftCustom  ShiftType = "custom"
)

type ShiftConfiguration struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Type        ShiftType `json:"type" db:"type"`
	StartHour   int      `json:"start_hour" db:"start_hour"`   // 0-23
	EndHour     int      `json:"end_hour" db:"end_hour"`       // 0-23
	WorkDays    []int    `json:"work_days" db:"work_days"`     // 0=Sun..6=Sat
	Timezone    string   `json:"timezone" db:"timezone"`
	SiteID      string   `json:"site_id" db:"site_id"`
	MaxTeamSize int      `json:"max_team_size" db:"max_team_size"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// WM-8.2.2: User ↔ Shift Assignment
// ═══════════════════════════════════════════════════════════════════════

type UserShiftAssignment struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	ShiftID    string    `json:"shift_id" db:"shift_id"`
	TeamID     string    `json:"team_id,omitempty" db:"team_id"`
	ValidFrom  time.Time `json:"valid_from" db:"valid_from"`
	ValidUntil *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	IsPrimary  bool      `json:"is_primary" db:"is_primary"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// WM-8.4.1: Skills Matrix
// ═══════════════════════════════════════════════════════════════════════

type SkillCategory string

const (
	SkillCCTV     SkillCategory = "cctv"     // CCTV-specific skills
	SkillNetwork  SkillCategory = "network"  // Networking
	SkillElectrical SkillCategory = "electrical"
	SkillSoftware SkillCategory = "software" // NVR/VMS software
	SkillSafety   SkillCategory = "safety"   // Safety/security
)

type Skill struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name" validate:"required,max=100"`
	Category    SkillCategory `json:"category" db:"category"`
	Description string        `json:"description,omitempty" db:"description"`
}

type UserSkill struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	SkillID   string    `json:"skill_id" db:"skill_id"`
	Level     int       `json:"level" db:"level"` // 1=beginner..5=expert
	VerifiedBy *string  `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt *time.Time `json:"verified_at,omitempty" db:"verified_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// WM-8.4.2: Certifications
// ═══════════════════════════════════════════════════════════════════════

type Certification struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name" validate:"required,max=200"`
	Issuer       string    `json:"issuer" db:"issuer"` // "Hikvision", "Dahua", "OSHA"
	Category     string    `json:"category" db:"category"`
	ExpiresAfter int       `json:"expires_after_days" db:"expires_after_days"` // 0 = never
}

type UserCertification struct {
	ID             string     `json:"id" db:"id"`
	UserID         string     `json:"user_id" db:"user_id"`
	CertificationID string    `json:"certification_id" db:"certification_id"`
	ObtainedAt     time.Time  `json:"obtained_at" db:"obtained_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	VerifiedBy     *string    `json:"verified_by,omitempty" db:"verified_by"`
}

// IsExpired проверяет истекла ли сертификация.
func (uc *UserCertification) IsExpired() bool {
	if uc.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*uc.ExpiresAt)
}
