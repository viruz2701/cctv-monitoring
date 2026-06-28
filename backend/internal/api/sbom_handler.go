// Package api — SBOM (Software Bill of Materials) endpoint.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-N1: Supply Chain Security (SBOM + SSDF)
//
// Соответствует:
//   - EU CRA (Dec 2027) — Software Bill of Materials
//   - US EO 14028 — Improving the Nation's Cybersecurity
//   - ISO 27001 A.15.1 — Supply Chain Security
//   - IEC 62443-4-1 — Secure Development Lifecycle (SD-6)
//
// Форматы:
//   - CycloneDX (JSON) — основной формат
//   - SPDX (JSON) — альтернативный формат
//
// Эндпоинт публичный (без JWT), только GET.
// SBOM генерируется в CI/CD и встраивается в бинарник при сборке.
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
)

// SBOMFormat определяет формат SBOM.
type SBOMFormat string

const (
	// SBOMFormatCycloneDX — CycloneDX JSON (основной)
	SBOMFormatCycloneDX SBOMFormat = "cyclonedx"
	// SBOMFormatSPDX — SPDX JSON
	SBOMFormatSPDX SBOMFormat = "spdx"
)

// SBOMProvider управляет загрузкой и кэшированием SBOM.
type SBOMProvider struct {
	mu        sync.RWMutex
	sbomDir   string
	cache     map[SBOMFormat]sbomCacheEntry
	cacheTTL  time.Duration
	buildTime string
	version   string
}

type sbomCacheEntry struct {
	data []byte
	time time.Time
}

// SBOMVersionInfo содержит метаинформацию о версии SBOM.
type SBOMVersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Format    string `json:"format"`
}

// SBOMResponse — ответ SBOM endpoint.
type SBOMResponse struct {
	Format      SBOMFormat      `json:"format"`
	SpecVersion string          `json:"spec_version"`
	Data        json.RawMessage `json:"data"`
	VersionInfo SBOMVersionInfo `json:"version_info"`
}

// SBOMListResponse — список доступных SBOM.
type SBOMListResponse struct {
	Available []SBOMFormat `json:"available"`
	Default   SBOMFormat   `json:"default"`
}

// NewSBOMProvider создаёт новый SBOMProvider.
//
// sbomDir — директория со SBOM файлами (генерируются в CI).
// buildTime — время сборки (встраивается при build).
// version — версия приложения.
func NewSBOMProvider(sbomDir, buildTime, version string) *SBOMProvider {
	return &SBOMProvider{
		sbomDir:   sbomDir,
		cache:     make(map[SBOMFormat]sbomCacheEntry),
		cacheTTL:  5 * time.Minute,
		buildTime: buildTime,
		version:   version,
	}
}

// loadSBOM загружает SBOM из файла с кэшированием.
func (p *SBOMProvider) loadSBOM(format SBOMFormat) ([]byte, error) {
	p.mu.RLock()
	entry, ok := p.cache[format]
	p.mu.RUnlock()

	if ok && time.Since(entry.time) < p.cacheTTL {
		return entry.data, nil
	}

	var filename string
	switch format {
	case SBOMFormatCycloneDX:
		filename = "sbom.cyclonedx.json"
	case SBOMFormatSPDX:
		filename = "sbom.spdx.json"
	default:
		return nil, os.ErrNotExist
	}

	fpath := filepath.Join(p.sbomDir, filename)
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cache[format] = sbomCacheEntry{data: data, time: time.Now()}
	p.mu.Unlock()

	return data, nil
}

// HandleListSBOM возвращает список доступных форматов SBOM.
// Endpoint: GET /api/v1/sbom
func (p *SBOMProvider) HandleListSBOM(w http.ResponseWriter, r *http.Request) {
	formats := []SBOMFormat{SBOMFormatCycloneDX}
	if _, err := os.Stat(filepath.Join(p.sbomDir, "sbom.spdx.json")); err == nil {
		formats = append(formats, SBOMFormatSPDX)
	}

	jsonResponse(w, http.StatusOK, SBOMListResponse{
		Available: formats,
		Default:   SBOMFormatCycloneDX,
	})
}

