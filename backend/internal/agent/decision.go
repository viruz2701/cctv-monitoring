// Package agent — decision tree: auto-fix vs human-approval vs escalate.
package agent

import (
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
type DecisionTree struct {
	// Пороги для auto-fix
	MaxAutoFixRetries  int
	MaxAutoFixPerDay   int
	ApprovalTimeout    time.Duration
	BusinessHoursStart int // 8
	BusinessHoursEnd   int // 20
}

// DefaultDecisionTree возвращает дерево с разумными дефолтами.
func DefaultDecisionTree() *DecisionTree {
	return &DecisionTree{
		MaxAutoFixRetries:  3,
		MaxAutoFixPerDay:   10,
		ApprovalTimeout:    5 * time.Minute,
		BusinessHoursStart: 8,
		BusinessHoursEnd:   20,
	}
}

// Decide принимает решение по тревоге.
func (dt *DecisionTree) Decide(ctx DecisionContext) Decision {
	// 1. Ложное срабатывание / дубликат
	if ctx.FailureCount > 10 && time.Since(ctx.LastFixTime) < 5*time.Minute {
		return Decision{
			Level:  DecisionIgnore,
			Reason: "flapping: too many failures in short window",
		}
	}

	// 2. Приоритет Critical → escalate сразу
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

	// 3. Определяем, можно ли auto-fix
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

	// 4. Требуется подтверждение
	if ctx.IsBusinessHours || dt.isBusinessHours() {
		return Decision{
			Level:       DecisionApprove,
			Reason:      "requires human approval during business hours",
			ApprovalTTL: dt.ApprovalTimeout,
			Metadata: map[string]string{
				"device_id": ctx.Alarm.DeviceID,
			},
		}
	}

	// 5. Не бизнес-часы → эскалация
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
