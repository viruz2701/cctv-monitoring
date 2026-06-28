// Package playbook — Playbook Registry с версионированием.
//
// Предоставляет реестр плейбуков с поддержкой:
//   - Версионирование (semver)
//   - Hot reload без restart
//   - Rollback к предыдущей версии
//   - История версий
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — hot reload)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail версий)
//   - ISO 27001 A.12.6.1 (Capacity management — version rollback)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
package playbook

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"
)

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// PlaybookStep — один шаг плейбука.
type PlaybookStep struct {
	Name       string            `yaml:"name"`
	Action     string            `yaml:"action"`
	Target     string            `yaml:"target"`
	Timeout    string            `yaml:"timeout"`
	Retries    int               `yaml:"retries"`
	RetryDelay string            `yaml:"retry_delay"`
	OnFailure  string            `yaml:"on_failure"`
	Params     map[string]string `yaml:"params"`
}

// PlaybookRule — условие применимости плейбука.
type PlaybookRule struct {
	VendorType  string `yaml:"vendor_type"`
	AlarmMethod int    `yaml:"alarm_method"`
	DeviceType  string `yaml:"device_type"`
	MinPriority int    `yaml:"min_priority"`
}

// PlaybookSchema — схема плейбука с версионированием.
type PlaybookSchema struct {
	// Метаданные
	Name        string   `yaml:"name" validate:"required"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version" validate:"required,semver"` // semver (напр. "1.2.3")
	Tags        []string `yaml:"tags,omitempty"`
	Author      string   `yaml:"author,omitempty"`

	// Совместимость
	MinAgentVersion string         `yaml:"min_agent_version,omitempty"`
	Applicable      []PlaybookRule `yaml:"applicable"`

	// Шаги
	Steps      []PlaybookStep `yaml:"steps" validate:"required,min=1"`
	MaxRetries int            `yaml:"max_retries"`
	Cooldown   string         `yaml:"cooldown"`

	// Deprecated — отмечает устаревшие плейбуки
	Deprecated bool   `yaml:"deprecated,omitempty"`
	ReplacedBy string `yaml:"replaced_by,omitempty"`
}

// VersionEntry — запись в истории версий.
type VersionEntry struct {
	Version    string    `json:"version"`
	FilePath   string    `json:"file_path"`
	SHA256     string    `json:"sha256"`
	LoadedAt   time.Time `json:"loaded_at"`
	Active     bool      `json:"active"`
	RollbackOf string    `json:"rollback_of,omitempty"` // если это rollback — указывает исходную версию
}

// VersionTag — тег для версии плейбука.
type VersionTag struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// VersionDiff — структурированный diff между двумя версиями.
type VersionDiff struct {
	Name            string            `json:"name"`
	From            string            `json:"from"`
	To              string            `json:"to"`
	StepsAdded      []PlaybookStep    `json:"steps_added,omitempty"`
	StepsRemoved    []PlaybookStep    `json:"steps_removed,omitempty"`
	StepsChanged    []StepChange      `json:"steps_changed,omitempty"`
	MetadataChanged map[string]Change `json:"metadata_changed,omitempty"`
}

// StepChange — изменение одного шага.
type StepChange struct {
	Name   string `json:"name"`
	Field  string `json:"field"`
	OldVal string `json:"old_val"`
	NewVal string `json:"new_val"`
}

// Change — универсальное представление изменения.
type Change struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// RegistryConfig — конфигурация PlaybookRegistry.
type RegistryConfig struct {
	// WatchInterval — интервал проверки изменений для hot reload.
	// 0 = hot reload отключён.
	WatchInterval time.Duration `json:"watch_interval"`

	// MaxVersionHistory — макс. количество хранимых версий в истории.
	// 0 = без ограничения.
	MaxVersionHistory int `json:"max_version_history"`

	// StrictMode — если true, загрузка плейбука с ошибкой блокирует весь реестр.
	StrictMode bool `json:"strict_mode"`

	// Logger — опциональный логгер.
	Logger *slog.Logger
}

// DefaultRegistryConfig — значения по умолчанию.
var DefaultRegistryConfig = RegistryConfig{
	WatchInterval:     30 * time.Second,
	MaxVersionHistory: 20,
	StrictMode:        false,
}

