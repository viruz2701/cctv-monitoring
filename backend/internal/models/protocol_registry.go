// Package models — domain models for CCTV Health Monitor.
//
// PROTO-07: Community Protocol Registry — публичный реестр Protocol Descriptor'ов.
//
// Compliance:
//   - OWASP ASVS V5 (Input validation — validate теги)
//   - OWASP ASVS V8 (Data protection — json:"-" для sensitive)
//   - ISO 27001 A.12.4 (Audit — created_at/updated_at)
//   - IEC 62443-3-3 SL-3 (Zone 3 — Backend)
package models

import (
	"encoding/json"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// CommunityDescriptor — публичный дескриптор протокола от community.
//
// Хранит JSON-дескриптор протокола для CCTV устройств определённого
// вендора. Аналогичен Docker Hub, но для Protocol Descriptor'ов.
// ═══════════════════════════════════════════════════════════════════════

type CommunityDescriptor struct {
	ID         string          `json:"id" db:"id" validate:"required,uuid"`
	Vendor     string          `json:"vendor" db:"vendor" validate:"required,min=1,max=200"`
	Version    string          `json:"version" db:"version" validate:"required,max=50"`
	Descriptor json.RawMessage `json:"descriptor" db:"descriptor" validate:"required"`
	AuthorID   string          `json:"author_id" db:"author_id" validate:"required,uuid"`
	Rating     float64         `json:"rating" db:"rating" validate:"min=0,max=5"`
	Downloads  int             `json:"downloads" db:"downloads" validate:"min=0"`
	Verified   bool            `json:"verified" db:"verified"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// CommunityDescriptorRating — оценка дескриптора пользователем (1-5).
// ═══════════════════════════════════════════════════════════════════════

type CommunityDescriptorRating struct {
	ID           string    `json:"id" db:"id" validate:"required,uuid"`
	DescriptorID string    `json:"descriptor_id" db:"descriptor_id" validate:"required,uuid"`
	UserID       string    `json:"user_id" db:"user_id" validate:"required,uuid"`
	Score        int       `json:"score" db:"score" validate:"required,min=1,max=5"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Request/Response DTOs
// ═══════════════════════════════════════════════════════════════════════

// PublishDescriptorRequest — запрос на публикацию дескриптора.
type PublishDescriptorRequest struct {
	Vendor     string          `json:"vendor" validate:"required,min=1,max=200"`
	Version    string          `json:"version" validate:"required,max=50"`
	Descriptor json.RawMessage `json:"descriptor" validate:"required"`
}

// RateDescriptorRequest — запрос на оценку дескриптора.
type RateDescriptorRequest struct {
	Score int `json:"score" validate:"required,min=1,max=5"`
}

// CommunityDescriptorFilter — фильтр для списка community дескрипторов.
type CommunityDescriptorFilter struct {
	Search    string  // поиск по вендору/ключевым словам
	MinRating float64 // минимальный рейтинг
	Verified  *bool   // только verified
	Page      int     // номер страницы
	PageSize  int     // размер страницы
	SortBy    string  // поле сортировки: rating, downloads, created_at
	SortDir   string  // направление: asc, desc
}

// CommunityDescriptorListResponse — ответ со списком дескрипторов.
type CommunityDescriptorListResponse struct {
	Descriptors []CommunityDescriptorSummary `json:"descriptors"`
	Total       int                          `json:"total"`
	Page        int                          `json:"page"`
	PageSize    int                          `json:"page_size"`
	TotalPages  int                          `json:"total_pages"`
}

// CommunityDescriptorSummary — краткая информация для списка.
// Не включает полный descriptor JSON (OWASP ASVS V8 — Data Protection).
type CommunityDescriptorSummary struct {
	ID        string    `json:"id"`
	Vendor    string    `json:"vendor"`
	Version   string    `json:"version"`
	Rating    float64   `json:"rating"`
	Downloads int       `json:"downloads"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
