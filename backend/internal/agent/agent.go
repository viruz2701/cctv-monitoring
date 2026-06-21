// Package agent — Self-Healing Agent: ядро автоматического восстановления устройств.
// Пайплайн: Alarm → DecisionTree → Playbook → Actions → CMMS/Notify.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gb-telemetry-collector/internal/events"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
)

// Agent — главный Self-Healing Agent.
type Agent struct {
	cfg       AgentConfig
	stateMgr  state.DeviceStateManager
	topology  *TopologyGraph
	decisions *DecisionTree
	playbooks *PlaybookRegistry
	publisher *events.Publisher
	executor  *ActionExecutor
	cmmsInt   *CMMSIntegrator
	approval  *ApprovalManager
	logger    *slog.Logger

	// Статистика
	mu         sync.RWMutex
	failureMap map[string]*deviceFailure // deviceID → failure history
	actions    []ActionLog

	// Callbacks (внедряются извне для CMMS/approval)
	OnAutoFixDone  func(ctx context.Context, result PlaybookResult, deviceID string)
	OnNeedApproval func(ctx context.Context, decision Decision, deviceID string)
	OnEscalate     func(ctx context.Context, decision Decision, deviceID string)
}

// AgentConfig — конфигурация агента.
type AgentConfig struct {
	Enabled            bool
	PlaybookDir        string
	TopologyRefreshSec int // интервал обновления топологии
	Logger             *slog.Logger
}

// deviceFailure — история сбоев устройства.
type deviceFailure struct {
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
	LastFix   time.Time
	FixCount  int
}

// ActionLog — запись о выполненном действии (audit trail).
type ActionLog struct {
	Timestamp  time.Time
	DeviceID   string
	Action     string
	Decision   DecisionLevel
	Playbook   string
	Success    bool
	Details    string
	ApprovedBy string
}

// NewAgent создаёт нового агента.
func NewAgent(cfg AgentConfig, stateMgr state.DeviceStateManager, pub *events.Publisher, executor *ActionExecutor, cmmsInt *CMMSIntegrator, approval *ApprovalManager) *Agent {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.TopologyRefreshSec <= 0 {
		cfg.TopologyRefreshSec = 300 // 5 минут
	}

	a := &Agent{
		cfg:        cfg,
		stateMgr:   stateMgr,
		decisions:  DefaultDecisionTree(),
		playbooks:  NewPlaybookRegistry(cfg.Logger),
		publisher:  pub,
		executor:   executor,
		cmmsInt:    cmmsInt,
		approval:   approval,
		logger:     cfg.Logger,
		failureMap: make(map[string]*deviceFailure),
	}

	return a
}

// Start запускает агент: загружает плейбуки, строит топологию, запускает фоновые задачи.
func (a *Agent) Start(ctx context.Context) error {
	if !a.cfg.Enabled {
		a.logger.Info("self-healing agent disabled")
		return nil
	}

	// Загружаем плейбуки
	if a.cfg.PlaybookDir != "" {
		if err := a.playbooks.LoadDir(a.cfg.PlaybookDir); err != nil {
			a.logger.Warn("failed to load playbooks from dir", "dir", a.cfg.PlaybookDir, "error", err)
		}
	}
	a.logger.Info("agent started", "playbooks", len(a.playbooks.List()))

	// Строим топологию
	a.topology = BuildFromState(a.stateMgr, a.logger)

	// Фоновое обновление топологии
	go a.topologyRefreshLoop(ctx)

	return nil
}

// Stop останавливает агента.
func (a *Agent) Stop() {
	a.logger.Info("agent stopped")
}