func (c *RegistryConfig) validate() {
	if c.WatchInterval <= 0 {
		c.WatchInterval = DefaultRegistryConfig.WatchInterval
	}
	// 0 = без ограничения (unlimited), отрицательные = default
	if c.MaxVersionHistory < 0 {
		c.MaxVersionHistory = DefaultRegistryConfig.MaxVersionHistory
	}
}

// ═══════════════════════════════════════════════════════════════════════
// PlaybookRegistry
// ═══════════════════════════════════════════════════════════════════════

// PlaybookRegistry — реестр плейбуков с версионированием.
//
// Поддерживает:
//   - Версионирование: каждая загрузка сохраняется в истории
//   - Hot reload: WatchDir следит за изменениями файлов
//   - Rollback: восстановление предыдущей активной версии
//   - Graceful: ошибки в одном файле не блокируют остальные
type PlaybookRegistry struct {
	mu       sync.RWMutex
	schemas  map[string]*PlaybookSchema // name → текущий (последний загруженный)
	versions map[string][]VersionEntry  // name → история версий
	tags     map[string][]VersionTag    // name → список тегов
	files    map[string]string          // filePath → name (для hot reload)
	dir      string                     // директория с плейбуками
	cfg      RegistryConfig
	logger   *slog.Logger
	stopCh   chan struct{}

	// activeTimeline — хронологический порядок уникальных версий.
	// Используется для RollbackLatest: каждый вызов LoadFile с новой
	// версией добавляет её в timeline; Rollback не модифицирует timeline.
	// RollbackLatest откатывается к предыдущей версии в timeline и
	// удаляет текущую.
	activeTimeline map[string][]string // name → [v1, v2, v3, ...]
}

