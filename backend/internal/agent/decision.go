// Package agent — decision tree: auto-fix vs human-approval vs escalate.
package agent

import (
	"sync"
	"time"

	"gb-telemetry-collector/internal/models"
)

// DecisionLevel — уровень решения по тревоге.
type DecisionLevel string

const (
	DecisionAutoFix  DecisionLevel = "auto_fix" // исправить автоматически
	DecisionApprove  DecisionLevel = "approve"  // требуется подтверждение человека
	DecisionEscalate DecisionLevel = "escalate" // эскалация инженеру
	DecisionIgnore   DecisionLevel = "ignore"   // игнорировать (дубликат/ложное срабатывание)
	DecisionSchedule DecisionLevel = "schedule" // запланировать обслуживание
)

// DecisionContext содержит контекст для принятия решения.
type DecisionContext struct {
	Alarm           models.Alarm
	Device          *models.Device
	Topology        *TopologyGraph
	FailureCount    int       // количество срабатываний за последние 24h
	LastFixTime     time.Time // время последнего успешного self-healing
	IsBusinessHours bool      // true если 8:00-20:00 будни
}

// Decision — результат работы decision tree.
type Decision struct {
	Level       DecisionLevel
	Reason      string
	PlaybookRef string            // ссылка на playbook (если auto_fix)
	MaxRetries  int               // макс. попыток авто-исправления
	ApprovalTTL time.Duration     // таймаут для human-approval
	EscalateTo  string            // роль/пользователь для эскалации
	Metadata    map[string]string // доп. контекст
}

// DecisionTree — дерево принятия решений.
//
// CCTV-2.3.3: Cooldown & rate limiting — MaxAutoFixPerDay + global rate limiter.
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — DoS protection)
//   - ISO 27001 A.12.6.1 (Capacity management)
type DecisionTree struct {
	// Пороги для auto-fix
	MaxAutoFixRetries  int           // макс. ретраев на одно устройство
	MaxAutoFixPerDay   int           // глобальный лимит auto-fix в день
	flappingWindow     time.Duration // окно для детекции флаппинга (по умолч. 5min)
	flappingThreshold  int           // кол-во сбоев в окне для флаппинга (по умолч. 10)
	ApprovalTimeout    time.Duration
	BusinessHoursStart int // 8
	BusinessHoursEnd   int // 20

	// Rate limiter (CCTV-2.3.3)
	mu             sync.Mutex
	autoFixCount   int              // счётчик auto-fix за сегодня
	autoFixResetAt time.Time        // время сброса счётчика
	autoFixHistory map[string][]time.Time // deviceID → история auto-fix за день
}

// DefaultDecisionTree возвращает дерево с разумными дефолтами.
func DefaultDecisionTree() *DecisionTree {
	return &DecisionTree{
		MaxAutoFixRetries:  3,
		MaxAutoFixPerDay:   10,
		flappingWindow:     5 * time.Minute,
		flappingThreshold:  10,
		ApprovalTimeout:    5 * time.Minute,
		BusinessHoursStart: 8,
		BusinessHoursEnd:   20,
		autoFixHistory:     make(map[string][]time.Time),
		autoFixResetAt:     time.Now().Add(24 * time.Hour),
	}
}

// Decide принимает решение по тревоге.
func (dt *DecisionTree) Decide(ctx DecisionContext) Decision {
	// 1. Флаппинг детекция (CCTV-2.3.3)
	if ctx.FailureCount > dt.flappingThreshold && time.Since(ctx.LastFixTime) < dt.flappingWindow {
		return Decision{
			Level:  DecisionIgnore,
			Reason: "flapping: too many failures in short window",
		}
	}

	// 2. Глобальный rate limit (CCTV-2.3.3)
	if !dt.allowAutoFix(ctx.Alarm.DeviceID) {
		return Decision{
			Level:      DecisionSchedule,
			Reason:     "global auto-fix rate limit exceeded for today",
			Metadata:   map[string]string{"device_id": ctx.Alarm.DeviceID},
		}
	}

	// 3. Приоритет Critical → escalate сразу
	if ctx.Alarm.Priority == models.AlarmPriorityHigh {
		return Decision{
			Level:      DecisionEscalate,
			Reason:     "critical priority alarm requires human intervention",
			EscalateTo: "engineer_oncall",
			Metadata: map[string]string{
				"priority": "critical",
			},
		}
	}

	// 4. Определяем, можно ли auto-fix
	autoFixable := dt.isAutoFixable(ctx)

	if autoFixable {
		// Проверяем лимиты
		if ctx.FailureCount > dt.MaxAutoFixRetries {
			return Decision{
				Level:      DecisionEscalate,
				Reason:     "exceeded max auto-fix retries",
				EscalateTo: "engineer_oncall",
				Metadata: map[string]string{
					"failure_count": itoa(ctx.FailureCount),
					"max_retries":   itoa(dt.MaxAutoFixRetries),
				},
			}
		}

		return Decision{
			Level:       DecisionAutoFix,
			Reason:      "auto-fixable failure detected",
			PlaybookRef: dt.selectPlaybook(ctx),
			MaxRetries:  dt.MaxAutoFixRetries,
			Metadata: map[string]string{
				"device_id": ctx.Alarm.DeviceID,
				"method":    itoa(int(ctx.Alarm.Method)),
			},
		}
	}

	// 5. Требуется подтверждение
	// IsBusinessHours берётся из контекста (передан агентом).
	// Если в контексте не указано — fallback на dt.isBusinessHours().
	needsApproval := ctx.IsBusinessHours
	if !needsApproval && ctx.FailureCount == 0 {
		// Fallback: если IsBusinessHours не задан явно (zero-value),
		// проверяем по системному времени
		needsApproval = dt.isBusinessHours()
	}
	if needsApproval {
		return Decision{
			Level:       DecisionApprove,
			Reason:      "requires human approval during business hours",
			ApprovalTTL: dt.ApprovalTimeout,
			Metadata: map[string]string{
				"device_id": ctx.Alarm.DeviceID,
			},
		}
	}

	// 6. Не бизнес-часы → эскалация
	return Decision{
		Level:      DecisionEscalate,
		Reason:     "non-business hours: escalate to on-call engineer",
		EscalateTo: "engineer_oncall",
		Metadata: map[string]string{
			"priority": "medium",
		},
	}
}

