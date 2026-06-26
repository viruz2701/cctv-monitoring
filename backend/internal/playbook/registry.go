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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	if c.MaxVersionHistory <= 0 {
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
	files    map[string]string          // filePath → name (для hot reload)
	dir      string                     // директория с плейбуками
	cfg      RegistryConfig
	logger   *slog.Logger
	stopCh   chan struct{}
}

// NewPlaybookRegistry создаёт новый PlaybookRegistry.
func NewPlaybookRegistry(cfg RegistryConfig) *PlaybookRegistry {
	cfg.validate()
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &PlaybookRegistry{
		schemas:  make(map[string]*PlaybookSchema),
		versions: make(map[string][]VersionEntry),
		files:    make(map[string]string),
		cfg:      cfg,
		logger:   cfg.Logger.With("component", "playbook-registry"),
		stopCh:   make(chan struct{}),
	}
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

// LoadFile загружает один playbook-файл.
//
// Если плейбук с таким именем уже существует — сохраняет предыдущую
// версию в истории и активирует новую.
func (r *PlaybookRegistry) LoadFile(path string) error {
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
	if len(schema.Steps) == 0 {
		return fmt.Errorf("%s: playbook %q must have at least one step", path, schema.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Сохраняем предыдущую версию в историю (если есть)
	if existing, ok := r.schemas[schema.Name]; ok {
		r.addVersionLocked(existing.Name, existing.Version, path, false)
	}

	// Активируем новую версию
	r.schemas[schema.Name] = &schema
	r.files[path] = schema.Name
	r.addVersionLocked(schema.Name, schema.Version, path, true)

	r.logger.Info("playbook loaded/updated",
		"name", schema.Name,
		"version", schema.Version,
		"file", path,
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
// Возвращает:
//   - true + nil — rollback successful
//   - false + nil — версия не найдена (не ошибка)
//   - false + error — ошибка при rollback
func (r *PlaybookRegistry) Rollback(name, version string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	history, ok := r.versions[name]
	if !ok {
		return false, nil
	}

	// Ищем указанную версию
	var target *VersionEntry
	for i, entry := range history {
		if entry.Version == version {
			target = &history[i]
			break
		}
	}
	if target == nil {
		return false, nil
	}

	// Сохраняем текущую версию в историю перед rollback
	if current, ok := r.schemas[name]; ok {
		r.addVersionLocked(current.Name, current.Version, "", false)
		r.schemas[name] = &PlaybookSchema{
			Name:    name,
			Version: target.Version,
			Tags:    []string{"rollback"},
		}
	}

	// Загружаем из файла (если есть)
	if target.FilePath != "" {
		r.mu.Unlock()
		err := r.LoadFile(target.FilePath)
		r.mu.Lock()
		if err != nil {
			return false, fmt.Errorf("rollback %s@%s: %w", name, version, err)
		}
	}

	// Помечаем в истории как rollback
	history = r.versions[name]
	for i := range history {
		if history[i].Version == version {
			history[i].Active = true
		} else {
			history[i].Active = false
		}
	}

	r.logger.Warn("playbook rollback performed",
		"name", name,
		"version", version,
	)

	return true, nil
}

// RollbackLatest откатывает последнее изменение плейбука.
func (r *PlaybookRegistry) RollbackLatest(name string) (bool, error) {
	r.mu.RLock()
	history, ok := r.versions[name]
	r.mu.RUnlock()

	if !ok || len(history) < 2 {
		return false, nil
	}

	// Ищем предыдущую активную версию
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Active && history[i].Version != r.GetVersion(name) {
			return r.Rollback(name, history[i].Version)
		}
		// Первая не-active версия перед последней активной
		if !history[i].Active && i > 0 && history[i-1].Active {
			return r.Rollback(name, history[i-1].Version)
		}
	}

	return false, nil
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

func (r *PlaybookRegistry) addVersionLocked(name, version, filePath string, active bool) {
	entry := VersionEntry{
		Version:  version,
		FilePath: filePath,
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