// HandleAlarm — точка входа: обрабатывает тревогу от NATS.
func (a *Agent) HandleAlarm(ctx context.Context, event events.AlarmEvent) {
	a.logger.Info("agent handling alarm",
		"device_id", event.DeviceID,
		"type", event.Type,
		"severity", event.Severity,
	)

	// Обновляем статистику сбоев
	a.recordFailure(event.DeviceID)

	// Получаем устройство из state
	dev, ok := a.stateMgr.Get(event.DeviceID)
	if !ok {
		a.logger.Warn("device not found in state", "device_id", event.DeviceID)
		// Создаём виртуальное устройство
		dev = &models.Device{
			DeviceID: event.DeviceID,
			Name:     event.DeviceName,
		}
	}

	fail := a.getFailure(event.DeviceID)

	// Конвертируем тип тревоги в AlarmMethod
	alarmMethod := mapAlarmMethod(event.Type)

	// Строим контекст решения
	decisionCtx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID:    event.DeviceID,
			Priority:    mapSeverity(event.Severity),
			Method:      alarmMethod,
			Timestamp:   event.Timestamp,
			Description: event.Message,
		},
		Device:          dev,
		Topology:        a.topology,
		FailureCount:    fail.Count,
		LastFixTime:     fail.LastFix,
		IsBusinessHours: a.decisions.isBusinessHours(),
	}

	// Принимаем решение
	decision := a.decisions.Decide(decisionCtx)
	a.logger.Info("agent decision",
		"device_id", event.DeviceID,
		"decision", decision.Level,
		"reason", decision.Reason,
	)

	// Действуем по решению
	switch decision.Level {
	case DecisionAutoFix:
		a.executeAutoFix(ctx, event.DeviceID, decision)

	case DecisionApprove:
		a.requestApproval(ctx, decision, event.DeviceID)

	case DecisionEscalate:
		a.escalate(ctx, decision, event.DeviceID)

	case DecisionIgnore:
		a.logger.Info("alarm ignored", "device_id", event.DeviceID, "reason", decision.Reason)

	case DecisionSchedule:
		a.scheduleMaintenance(ctx, event.DeviceID, decision)
	}
}

// executeAutoFix выполняет автоматическое исправление.
func (a *Agent) executeAutoFix(ctx context.Context, deviceID string, decision Decision) {
	pb, ok := a.playbooks.Get(decision.PlaybookRef)
	if !ok {
		a.logger.Warn("playbook not found", "playbook", decision.PlaybookRef)
		return
	}

	if !a.playbooks.CanRun(pb.Name, deviceID) {
		a.logger.Info("playbook cooldown active", "playbook", pb.Name, "device", deviceID)
		return
	}

	a.playbooks.MarkRun(pb.Name, deviceID)
	a.logger.Info("executing playbook", "playbook", pb.Name, "device", deviceID)

	result := PlaybookResult{
		PlaybookName: pb.Name,
		StartedAt:    time.Now(),
		StepsTotal:   len(pb.Steps),
	}

	allSuccess := true
	for i, step := range pb.Steps {
		stepResult := a.executeStep(ctx, step, deviceID, pb.MaxRetries)
		result.StepResults = append(result.StepResults, stepResult)
		result.StepsDone = i + 1

		if !stepResult.Success {
			allSuccess = false
			switch step.OnFailure {
			case "abort":
				result.Error = fmt.Sprintf("step %s failed: %s", step.Name, stepResult.Error)
				result.FinishedAt = time.Now()
				result.Success = false
				a.logAction(deviceID, "playbook_aborted", decision.Level, pb.Name, false, result.Error)
				return
			case "escalate":
				result.Error = fmt.Sprintf("step %s failed: %s", step.Name, stepResult.Error)
				result.FinishedAt = time.Now()
				result.Success = false
				a.logAction(deviceID, "playbook_escalated", decision.Level, pb.Name, false, result.Error)
				a.escalate(ctx, Decision{
					Level:      DecisionEscalate,
					Reason:     result.Error,
					EscalateTo: "engineer_oncall",
				}, deviceID)
				return
			} // "continue" — идём дальше
		}
	}

	result.FinishedAt = time.Now()
	result.Success = allSuccess

	a.logAction(deviceID, "playbook_completed", decision.Level, pb.Name, result.Success, "")

	if result.Success {
		a.markFixed(deviceID)
		a.logger.Info("playbook succeeded", "playbook", pb.Name, "device", deviceID, "duration", result.FinishedAt.Sub(result.StartedAt))
		if a.OnAutoFixDone != nil {
			a.OnAutoFixDone(ctx, result, deviceID)
		}
	} else {
		a.logger.Warn("playbook failed", "playbook", pb.Name, "device", deviceID, "error", result.Error)
	}
}

