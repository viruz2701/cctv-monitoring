package agent

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// TrafficPriority defines MQTT QoS mapping for traffic types.
//
// Compliance: IEC 62443-3-3 SR 7.1 — Resource Availability
type TrafficPriority int

const (
	// PriorityDiagnostics maps to MQTT QoS 0 — no rate limit, circuit breaker only.
	PriorityDiagnostics TrafficPriority = iota
	// PriorityTelemetry maps to MQTT QoS 1 — rate limited by token bucket.
	PriorityTelemetry
	// PriorityAlert maps to MQTT QoS 2 — bypasses all rate limits and circuit breaker.
	PriorityAlert
)

func (p TrafficPriority) String() string {
	switch p {
	case PriorityDiagnostics:
		return "diagnostics"
	case PriorityTelemetry:
		return "telemetry"
	case PriorityAlert:
		return "alert"
	default:
		return "unknown"
	}
}

// QoS returns the MQTT QoS level for the TrafficPriority.
func (p TrafficPriority) QoS() byte {
	switch p {
	case PriorityDiagnostics:
		return 0
	case PriorityTelemetry:
		return 1
	case PriorityAlert:
		return 2
	default:
		return 0
	}
}

// TrafficStats contains traffic shaping counters accessible via Stats().
type TrafficStats struct {
	TotalAllowed     uint64    `json:"total_allowed"`
	TotalBlocked     uint64    `json:"total_blocked"`
	TelemetryDropped uint64    `json:"telemetry_dropped"`
	CircuitOpened    uint64    `json:"circuit_opened"`
	LastCircuitOpen  time.Time `json:"last_circuit_open"`
}

// CircuitState represents the circuit breaker states.
type CircuitState int32

const (
	CircuitClosed   CircuitState = iota // normal operation
	CircuitOpen                         // rejecting requests
	CircuitHalfOpen                     // testing recovery
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// circuitBreaker implements a simple state-machine circuit breaker.
//
// Compliance:
//   - IEC 62443-3-3 SR 7.1 — Resource Availability
//   - Приказ ОАЦ №66 п. 7.18.6 — мониторинг и реагирование
type circuitBreaker struct {
	state        CircuitState
	failureCount uint64
	threshold    uint64
	timeout      time.Duration
	resetTime    time.Time
	mu           sync.Mutex
}

func newCircuitBreaker(threshold uint64, timeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		state:     CircuitClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.resetTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
		cb.resetTime = time.Now()
	}
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failureCount = 0
	}
}

func (cb *circuitBreaker) getState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// TrafficShaper manages traffic prioritization, rate limiting, and circuit breaking.
//
// Priority rules:
//   - Diagnostics (QoS 0): Always allowed, checked against circuit breaker
//   - Telemetry   (QoS 1): Rate limited by token bucket + circuit breaker
//   - Alert       (QoS 2): Always allowed, bypasses all restrictions
//
// Compliance:
//   - IEC 62443-3-3 SL-3 (Zone 5 — Edge)
//   - IEC 62443-3-3 SR 7.1: Resource Availability — rate limiting + circuit breaker
//   - Приказ ОАЦ №66 п. 7.18.6: Мониторинг и реагирование
type TrafficShaper struct {
	mu      sync.Mutex
	bucket  *rate.Limiter  // token bucket for telemetry
	breaker *circuitBreaker
	stats   TrafficStats
	logger  *slog.Logger
}

// NewTrafficShaper creates a new TrafficShaper with the given rate and burst.
//
// Parameters:
//   - telemetryRate: max telemetry messages per second
//   - burst:         token bucket burst size
//   - logger:        structured logger
func NewTrafficShaper(telemetryRate float64, burst int, logger *slog.Logger) *TrafficShaper {
	if telemetryRate <= 0 {
		telemetryRate = 1
	}
	if burst <= 0 {
		burst = 5
	}

	ts := &TrafficShaper{
		bucket:  rate.NewLimiter(rate.Limit(telemetryRate), burst),
		breaker: newCircuitBreaker(10, 30*time.Second),
		logger:  logger.With("component", "traffic_shaper"),
	}

	ts.logger.Info("traffic shaper initialized",
		"telemetry_rate", telemetryRate,
		"burst", burst,
		"cb_threshold", 10,
		"cb_timeout", "30s",
	)

	return ts
}

