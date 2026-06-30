package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"edge-agent/internal/discovery"
)

// TelemetryResult holds telemetry data for a single device.
type TelemetryResult struct {
	// DeviceID is the device MAC or IP
	DeviceID string `json:"device_id"`
	// AgentID is the reporting edge agent
	AgentID string `json:"agent_id"`
	// Timestamp of collection
	Timestamp time.Time `json:"timestamp"`
	// Reachable indicates if device responded
	Reachable bool `json:"reachable"`
	// ResponseTime in milliseconds
	ResponseTimeMs int64 `json:"response_time_ms"`
	// CPU usage (if available)
	CPU float64 `json:"cpu,omitempty"`
	// Memory usage (if available)
	Memory float64 `json:"memory,omitempty"`
	// Uptime in seconds
	Uptime int64 `json:"uptime,omitempty"`
	// Firmware version (if available)
	Firmware string `json:"firmware,omitempty"`
	// Error message if unreachable
	Error string `json:"error,omitempty"`
}

// Marshal serializes TelemetryResult to JSON.
func (t *TelemetryResult) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

// Poller periodically collects health/telemetry from discovered devices.
//
// Compliance: IEC 62443-3-3 SL-3 — мониторинг состояния устройств
type Poller struct {
	interval time.Duration
	logger   *slog.Logger
}

// NewPoller creates a new telemetry poller.
func NewPoller(interval time.Duration, logger *slog.Logger) *Poller {
	return &Poller{
		interval: interval,
		logger:   logger.With("component", "poller"),
	}
}

// Collect gathers telemetry from all discovered devices.
func (p *Poller) Collect(ctx context.Context, devices []discovery.Device) []TelemetryResult {
	if len(devices) == 0 {
		return nil
	}

	results := make([]TelemetryResult, 0, len(devices))

	for _, device := range devices {
		select {
		case <-ctx.Done():
			return results
		default:
		}

		result := p.collectDevice(ctx, device)
		if result != nil {
			results = append(results, *result)
		}
	}

	p.logger.Debug("telemetry collected", "count", len(results))
	return results
}

// collectDevice polls a single device for telemetry data.
func (p *Poller) collectDevice(ctx context.Context, device discovery.Device) *TelemetryResult {
	deviceID := device.MAC.String()
	if deviceID == "" {
		deviceID = device.IP.String()
	}

	result := &TelemetryResult{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
	}

	// Probe common CCTV ports for reachability
	start := time.Now()
	ports := device.Ports
	if len(ports) == 0 {
		ports = []int{80, 554, 8000, 443}
	}

	reachable := false
	for _, port := range ports {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		addr := net.JoinHostPort(device.IP.String(), fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			reachable = true
			result.ResponseTimeMs = time.Since(start).Milliseconds()
			break
		}
	}

	result.Reachable = reachable
	if !reachable {
		result.Error = "device unreachable"
		result.ResponseTimeMs = time.Since(start).Milliseconds()
	}

	return result
}