// executeStep выполняет один шаг плейбука с ретраями.
func (a *Agent) executeStep(ctx context.Context, step PlaybookStep, deviceID string, maxRetries int) StepResult {
	start := time.Now()
	var lastErr error

	for attempt := 0; attempt <= min(step.Retries, maxRetries); attempt++ {
		if attempt > 0 {
			delay := step.retryDelay
			if delay == 0 {
				delay = 5 * time.Second
			}
			a.logger.Info("retrying step", "step", step.Name, "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return StepResult{
					StepName:   step.Name,
					Action:     step.Action,
					Success:    false,
					Duration:   time.Since(start),
					Error:      ctx.Err().Error(),
					RetryCount: attempt,
				}
			case <-time.After(delay):
			}
		}

		output, err := a.dispatchAction(ctx, step, deviceID)
		if err == nil {
			return StepResult{
				StepName:   step.Name,
				Action:     step.Action,
				Success:    true,
				Duration:   time.Since(start),
				Output:     output,
				RetryCount: attempt,
			}
		}
		lastErr = err
		a.logger.Warn("step failed", "step", step.Name, "attempt", attempt+1, "error", err)
	}

	return StepResult{
		StepName:   step.Name,
		Action:     step.Action,
		Success:    false,
		Duration:   time.Since(start),
		Error:      lastErr.Error(),
		RetryCount: min(step.Retries, maxRetries),
	}
}

// dispatchAction выполняет конкретное действие.
// Действия ISAPI/ONVIF/SNMP/SSH будут реализованы в actions.go (Epic 3.3.2).
func (a *Agent) dispatchAction(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	// Таймаут для шага
	timeout := step.timeoutDur
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch step.Action {
	case ActionWait:
		dur, err := time.ParseDuration(step.Params["duration"])
		if err != nil {
			dur = 10 * time.Second
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(dur):
			return "wait completed", nil
		}

	case ActionHealthCheck:
		dev, ok := a.stateMgr.Get(deviceID)
		if !ok {
			return "", fmt.Errorf("device %s not found", deviceID)
		}
		if dev.Status == models.StatusOnline {
			return "device online", nil
		}
		return "", fmt.Errorf("device %s is %s", deviceID, dev.Status)

	case ActionNotify:
		msg := step.Params["message"]
		if msg == "" {
			msg = fmt.Sprintf("Action %s executed for device %s", step.Name, deviceID)
		}
		a.logger.Info("agent notification", "device", deviceID, "message", msg)
		return "notified", nil

	// ISAPI/ONVIF/SNMP/SSH — через ActionExecutor
	case ActionISAPIReboot:
		return a.dispatchISAPIReboot(ctx, step, deviceID)
	case ActionISAPIReset:
		return a.dispatchISAPIReset(ctx, step, deviceID)
	case ActionISAPIRestore:
		return a.dispatchISAPIRestore(ctx, step, deviceID)
	case ActionONVIFReboot:
		return a.dispatchONVIFReboot(ctx, step, deviceID)
	case ActionONVIFPTZHome:
		return a.dispatchONVIFPTZHome(ctx, step, deviceID)
	case ActionSNMPReset:
		return a.dispatchSNMPReset(ctx, step, deviceID)
	case ActionSNMPColdStart:
		return a.dispatchSNMPColdStart(ctx, step, deviceID)
	case ActionSSHRestart:
		return a.dispatchSSHRestart(ctx, step, deviceID)
	case ActionSSHServiceRestart:
		return a.dispatchSSHServiceRestart(ctx, step, deviceID)

	case ActionCreateTicket:
		return a.dispatchCreateTicket(ctx, step, deviceID)
	case ActionCloseTicket:
		return a.dispatchCloseTicket(ctx, step, deviceID)

	default:
		return "", fmt.Errorf("unknown action: %s", step.Action)
	}
}