// NewPlaybookRegistry создаёт новый PlaybookRegistry.
func NewPlaybookRegistry(cfg RegistryConfig) *PlaybookRegistry {
	cfg.validate()
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &PlaybookRegistry{
		schemas:        make(map[string]*PlaybookSchema),
		versions:       make(map[string][]VersionEntry),
		tags:           make(map[string][]VersionTag),
		files:          make(map[string]string),
		cfg:            cfg,
		logger:         cfg.Logger.With("component", "playbook-registry"),
		stopCh:         make(chan struct{}),
		activeTimeline: make(map[string][]string),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Semver Validation
// ═══════════════════════════════════════════════════════════════════════

// semverPattern — regex для валидации semver (MAJOR.MINOR.PATCH с опциональными pre-release/build).
var semverPattern = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[\da-zA-Z-]+(\.[\da-zA-Z-]+)*)?(\+[\da-zA-Z-]+(\.[\da-zA-Z-]+)*)?$`)

// isValidSemver проверяет, что строка является корректным semver.
func isValidSemver(version string) bool {
	return semverPattern.MatchString(version)
}

// ═══════════════════════════════════════════════════════════════════════
// Load Operations
// ═══════════════════════════════════════════════════════════════════════

// LoadDir загружает все *.yml / *.yaml файлы из директории.
//
// Goroutine-safe: каждая загрузка атомарна.
// Graceful: ошибки в одном файле не блокируют загрузку остальных.
func (r *PlaybookRegistry) LoadDir(dir string) error {
	r.dir = dir

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("playbook: read dir %s: %w", dir, err)
	}

	var loadErrors int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if !isYAMLFile(entry.Name()) {
			continue
		}
		if err := r.LoadFile(path); err != nil {
			loadErrors++
			r.logger.Warn("playbook load error", "file", path, "error", err)
			if r.cfg.StrictMode {
				return fmt.Errorf("playbook: strict mode: %s: %w", path, err)
			}
		}
	}

	if loadErrors > 0 {
		r.logger.Warn("playbook load complete with errors",
			"total", len(entries),
			"errors", loadErrors,
			"strict_mode", r.cfg.StrictMode,
		)
	}

	return nil
}

// LoadFile загружает один playbook-файл и записывает версию в timeline.
//
// Если плейбук с таким именем уже существует — сохраняет предыдущую
// версию в истории и активирует новую.
func (r *PlaybookRegistry) LoadFile(path string) error {
	return r.loadFile(path, true)
}

// loadFile — внутренняя загрузка с контролем записи в activeTimeline.
// Параметр recordTimeline=true используется для обычных загрузок (новые версии),
// false — для rollback (чтобы не дублировать timeline).
func (r *PlaybookRegistry) loadFile(path string, recordTimeline bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var schema PlaybookSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// Валидация
	if schema.Name == "" {
		return fmt.Errorf("%s: playbook name is required", path)
	}
	if schema.Version == "" {
		return fmt.Errorf("%s: playbook %q version is required", path, schema.Name)
	}
	if !isValidSemver(schema.Version) {
		return fmt.Errorf("%s: playbook %q version %q is not valid semver", path, schema.Name, schema.Version)
	}
	if len(schema.Steps) == 0 {
		return fmt.Errorf("%s: playbook %q must have at least one step", path, schema.Name)
	}

	// Вычисляем SHA256 хеш файла для audit trail
	hash := sha256.Sum256(data)
	sha256Hex := hex.EncodeToString(hash[:])

	r.mu.Lock()
	defer r.mu.Unlock()

	// Сохраняем предыдущую версию в историю (если есть)
	if existing, ok := r.schemas[schema.Name]; ok {
		r.addVersionLocked(existing.Name, existing.Version, path, false)
	}

	// Активируем новую версию с SHA256
	r.schemas[schema.Name] = &schema
	r.files[path] = schema.Name
	r.addVersionLockedWithSHA(schema.Name, schema.Version, path, sha256Hex, true)

	// Добавляем в activeTimeline (только для обычных загрузок, не для rollback)
	if recordTimeline {
		r.recordActiveTimeline(schema.Name, schema.Version)
	}

	r.logger.Info("playbook loaded/updated",
		"name", schema.Name,
		"version", schema.Version,
		"file", path,
		"sha256", sha256Hex[:12], // префикс для логов
		"steps", len(schema.Steps),
		"deprecated", schema.Deprecated,
	)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Hot Reload
// ═══════════════════════════════════════════════════════════════════════

// WatchDir запускает горутину для hot reload.
// Проверяет изменения файлов с интервалом WatchInterval.
// Возвращает канал ошибок (буферизованный, размер 10).
func (r *PlaybookRegistry) WatchDir() <-chan error {
	errCh := make(chan error, 10)

	if r.cfg.WatchInterval <= 0 {
		close(errCh)
		return errCh
	}

	if r.dir == "" {
		errCh <- fmt.Errorf("playbook: WatchDir called before LoadDir")
		return errCh
	}

	go r.watchLoop(errCh)
	return errCh
}

// StopWatch останавливает горутину hot reload.
func (r *PlaybookRegistry) StopWatch() {
	close(r.stopCh)
}

func (r *PlaybookRegistry) watchLoop(errCh chan<- error) {
	ticker := time.NewTicker(r.cfg.WatchInterval)
	defer ticker.Stop()

	// Собираем Last modified для всех известных файлов
	lastMod := r.collectModTimes()

	for {
		select {
		case <-r.stopCh:
			close(errCh)
			return
		case <-ticker.C:
			r.checkForChanges(lastMod, errCh)
		}
	}
}

func (r *PlaybookRegistry) collectModTimes() map[string]time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modTimes := make(map[string]time.Time, len(r.files))
	for path := range r.files {
		info, err := os.Stat(path)
		if err == nil {
			modTimes[path] = info.ModTime()
		}
	}
	return modTimes
}

func (r *PlaybookRegistry) checkForChanges(lastMod map[string]time.Time, errCh chan<- error) {
	r.mu.RLock()
	knownFiles := make([]string, 0, len(r.files))
	for path := range r.files {
		knownFiles = append(knownFiles, path)
	}
	r.mu.RUnlock()

	for _, path := range knownFiles {
		info, err := os.Stat(path)
		if err != nil {
			r.logger.Warn("playbook file stat error", "file", path, "error", err)
			continue
		}

		lastTime, ok := lastMod[path]
		if ok && !info.ModTime().After(lastTime) {
			continue
		}

		r.logger.Info("playbook file changed, reloading", "file", path)
		lastMod[path] = info.ModTime()

		if err := r.LoadFile(path); err != nil {
			r.logger.Error("playbook hot reload failed",
				"file", path, "error", err,
			)
			errCh <- fmt.Errorf("hot reload %s: %w", path, err)
		}
	}

	// Проверяем новые файлы в директории
	if r.dir == "" {
		return
	}
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}
		path := filepath.Join(r.dir, entry.Name())
		if _, known := lastMod[path]; !known {
			info, err := entry.Info()
			if err != nil {
				r.logger.Warn("cannot get file info for new playbook", "file", path, "error", err)
				continue
			}
			r.logger.Info("new playbook file detected", "file", path)
			lastMod[path] = info.ModTime()
			if err := r.LoadFile(path); err != nil {
				r.logger.Error("playbook hot reload new file failed",
					"file", path, "error", err,
				)
				errCh <- fmt.Errorf("hot reload new %s: %w", path, err)
			}
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rollback
// ═══════════════════════════════════════════════════════════════════════

// Rollback восстанавливает указанную версию плейбука из истории.
//
// Загружает файл из VersionEntry.FilePath через LoadFile, который:
//   - Сохраняет текущую активную версию в историю (как неактивную)
//   - Активирует версию из файла
//   - Добавляет запись в историю с Active=true
//
// LoadFile сам управляет блокировкой, поэтому Rollback не удерживает
// mutex на время вызова.
//
// Возвращает:
//   - true + nil — rollback successful
//   - false + nil — версия не найдена (не ошибка)
//   - false + error — ошибка при rollback
func (r *PlaybookRegistry) Rollback(name, version string) (bool, error) {
	r.mu.RLock()
	history, ok := r.versions[name]
	if !ok {
		r.mu.RUnlock()
		return false, nil
	}

	// Ищем указанную версию
	var target *VersionEntry
	for i := range history {
		if history[i].Version == version {
			target = &history[i]
			break
		}
	}
	if target == nil {
		r.mu.RUnlock()
		return false, nil
	}

	// Проверяем, что целевая версия не совпадает с текущей активной
	currentVersion := ""
	if current, ok := r.schemas[name]; ok {
		currentVersion = current.Version
	}
	r.mu.RUnlock()

	if currentVersion == version {
		r.logger.Debug("rollback skipped: already at target version",
			"name", name,
			"version", version,
		)
		return true, nil
	}

	if target.FilePath == "" {
		return false, fmt.Errorf("rollback %s@%s: no file path in version entry", name, version)
	}

	// loadFile без записи в timeline — rollback не создаёт новую запись в
	// хронологической последовательности, только восстанавливает старую.
	err := r.loadFile(target.FilePath, false)
	if err != nil {
		return false, fmt.Errorf("rollback %s@%s: %w", name, version, err)
	}

	// Помечаем в истории: только ПОСЛЕДНЯЯ запись целевой версии — активна.
	// Старые записи той же версии становятся неактивными (предотвращает
	// дублирование Active), но Active-флаги других версий сохраняются,
	// чтобы RollbackLatest мог корректно найти предыдущую активную версию.
	r.mu.Lock()
	updatedHistory := r.versions[name]
	// Сбрасываем Active только для записей целевой версии
	for i := range updatedHistory {
		if updatedHistory[i].Version == version {
			updatedHistory[i].Active = false
		}
	}
	// Активируем последнее вхождение целевой версии
	for i := len(updatedHistory) - 1; i >= 0; i-- {
		if updatedHistory[i].Version == version {
			updatedHistory[i].Active = true
			break
		}
	}
	r.versions[name] = updatedHistory
	r.mu.Unlock()

	r.logger.Warn("playbook rollback performed",
		"name", name,
		"version", version,
	)

	return true, nil
}

// RollbackLatest откатывает последнее изменение плейбука.
//
// Использует activeTimeline для нахождения предыдущей версии в
// хронологической последовательности. При каждом RollbackLatest
// текущая версия удаляется из timeline, и реестр откатывается к
// предыдущей.
//
// Возвращает false без ошибки, если timeline содержит < 2 записей.
func (r *PlaybookRegistry) RollbackLatest(name string) (bool, error) {
	r.mu.Lock()

	timeline, ok := r.activeTimeline[name]
	if !ok || len(timeline) < 2 {
		r.mu.Unlock()
		return false, nil
	}

	// Предыдущая версия — предпоследняя в timeline
	prevVersion := timeline[len(timeline)-2]

	// Удаляем текущую версию из timeline
	r.activeTimeline[name] = timeline[:len(timeline)-1]
	r.mu.Unlock()

	return r.Rollback(name, prevVersion)
}

// ═══════════════════════════════════════════════════════════════════════
// Versioning Enhancements
// ═══════════════════════════════════════════════════════════════════════

// getVersionEntry возвращает VersionEntry для указанной версии (без блокировки).
func (r *PlaybookRegistry) getVersionEntry(name, version string) *VersionEntry {
	history, ok := r.versions[name]
	if !ok {
		return nil
	}
	for i := range history {
		if history[i].Version == version {
			return &history[i]
		}
	}
	return nil
}

// getSchemaByName возвращает схему по имени из history (без блокировки).
func (r *PlaybookRegistry) getSchemaFromHistory(name, version string) *PlaybookSchema {
	entry := r.getVersionEntry(name, version)
	if entry == nil || entry.FilePath == "" {
		return nil
	}
	data, err := os.ReadFile(entry.FilePath)
	if err != nil {
		return nil
	}
	var schema PlaybookSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil
	}
	return &schema
}

// stepKey возвращает уникальный ключ шага (name:action:target).
func stepKey(s PlaybookStep) string {
	return fmt.Sprintf("%s:%s:%s", s.Name, s.Action, s.Target)
}

// DiffVersions возвращает структурированный diff между двумя версиями плейбука.
//
// Сравнивает:
//   - Добавленные/удалённые шаги
//   - Изменённые поля шагов
//   - Изменения метаданных (description, min_agent_version, max_retries, cooldown)
func (r *PlaybookRegistry) DiffVersions(name, v1, v2 string) (*VersionDiff, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema1 := r.getSchemaFromHistory(name, v1)
	schema2 := r.getSchemaFromHistory(name, v2)

	if schema1 == nil && schema2 == nil {
		return nil, fmt.Errorf("playbook %q: neither version %q nor %q found", name, v1, v2)
	}

	diff := &VersionDiff{
		Name:         name,
		From:         v1,
		To:           v2,
		StepsAdded:   nil,
		StepsRemoved: nil,
		StepsChanged: nil,
	}

	// Собираем шаги по ключам
	steps1 := make(map[string]PlaybookStep)
	if schema1 != nil {
		for _, s := range schema1.Steps {
			steps1[stepKey(s)] = s
		}
		diff.From = schema1.Version
	}

	steps2 := make(map[string]PlaybookStep)
	if schema2 != nil {
		for _, s := range schema2.Steps {
			steps2[stepKey(s)] = s
		}
		diff.To = schema2.Version
	}

	// Ищем добавленные и изменённые шаги
	for key, s2 := range steps2 {
		if s1, exists := steps1[key]; exists {
			// Проверяем изменения полей
			diff.StepsChanged = append(diff.StepsChanged, diffSteps(s1, s2)...)
		} else {
			diff.StepsAdded = append(diff.StepsAdded, s2)
		}
	}

	// Ищем удалённые шаги
	for key, s1 := range steps1 {
		if _, exists := steps2[key]; !exists {
			diff.StepsRemoved = append(diff.StepsRemoved, s1)
		}
	}

	// Сравниваем метаданные
	diff.MetadataChanged = make(map[string]Change)
	if schema1 != nil && schema2 != nil {
		if schema1.Description != schema2.Description {
			diff.MetadataChanged["description"] = Change{Old: schema1.Description, New: schema2.Description}
		}
		if schema1.MinAgentVersion != schema2.MinAgentVersion {
			diff.MetadataChanged["min_agent_version"] = Change{Old: schema1.MinAgentVersion, New: schema2.MinAgentVersion}
		}
		if schema1.MaxRetries != schema2.MaxRetries {
			diff.MetadataChanged["max_retries"] = Change{Old: schema1.MaxRetries, New: schema2.MaxRetries}
		}
		if schema1.Cooldown != schema2.Cooldown {
			diff.MetadataChanged["cooldown"] = Change{Old: schema1.Cooldown, New: schema2.Cooldown}
		}
		if schema1.Deprecated != schema2.Deprecated {
			diff.MetadataChanged["deprecated"] = Change{Old: schema1.Deprecated, New: schema2.Deprecated}
		}
	}

	return diff, nil
}

// diffSteps сравнивает два шага и возвращает список изменений полей.
func diffSteps(s1, s2 PlaybookStep) []StepChange {
	var changes []StepChange

	if s1.Action != s2.Action {
		changes = append(changes, StepChange{Name: s1.Name, Field: "action", OldVal: s1.Action, NewVal: s2.Action})
	}
	if s1.Target != s2.Target {
		changes = append(changes, StepChange{Name: s1.Name, Field: "target", OldVal: s1.Target, NewVal: s2.Target})
	}
	if s1.Timeout != s2.Timeout {
		changes = append(changes, StepChange{Name: s1.Name, Field: "timeout", OldVal: s1.Timeout, NewVal: s2.Timeout})
	}
	if s1.Retries != s2.Retries {
		changes = append(changes, StepChange{Name: s1.Name, Field: "retries", OldVal: fmt.Sprintf("%d", s1.Retries), NewVal: fmt.Sprintf("%d", s2.Retries)})
	}
	if s1.RetryDelay != s2.RetryDelay {
		changes = append(changes, StepChange{Name: s1.Name, Field: "retry_delay", OldVal: s1.RetryDelay, NewVal: s2.RetryDelay})
	}
	if s1.OnFailure != s2.OnFailure {
		changes = append(changes, StepChange{Name: s1.Name, Field: "on_failure", OldVal: s1.OnFailure, NewVal: s2.OnFailure})
	}

	return changes
}

// TagVersion добавляет тег к указанной версии плейбука.
//
// Теги позволяют помечать версии как "stable", "production", "testing" и т.д.
// Если тег уже существует для этого плейбука — он перезаписывается.
func (r *PlaybookRegistry) TagVersion(name, version, tag string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, что версия существует
	if _, ok := r.schemas[name]; !ok {
		return fmt.Errorf("playbook %q not found", name)
	}

	entry := r.getVersionEntry(name, version)
	if entry == nil {
		return fmt.Errorf("playbook %q version %q not found in history", name, version)
	}

	// Удаляем существующий тег с таким именем для этого плейбука
	tags := r.tags[name]
	for i, t := range tags {
		if t.Name == tag {
			tags = append(tags[:i], tags[i+1:]...)
			break
		}
	}

	// Добавляем новый тег
	r.tags[name] = append(tags, VersionTag{
		Name:      tag,
		Version:   version,
		CreatedAt: time.Now().UTC(),
	})

	r.logger.Info("playbook version tagged",
		"name", name,
		"version", version,
		"tag", tag,
	)

	return nil
}

// GetVersionByTag возвращает версию плейбука по тегу.
func (r *PlaybookRegistry) GetVersionByTag(name, tag string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tags, ok := r.tags[name]
	if !ok {
		return "", false
	}

	for _, t := range tags {
		if t.Name == tag {
			return t.Version, true
		}
	}

	return "", false
}

// ListTags возвращает все теги для указанного плейбука.
func (r *PlaybookRegistry) ListTags(name string) []VersionTag {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tags, ok := r.tags[name]
	if !ok {
		return nil
	}

	result := make([]VersionTag, len(tags))
	copy(result, tags)
	return result
}

// ═══════════════════════════════════════════════════════════════════════
// Query Methods
// ═══════════════════════════════════════════════════════════════════════

// Get возвращает плейбук по имени.
func (r *PlaybookRegistry) Get(name string) (*PlaybookSchema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[name]
	return s, ok
}

// GetVersion возвращает версию активного плейбука.
func (r *PlaybookRegistry) GetVersion(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.schemas[name]; ok {
		return s.Version
	}
	return ""
}

// List возвращает имена всех загруженных плейбуков.
func (r *PlaybookRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListActive возвращает только активные (не deprecated) плейбуки.
func (r *PlaybookRegistry) ListActive() []*PlaybookSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*PlaybookSchema
	for _, s := range r.schemas {
		if !s.Deprecated {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetHistory возвращает историю версий для плейбука.
func (r *PlaybookRegistry) GetHistory(name string) []VersionEntry {
	return r.GetVersionHistory(name)
}

// GetVersionHistory возвращает историю версий для плейбука.
//
// Является основным методом для получения версионной истории.
// GetHistory — алиас для обратной совместимости.
//
// Возвращает срез VersionEntry, отсортированный от старых к новым.
// Каждая запись содержит:
//   - Version: semver версия
//   - FilePath: путь к файлу
//   - SHA256: хеш файла (для audit trail)
//   - LoadedAt: время загрузки
//   - Active: является ли эта версия текущей активной
//   - RollbackOf: если это rollback — версия, с которой выполнен откат
func (r *PlaybookRegistry) GetVersionHistory(name string) []VersionEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	history, ok := r.versions[name]
	if !ok {
		return nil
	}
	result := make([]VersionEntry, len(history))
	copy(result, history)
	return result
}

// FindApplicable находит плейбуки, подходящие под условия.
func (r *PlaybookRegistry) FindApplicable(vendorType string, alarmMethod int, deviceType string, priority int) []*PlaybookSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*PlaybookSchema
	for _, s := range r.schemas {
		if s.Deprecated {
			continue
		}
		if matchesRules(s, vendorType, alarmMethod, deviceType, priority) {
			result = append(result, s)
		}
	}
	return result
}

// Count возвращает количество загруженных плейбуков.
func (r *PlaybookRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.schemas)
}

// ═══════════════════════════════════════════════════════════════════════
// Internal
// ═══════════════════════════════════════════════════════════════════════

// recordActiveTimeline добавляет версию в хронологическую последовательность
// активных версий (activeTimeline). Вызывается только при обычных загрузках
// через LoadFile, НЕ при rollback (loadFile с recordTimeline=false).
//
// Если версия уже является последней в timeline — не дублирует запись.
// Это гарантирует, что timeline содержит строгую последовательность
// уникальных версий без дубликатов при повторных загрузках того же файла.
func (r *PlaybookRegistry) recordActiveTimeline(name, version string) {
	timeline := r.activeTimeline[name]
	if len(timeline) > 0 && timeline[len(timeline)-1] == version {
		return // уже последняя, не дублируем
	}
	r.activeTimeline[name] = append(timeline, version)
}

func (r *PlaybookRegistry) addVersionLocked(name, version, filePath string, active bool) {
	r.addVersionLockedWithSHA(name, version, filePath, "", active)
}

func (r *PlaybookRegistry) addVersionLockedWithSHA(name, version, filePath, sha256Hex string, active bool) {
	entry := VersionEntry{
		Version:  version,
		FilePath: filePath,
		SHA256:   sha256Hex,
		LoadedAt: time.Now().UTC(),
		Active:   active,
	}

	r.versions[name] = append(r.versions[name], entry)

	// Ограничение истории
	if r.cfg.MaxVersionHistory > 0 && len(r.versions[name]) > r.cfg.MaxVersionHistory {
		r.versions[name] = r.versions[name][len(r.versions[name])-r.cfg.MaxVersionHistory:]
	}
}

func matchesRules(schema *PlaybookSchema, vendorType string, alarmMethod int, deviceType string, priority int) bool {
	if len(schema.Applicable) == 0 {
		return true
	}
	for _, rule := range schema.Applicable {
		if rule.VendorType != "" && rule.VendorType != vendorType {
			continue
		}
		if rule.AlarmMethod != 0 && rule.AlarmMethod != alarmMethod {
			continue
		}
		if rule.DeviceType != "" && rule.DeviceType != deviceType {
			continue
		}
		if rule.MinPriority > 0 && priority < rule.MinPriority {
			continue
		}
		return true
	}
	return false
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yml" || ext == ".yaml"
}