// HandleGetSBOM возвращает SBOM в указанном формате.
// Endpoint: GET /api/v1/sbom/{format}
//
// Формат указывается в URL: /api/v1/sbom/cyclonedx или /api/v1/sbom/spdx.
// По умолчанию — CycloneDX (редирект с /api/v1/sbom).
func (p *SBOMProvider) HandleGetSBOM(w http.ResponseWriter, r *http.Request) {
	formatStr := chi.URLParam(r, "format")
	if formatStr == "" {
		formatStr = string(SBOMFormatCycloneDX)
	}

	format := SBOMFormat(formatStr)
	if format != SBOMFormatCycloneDX && format != SBOMFormatSPDX {
		RespondError(w, r, NewBadRequestError(
			"unsupported SBOM format: "+formatStr+". Supported: cyclonedx, spdx"))
		return
	}

	data, err := p.loadSBOM(format)
	if err != nil {
		RespondError(w, r, NewNotFoundError("SBOM not found for format: "+formatStr))
		return
	}

	// Парсим JSON для проверки валидности
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		slog.Error("invalid SBOM data", "format", format, "error", err)
		RespondError(w, r, NewInternalError("invalid SBOM data", err))
		return
	}

	resp := SBOMResponse{
		Format:      format,
		SpecVersion: p.specVersion(format),
		Data:        raw,
		VersionInfo: SBOMVersionInfo{
			Version:   p.version,
			BuildTime: p.buildTime,
			Format:    string(format),
		},
	}

	jsonResponse(w, http.StatusOK, resp)
}

// HandleGetSBOMRaw возвращает "сырой" SBOM файл (для инструментов).
// Endpoint: GET /api/v1/sbom/{format}/raw
//
// Используется сторонними инструментами (grype, trivy, dependabot),
// которые ожидают прямой SBOM в стандартном формате.
func (p *SBOMProvider) HandleGetSBOMRaw(w http.ResponseWriter, r *http.Request) {
	formatStr := chi.URLParam(r, "format")
	if formatStr == "" {
		formatStr = string(SBOMFormatCycloneDX)
	}

	format := SBOMFormat(formatStr)
	if format != SBOMFormatCycloneDX && format != SBOMFormatSPDX {
		RespondError(w, r, NewBadRequestError("unsupported SBOM format: "+formatStr))
		return
	}

	data, err := p.loadSBOM(format)
	if err != nil {
		RespondError(w, r, NewNotFoundError("SBOM not found"))
		return
	}

	var contentType string
	switch format {
	case SBOMFormatCycloneDX:
		contentType = "application/vnd.cyclonedx+json"
	case SBOMFormatSPDX:
		contentType = "application/spdx+json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-SBOM-Version", p.version)
	w.Header().Set("X-SBOM-Format", string(format))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (p *SBOMProvider) specVersion(format SBOMFormat) string {
	switch format {
	case SBOMFormatCycloneDX:
		return "1.6"
	case SBOMFormatSPDX:
		return "2.3"
	default:
		return ""
	}
}

// mountSBOMRoutes монтирует SBOM маршруты на роутер.
//
// Эндпоинты публичные (без JWT), только GET.
// Соответствует: EU CRA, US EO 14028, ISO 27001 A.15.1.
func (s *Server) mountSBOMRoutes(r chi.Router) {
	if s.sbomProvider == nil {
		return
	}

	r.Get("/api/v1/sbom", s.sbomProvider.HandleListSBOM)
	r.Get("/api/v1/sbom/{format}", s.sbomProvider.HandleGetSBOM)
	r.Get("/api/v1/sbom/{format}/raw", s.sbomProvider.HandleGetSBOMRaw)
}