// ── Action dispatchers ─────────────────────────────────────────────

func (a *Agent) resolveDeviceIP(step PlaybookStep, deviceID string) string {
	if ip := step.Params["device_ip"]; ip != "" {
		return ip
	}
	if dev, ok := a.stateMgr.Get(deviceID); ok {
		return dev.Location
	}
	return ""
}

func (a *Agent) dispatchISAPIReboot(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	if ip == "" {
		return "", fmt.Errorf("isapi_reboot: device_ip not specified")
	}
	return "isapi reboot sent", a.executor.ISAPIReboot(ctx, ip, step.Params["username"], step.Params["password"])
}

func (a *Agent) dispatchISAPIReset(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	mode := step.Params["reset_type"]
	return "isapi reset sent", a.executor.ISAPIReset(ctx, ip, step.Params["username"], step.Params["password"], mode)
}

func (a *Agent) dispatchISAPIRestore(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	return "isapi restore sent", a.executor.ISAPIRestore(ctx, ip, step.Params["username"], step.Params["password"], nil)
}

func (a *Agent) dispatchONVIFReboot(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	return "onvif reboot sent", a.executor.ONVIFReboot(ctx, ip, step.Params["username"], step.Params["password"])
}

func (a *Agent) dispatchONVIFPTZHome(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	return "onvif ptz home sent", a.executor.ONVIFPTZHome(ctx, ip, step.Params["username"], step.Params["password"])
}

func (a *Agent) dispatchSNMPReset(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	snmpCfg := DefaultSNMPConfig()
	if step.Params["community"] != "" {
		snmpCfg.Community = step.Params["community"]
	}
	return "snmp reset sent", a.executor.SNMPReset(ctx, ip, snmpCfg)
}

func (a *Agent) dispatchSNMPColdStart(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	snmpCfg := DefaultSNMPConfig()
	if step.Params["community"] != "" {
		snmpCfg.Community = step.Params["community"]
	}
	return "snmp cold start sent", a.executor.SNMPColdStart(ctx, ip, snmpCfg)
}

func (a *Agent) dispatchSSHRestart(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	return "ssh restart sent", a.executor.SSHRestartDevice(ctx, ip, step.Params["username"], step.Params["password"], 22)
}

func (a *Agent) dispatchSSHServiceRestart(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.executor == nil {
		return "", fmt.Errorf("action executor not configured")
	}
	ip := a.resolveDeviceIP(step, deviceID)
	svc := step.Params["service_name"]
	if svc == "" {
		svc = "camera-service"
	}
	return "ssh service restart sent", a.executor.SSHServiceRestart(ctx, ip, step.Params["username"], step.Params["password"], svc, 22)
}

