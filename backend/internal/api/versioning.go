// Package api — P2-API: URL-based + header-based API versioning.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-API: API Versioning Strategy
//
// Поддерживаются два механизма:
//  1. URL-based: /api/v1/..., /api/v2/...
//  2. Header-based: X-API-Version: v1|v2 (приоритетнее URL)
//
// Middleware проверяет версию на каждом запросе и добавляет:
//   - Sunset header для deprecated версий (RFC 8594)
//   - Deprecation header для deprecated версий
//   - X-API-Version response header
//
// Соответствует:
//   - IEC 62443-3-3 SL-2 (Zone 2 — DMZ): Контроль версий API
//   - ISO 27001 A.12.4.1: Audit trail для изменений версий
//   - OWASP ASVS V2.1.1: Версионирование API
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// APIVersion — семантическая версия API.
type APIVersion string

const (
	V1 APIVersion = "v1"
	V2 APIVersion = "v2"
)

// ValidVersions — множество валидных версий.
var ValidVersions = map[APIVersion]bool{
	V1: true,
	V2: true,
}

// VersionInfo — метаданные версии API.
type VersionInfo struct {
	Version      APIVersion `json:"version"`
	Deprecated   bool       `json:"deprecated"`
	Sunset       string     `json:"sunset,omitempty"`        // RFC 3339
	ReleasedAt   string     `json:"released_at"`             // RFC 3339
	DeprecatedAt string     `json:"deprecated_at,omitempty"` // RFC 3339
	Changelog    string     `json:"changelog"`
}

// ── Context Key ──────────────────────────────────────────────────────────

type versionContextKey string

const apiVersionCtxKey versionContextKey = "api_version"

// VersionFromContext извлекает APIVersion из контекста запроса.
func VersionFromContext(ctx context.Context) (APIVersion, bool) {
	v, ok := ctx.Value(apiVersionCtxKey).(APIVersion)
	return v, ok
}

// ── VersionStore ─────────────────────────────────────────────────────────

// VersionStore — интерфейс хранения метаданных версий API.
type VersionStore interface {
	ListVersions() ([]VersionInfo, error)
	GetVersion(version APIVersion) (*VersionInfo, error)
	CreateVersion(version APIVersion, changelog string) error
	UpdateVersion(version APIVersion, info VersionInfo) error
}

// ── Helpers ──────────────────────────────────────────────────────────────

// extractVersionFromPath извлекает версию API из URL-пути.
// Ожидает формат: /api/v{number}/...
func extractVersionFromPath(path string) APIVersion {
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		return ""
	}
	// Ищем часть вида "v1", "v2"
	seg := strings.ToLower(parts[1])
	if strings.HasPrefix(seg, "v") && len(seg) == 2 {
		v := APIVersion(seg)
		if ValidVersions[v] {
			return v
		}
	}
	return ""
}

// isDeprecated проверяет, объявлена ли версия устаревшей.
func isDeprecated(version APIVersion, store VersionStore) bool {
	if store == nil {
		return false
	}
	info, err := store.GetVersion(version)
	if err != nil || info == nil {
		return false
	}
	return info.Deprecated
}

// getSunsetDate возвращает дату sunset для версии.
func getSunsetDate(version APIVersion, store VersionStore) string {
	if store == nil {
		return ""
	}
	info, err := store.GetVersion(version)
	if err != nil || info == nil {
		return ""
	}
	return info.Sunset
}

// ── VersionMiddleware ────────────────────────────────────────────────────

// VersionMiddleware проверяет версию API из URL или X-API-Version header.
// Добавляет Sunset/Deprecation headers для deprecated версий.
// Устанавливает X-API-Version response header.
//
// Порядок определения версии (приоритет):
//  1. X-API-Version header
//  2. URL path (/api/v1/..., /api/v2/...)
//
// Соответствует:
//   - OWASP ASVS V2.1.1: Версионирование API
//   - IEC 62443-3-3 SR 2.1 (SL-2): Управление изменениями
func VersionMiddleware(store VersionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Определяем версию из URL
			version := extractVersionFromPath(r.URL.Path)

			// 2. Header X-API-Version имеет приоритет
			if headerVersion := r.Header.Get("X-API-Version"); headerVersion != "" {
				v := APIVersion(strings.TrimSpace(strings.ToLower(headerVersion)))
				if ValidVersions[v] {
					version = v
				}
			}

			// 3. Если версия не определена — пропускаем без изменений
			if version == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 4. Добавляем Deprecation/Sunset headers для deprecated версий
			if isDeprecated(version, store) {
				w.Header().Set("Deprecation", "true")
				if sunset := getSunsetDate(version, store); sunset != "" {
					w.Header().Set("Sunset", sunset)
				}
			}

			// 5. Устанавливаем X-API-Version в response и request context
			w.Header().Set("X-API-Version", string(version))
			ctx := context.WithValue(r.Context(), apiVersionCtxKey, version)
			r.Header.Set("X-API-Version", string(version))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ── Changelog Entry ──────────────────────────────────────────────────────

// ChangelogEntry — запись в changelog API.
type ChangelogEntry struct {
	Version  APIVersion `json:"version"`
	Date     string     `json:"date"` // RFC 3339
	Change   string     `json:"change"`
	Breaking bool       `json:"breaking"`
	JiraRef  string     `json:"jira_ref,omitempty"`
}

// DefaultChangelog возвращает changelog по умолчанию.
var DefaultChangelog = []ChangelogEntry{
	{
		Version:  V1,
		Date:     "2026-01-15T00:00:00Z",
		Change:   "Initial release",
		Breaking: false,
		JiraRef:  "P1-API",
	},
}

// defaultVersionStore — in-memory реализация VersionStore для development.
type defaultVersionStore struct {
	versions map[APIVersion]*VersionInfo
}

// NewDefaultVersionStore создаёт in-memory store с default версиями.
func NewDefaultVersionStore() VersionStore {
	return &defaultVersionStore{
		versions: map[APIVersion]*VersionInfo{
			V1: {
				Version:    V1,
				Deprecated: false,
				Sunset:     "",
				ReleasedAt: "2026-01-15T00:00:00Z",
				Changelog:  "Initial release v1",
			},
		},
	}
}

func (s *defaultVersionStore) ListVersions() ([]VersionInfo, error) {
	list := make([]VersionInfo, 0, len(s.versions))
	for _, v := range s.versions {
		list = append(list, *v)
	}
	return list, nil
}

func (s *defaultVersionStore) GetVersion(version APIVersion) (*VersionInfo, error) {
	v, ok := s.versions[version]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (s *defaultVersionStore) CreateVersion(version APIVersion, changelog string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	s.versions[version] = &VersionInfo{
		Version:    version,
		Deprecated: false,
		ReleasedAt: now,
		Changelog:  changelog,
	}
	return nil
}

func (s *defaultVersionStore) UpdateVersion(version APIVersion, info VersionInfo) error {
	s.versions[version] = &VersionInfo{
		Version:      info.Version,
		Deprecated:   info.Deprecated,
		Sunset:       info.Sunset,
		ReleasedAt:   info.ReleasedAt,
		DeprecatedAt: info.DeprecatedAt,
		Changelog:    info.Changelog,
	}
	return nil
}