// isAutoFixable определяет, можно ли исправить авто-действием.
func (dt *DecisionTree) isAutoFixable(ctx DecisionContext) bool {
	// VideoLoss → reboot камеры
	if ctx.Alarm.Method == models.AlarmMethodVideoLoss {
		return true
	}

	// EquipmentFault → зависит от типа устройства
	if ctx.Alarm.Method == models.AlarmMethodEquipmentFault {
		// SNMP/ISAPI устройства можно ресетить
		if ctx.Device != nil {
			switch ctx.Device.VendorType {
			case "Hikvision", "Dahua", "Dahua/Intelbras":
				return true
			}
		}
		return false
	}

	// Motion detection — не auto-fix (это норма)
	return false
}

// selectPlaybook выбирает подходящий playbook.
func (dt *DecisionTree) selectPlaybook(ctx DecisionContext) string {
	switch ctx.Alarm.Method {
	case models.AlarmMethodVideoLoss:
		return "reboot_camera"
	case models.AlarmMethodEquipmentFault:
		if ctx.Device != nil && ctx.Device.VendorType == "Hikvision" {
			return "hikvision_diagnostic"
		}
		return "camera_diagnostic"
	default:
		return "default_diagnostic"
	}
}

// ── Rate Limiter (CCTV-2.3.3) ───────────────────────────────────────

// allowAutoFix проверяет глобальный лимит auto-fix за день.
// Если превышен MaxAutoFixPerDay — возвращает false (требуется approval).
func (dt *DecisionTree) allowAutoFix(deviceID string) bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Сброс счётчика раз в день
	now := time.Now()
	if now.After(dt.autoFixResetAt) {
		dt.autoFixCount = 0
		dt.autoFixHistory = make(map[string][]time.Time)
		dt.autoFixResetAt = now.Add(24 * time.Hour)
	}

	// Глобальный лимит
	if dt.MaxAutoFixPerDay > 0 && dt.autoFixCount >= dt.MaxAutoFixPerDay {
		return false
	}

	// Per-device rate limit: макс. 3 auto-fix в день на одно устройство
	deviceFixes := dt.autoFixHistory[deviceID]
	if len(deviceFixes) >= 3 {
		return false
	}

	// Per-device cooldown: макс. 1 auto-fix в 30 минут
	if len(deviceFixes) > 0 {
		lastFix := deviceFixes[len(deviceFixes)-1]
		if now.Sub(lastFix) < 30*time.Minute {
			return false
		}
	}

	// Разрешаем auto-fix и записываем
	dt.autoFixCount++
	dt.autoFixHistory[deviceID] = append(dt.autoFixHistory[deviceID], now)
	return true
}

// GetRateLimitStats возвращает статистику rate limiter'а.
func (dt *DecisionTree) GetRateLimitStats() map[string]interface{} {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	totalDevices := len(dt.autoFixHistory)
	totalFixes := dt.autoFixCount
	remaining := dt.MaxAutoFixPerDay - totalFixes
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"auto_fix_today":     totalFixes,
		"auto_fix_remaining": remaining,
		"auto_fix_limit":     dt.MaxAutoFixPerDay,
		"devices_today":      totalDevices,
		"resets_at":          dt.autoFixResetAt.Format(time.RFC3339),
	}
}

func (dt *DecisionTree) isBusinessHours() bool {
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	return hour >= dt.BusinessHoursStart && hour < dt.BusinessHoursEnd
}

func itoa(n int) string {
	return fmtInt(n)
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