func (a *Agent) dispatchCreateTicket(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.cmmsInt == nil {
		return "", fmt.Errorf("cmms integrator not configured")
	}
	dev, _ := a.stateMgr.Get(deviceID)
	deviceName := deviceID
	if dev != nil && dev.Name != "" {
		deviceName = dev.Name
	}
	alarmType := step.Params["alarm_type"]
	severity := step.Params["severity"]
	desc := step.Params["description"]
	ticketID, err := a.cmmsInt.AutoCreateTicket(ctx, deviceID, deviceName, alarmType, severity, desc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ticket %s created", ticketID), nil
}

func (a *Agent) dispatchCloseTicket(ctx context.Context, step PlaybookStep, deviceID string) (string, error) {
	if a.cmmsInt == nil {
		return "", fmt.Errorf("cmms integrator not configured")
	}
	ticketID := step.Params["ticket_id"]
	resolution := step.Params["resolution"]
	if resolution == "" {
		resolution = "Self-healing completed successfully"
	}
	if err := a.cmmsInt.AutoCloseTicket(ctx, deviceID, ticketID, resolution); err != nil {
		return "", err
	}
	return "ticket closed", nil
}

// requestApproval запрашивает подтверждение у человека.
func (a *Agent) requestApproval(ctx context.Context, decision Decision, deviceID string) {
	a.logAction(deviceID, "approval_requested", decision.Level, decision.PlaybookRef, false, decision.Reason)
	if a.OnNeedApproval != nil {
		a.OnNeedApproval(ctx, decision, deviceID)
	}
}

// escalate эскалирует тревогу инженеру.
func (a *Agent) escalate(ctx context.Context, decision Decision, deviceID string) {
	a.logAction(deviceID, "escalated", decision.Level, decision.PlaybookRef, false, decision.Reason)
	if a.OnEscalate != nil {
		a.OnEscalate(ctx, decision, deviceID)
	}
}

// scheduleMaintenance планирует обслуживание.
func (a *Agent) scheduleMaintenance(_ context.Context, deviceID string, decision Decision) {
	a.logAction(deviceID, "maintenance_scheduled", decision.Level, decision.PlaybookRef, true, decision.Reason)
	a.logger.Info("maintenance scheduled", "device", deviceID, "reason", decision.Reason)
}

// ── Failure tracking ───────────────────────────────────────────────

func (a *Agent) recordFailure(deviceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	f, ok := a.failureMap[deviceID]
	if !ok {
		f = &deviceFailure{FirstSeen: time.Now()}
		a.failureMap[deviceID] = f
	}
	f.Count++
	f.LastSeen = time.Now()
}

func (a *Agent) getFailure(deviceID string) *deviceFailure {
	a.mu.RLock()
	defer a.mu.RUnlock()

	f, ok := a.failureMap[deviceID]
	if !ok {
		return &deviceFailure{}
	}
	return f
}

func (a *Agent) markFixed(deviceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	f, ok := a.failureMap[deviceID]
	if !ok {
		return
	}
	f.LastFix = time.Now()
	f.FixCount++
	f.Count = 0 // сбрасываем счётчик после успешного исправления
}

// ── Audit trail ────────────────────────────────────────────────────

func (a *Agent) logAction(deviceID, action string, level DecisionLevel, playbook string, success bool, details string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.actions = append(a.actions, ActionLog{
		Timestamp: time.Now(),
		DeviceID:  deviceID,
		Action:    action,
		Decision:  level,
		Playbook:  playbook,
		Success:   success,
		Details:   details,
	})

	// Ограничиваем историю 10000 записей
	if len(a.actions) > 10000 {
		a.actions = a.actions[len(a.actions)-5000:]
	}
}

// GetActions возвращает копию истории действий.
func (a *Agent) GetActions() []ActionLog {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]ActionLog, len(a.actions))
	copy(result, a.actions)
	return result
}

// GetFailureStats возвращает статистику сбоев.
func (a *Agent) GetFailureStats() map[string]*deviceFailure {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]*deviceFailure, len(a.failureMap))
	for k, v := range a.failureMap {
		cp := *v
		result[k] = &cp
	}
	return result
}

// ── Background tasks ───────────────────────────────────────────────

func (a *Agent) topologyRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.cfg.TopologyRefreshSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.topology = BuildFromState(a.stateMgr, a.logger)
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────

func mapAlarmMethod(eventType string) models.AlarmMethod {
	switch eventType {
	case "motion", "motion_detection", "VMD", "Motion":
		return models.AlarmMethodMotionDetection
	case "video_loss", "videoLoss", "VideoLoss":
		return models.AlarmMethodVideoLoss
	case "equipment_fault", "hardware", "fault", "EquipmentFault":
		return models.AlarmMethodEquipmentFault
	default:
		return models.AlarmMethodMotionDetection
	}
}

func mapSeverity(severity string) models.AlarmPriority {
	switch severity {
	case "critical", "high":
		return models.AlarmPriorityHigh
	case "medium":
		return models.AlarmPriorityMedium
	default:
		return models.AlarmPriorityLow
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