// Allow checks whether a message with the given priority may be published.
// Returns true if the message is allowed, false if rate-limited or circuit-open.
//
// Compliance: IEC 62443-3-3 SR 7.1 — Resource Availability
func (ts *TrafficShaper) Allow(priority TrafficPriority) bool {
	switch priority {
	case PriorityAlert:
		// Alerts bypass rate limiting and circuit breaker entirely.
		atomic.AddUint64(&ts.stats.TotalAllowed, 1)
		return true

	case PriorityDiagnostics:
		// Diagnostics: circuit breaker only.
		if !ts.breaker.allow() {
			atomic.AddUint64(&ts.stats.TotalBlocked, 1)
			ts.logger.Warn("circuit breaker blocked diagnostics message")
			return false
		}
		atomic.AddUint64(&ts.stats.TotalAllowed, 1)
		return true

	case PriorityTelemetry:
		// Telemetry: circuit breaker + rate limit.
		if !ts.breaker.allow() {
			atomic.AddUint64(&ts.stats.TelemetryDropped, 1)
			atomic.AddUint64(&ts.stats.TotalBlocked, 1)
			ts.logger.Warn("circuit breaker blocked telemetry message")
			return false
		}

		if !ts.bucket.Allow() {
			atomic.AddUint64(&ts.stats.TelemetryDropped, 1)
			atomic.AddUint64(&ts.stats.TotalBlocked, 1)
			ts.logger.Debug("telemetry rate limited")
			return false
		}

		atomic.AddUint64(&ts.stats.TotalAllowed, 1)
		return true

	default:
		ts.logger.Warn("unknown traffic priority", "priority", priority)
		return false
	}
}

// RecordFailure records a publish failure, which may trip the circuit breaker.
func (ts *TrafficShaper) RecordFailure() {
	ts.breaker.recordFailure()
	atomic.AddUint64(&ts.stats.CircuitOpened, 1)
	ts.stats.LastCircuitOpen = time.Now()

	ts.logger.Warn("circuit breaker failure recorded",
		"state", ts.breaker.getState(),
	)
}

// RecordSuccess records a successful publish, helping close the circuit breaker.
func (ts *TrafficShaper) RecordSuccess() {
	ts.breaker.recordSuccess()
}

// Stats returns a snapshot of current traffic shaping statistics.
func (ts *TrafficShaper) Stats() TrafficStats {
	return TrafficStats{
		TotalAllowed:     atomic.LoadUint64(&ts.stats.TotalAllowed),
		TotalBlocked:     atomic.LoadUint64(&ts.stats.TotalBlocked),
		TelemetryDropped: atomic.LoadUint64(&ts.stats.TelemetryDropped),
		CircuitOpened:    atomic.LoadUint64(&ts.stats.CircuitOpened),
		LastCircuitOpen:  ts.stats.LastCircuitOpen,
	}
}

// ResetStats resets all traffic shaping counters to zero.
func (ts *TrafficShaper) ResetStats() {
	atomic.StoreUint64(&ts.stats.TotalAllowed, 0)
	atomic.StoreUint64(&ts.stats.TotalBlocked, 0)
	atomic.StoreUint64(&ts.stats.TelemetryDropped, 0)
	atomic.StoreUint64(&ts.stats.CircuitOpened, 0)
	ts.stats.LastCircuitOpen = time.Time{}
}

// CircuitState returns the current circuit breaker state.
func (ts *TrafficShaper) CircuitState() CircuitState {
	return ts.breaker.getState()
}

// SetRateLimit dynamically updates the telemetry rate limiter.
func (ts *TrafficShaper) SetRateLimit(rps float64, burst int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.bucket = rate.NewLimiter(rate.Limit(rps), burst)
	ts.logger.Info("rate limit updated", "rps", rps, "burst", burst)
}
