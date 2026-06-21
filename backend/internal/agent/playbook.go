// Package agent — playbook engine: YAML-based remediation workflows.
package agent

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"
)

// PlaybookAction — тип действия в плейбуке.
type PlaybookAction string

const (
	ActionISAPIReboot       PlaybookAction = "isapi_reboot"
	ActionISAPIReset        PlaybookAction = "isapi_reset"
	ActionISAPIRestore      PlaybookAction = "isapi_restore"
	ActionONVIFReboot       PlaybookAction = "onvif_reboot"
	ActionONVIFPTZHome      PlaybookAction = "onvif_ptz_home"
	ActionSNMPReset         PlaybookAction = "snmp_reset"
	ActionSNMPColdStart     PlaybookAction = "snmp_cold_start"
	ActionSSHRestart        PlaybookAction = "ssh_restart"
	ActionSSHServiceRestart PlaybookAction = "ssh_service_restart"
	ActionWait              PlaybookAction = "wait"
	ActionHealthCheck       PlaybookAction = "health_check"
	ActionNotify            PlaybookAction = "notify"
	ActionCreateTicket      PlaybookAction = "create_ticket"
	ActionCloseTicket       PlaybookAction = "close_ticket"
)

// PlaybookStep — один шаг плейбука.
type PlaybookStep struct {
	Name       string            `yaml:"name"`
	Action     PlaybookAction    `yaml:"action"`
	Target     string            `yaml:"target"`      // device_id, switch_id, self
	Timeout    string            `yaml:"timeout"`     // "30s", "2m"
	Retries    int               `yaml:"retries"`     // 0 = без повторов
	RetryDelay string            `yaml:"retry_delay"` // "5s"
	OnFailure  string            `yaml:"on_failure"`  // "continue", "abort", "escalate"
	Params     map[string]string `yaml:"params"`      // дополнительные параметры
	timeoutDur time.Duration     // распарсенный timeout
	retryDelay time.Duration     // распарсенный retry_delay
}

// Playbook — YAML-определение плейбука самовосстановления.
type Playbook struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Version     string         `yaml:"version"`
	Applicable  []PlaybookRule `yaml:"applicable"`
	Steps       []PlaybookStep `yaml:"steps"`
	MaxRetries  int            `yaml:"max_retries"`
	Cooldown    string         `yaml:"cooldown"` // "5m" — минимальный интервал между запусками
	cooldownDur time.Duration  // распарсенный cooldown
}

// PlaybookRule — условие применимости плейбука.
type PlaybookRule struct {
	VendorType  string `yaml:"vendor_type"`  // Hikvision, Dahua, etc.
	AlarmMethod int    `yaml:"alarm_method"` // models.AlarmMethod*
	DeviceType  string `yaml:"device_type"`  // camera, switch, nvr
	MinPriority int    `yaml:"min_priority"`
}

// PlaybookResult — результат выполнения плейбука.
type PlaybookResult struct {
	PlaybookName string
	Success      bool
	StepsTotal   int
	StepsDone    int
	StepResults  []StepResult
	StartedAt    time.Time
	FinishedAt   time.Time
	Error        string
}

// StepResult — результат одного шага.
type StepResult struct {
	StepName   string
	Action     PlaybookAction
	Success    bool
	Duration   time.Duration
	Output     string
	Error      string
	RetryCount int
}

// PlaybookRegistry — реестр загруженных плейбуков.
type PlaybookRegistry struct {
	mu        sync.RWMutex
	playbooks map[string]*Playbook // name → playbook
	lastRun   map[string]time.Time // playbook:deviceID → last run
	logger    *slog.Logger
}

// NewPlaybookRegistry создаёт новый реестр.
func NewPlaybookRegistry(logger *slog.Logger) *PlaybookRegistry {
	return &PlaybookRegistry{
		playbooks: make(map[string]*Playbook),
		lastRun:   make(map[string]time.Time),
		logger:    logger,
	}
}

// LoadDir загружает все *.yml / *.yaml файлы из директории.
func (r *PlaybookRegistry) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read playbook dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < 5 {
			continue
		}
		ext := name[len(name)-4:]
		if ext != ".yml" && name[len(name)-5:] != ".yaml" {
			continue
		}

		path := dir + "/" + name
		if err := r.LoadFile(path); err != nil {
			r.logger.Warn("failed to load playbook", "file", path, "error", err)
			continue
		}
		r.logger.Info("playbook loaded", "file", path)
	}
	return nil
}

// LoadFile загружает один playbook-файл.
func (r *PlaybookRegistry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var pb Playbook
	if err := yaml.Unmarshal(data, &pb); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if pb.Name == "" {
		return fmt.Errorf("playbook %s has no name", path)
	}

	// Парсим длительности
	if pb.Cooldown != "" {
		pb.cooldownDur, err = time.ParseDuration(pb.Cooldown)
		if err != nil {
			return fmt.Errorf("playbook %s: invalid cooldown %q: %w", pb.Name, pb.Cooldown, err)
		}
	}
	if pb.MaxRetries <= 0 {
		pb.MaxRetries = 3
	}

	for i := range pb.Steps {
		if pb.Steps[i].Timeout != "" {
			pb.Steps[i].timeoutDur, err = time.ParseDuration(pb.Steps[i].Timeout)
			if err != nil {
				return fmt.Errorf("playbook %s step %s: invalid timeout: %w", pb.Name, pb.Steps[i].Name, err)
			}
		}
		if pb.Steps[i].RetryDelay != "" {
			pb.Steps[i].retryDelay, err = time.ParseDuration(pb.Steps[i].RetryDelay)
			if err != nil {
				return fmt.Errorf("playbook %s step %s: invalid retry_delay: %w", pb.Name, pb.Steps[i].Name, err)
			}
		}
	}

	r.mu.Lock()
	r.playbooks[pb.Name] = &pb
	r.mu.Unlock()
	return nil
}

// Get возвращает плейбук по имени.
func (r *PlaybookRegistry) Get(name string) (*Playbook, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pb, ok := r.playbooks[name]
	return pb, ok
}

// FindApplicable находит все плейбуки, подходящие под условия.
func (r *PlaybookRegistry) FindApplicable(vendorType string, alarmMethod int, deviceType string, priority int) []*Playbook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Playbook
	for _, pb := range r.playbooks {
		if r.matchesRules(pb, vendorType, alarmMethod, deviceType, priority) {
			result = append(result, pb)
		}
	}
	return result
}

// CanRun проверяет, можно ли запустить плейбук (cooldown).
func (r *PlaybookRegistry) CanRun(playbookName, deviceID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pb, ok := r.playbooks[playbookName]
	if !ok {
		return false
	}

	key := playbookName + ":" + deviceID
	last, exists := r.lastRun[key]
	if !exists {
		return true
	}

	if pb.cooldownDur == 0 {
		return true
	}

	return time.Since(last) >= pb.cooldownDur
}

// MarkRun отмечает факт запуска плейбука.
func (r *PlaybookRegistry) MarkRun(playbookName, deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRun[playbookName+":"+deviceID] = time.Now()
}

// List возвращает список имён загруженных плейбуков.
func (r *PlaybookRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.playbooks))
	for name := range r.playbooks {
		names = append(names, name)
	}
	return names
}

func (r *PlaybookRegistry) matchesRules(pb *Playbook, vendorType string, alarmMethod int, deviceType string, priority int) bool {
	if len(pb.Applicable) == 0 {
		return true // без правил — применим ко всем
	}
	for _, rule := range pb.Applicable {
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
