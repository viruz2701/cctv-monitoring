package agent

import (
	"context"
	"edge-agent/internal/wireguard"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// CommandType represents the type of command from Backend.
type CommandType string

const (
	CommandReboot      CommandType = "reboot"
	CommandSyncNow     CommandType = "sync_now"
	CommandDiscover    CommandType = "discover"
	CommandExec        CommandType = "exec"
	CommandGetStatus   CommandType = "get_status"
	CommandUpdateCache CommandType = "update_cache"

	// EDGE-08: WireGuard On-Demand VPN commands
	CommandStartVPNSession CommandType = "start_vpn_session"
	CommandStopVPNSession  CommandType = "stop_vpn_session"
)

// Command represents a command message from Backend via MQTT.
type Command struct {
	// ID is the unique command identifier
	ID string `json:"id"`
	// Type is the command type
	Type CommandType `json:"type"`
	// Target is the target device identifier (optional)
	Target string `json:"target,omitempty"`
	// Payload is command-specific data
	Payload json.RawMessage `json:"payload,omitempty"`
	// Timestamp of command creation
	Timestamp time.Time `json:"timestamp"`
	// ResponseTopic for command response
	ResponseTopic string `json:"response_topic,omitempty"`
}

// CommandResult represents the result of command execution.
type CommandResult struct {
	// ID matches the command ID
	ID string `json:"id"`
	// Status is the execution status
	Status string `json:"status"`
	// Error message if any
	Error string `json:"error,omitempty"`
	// Data is the result data
	Data interface{} `json:"data,omitempty"`
	// Timestamp of completion
	Timestamp time.Time `json:"timestamp"`
}

// CommandHandler processes MQTT commands from Backend.
//
// Compliance: OWASP ASVS L3 — input validation, error handling
//
//	Приказ ОАЦ №66 п. 7.18.4 — управление конечными узлами
type CommandHandler struct {
	agent     *Agent
	wgManager *wireguard.WireGuardManager
	logger    *slog.Logger
}

// NewCommandHandler creates a new command handler.
// Если wgManager == nil, VPN команды будут возвращать ошибку.
func NewCommandHandler(agent *Agent, wgManager *wireguard.WireGuardManager, logger *slog.Logger) *CommandHandler {
	return &CommandHandler{
		agent:     agent,
		wgManager: wgManager,
		logger:    logger.With("component", "command_handler"),
	}
}

// HandleMessage processes incoming MQTT messages.
// Implements mqtt.MessageHandler interface.
func (h *CommandHandler) HandleMessage(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()

	h.logger.Debug("received MQTT message",
		"topic", topic,
		"payload_size", len(payload),
	)

	// Parse command
	var cmd Command
	if err := json.Unmarshal(payload, &cmd); err != nil {
		h.logger.Error("failed to parse command",
			"error", err,
			"payload", string(payload),
		)
		return
	}

	// Validate command
	if err := h.validateCommand(&cmd); err != nil {
		h.logger.Error("invalid command",
			"error", err,
			"command_id", cmd.ID,
		)
		h.sendError(cmd, "invalid_command: "+err.Error())
		return
	}

	// Execute command
	h.logger.Info("executing command",
		"command_id", cmd.ID,
		"type", cmd.Type,
		"target", cmd.Target,
	)

	result := h.execute(cmd)

	// Send result
	if cmd.ResponseTopic != "" {
		h.sendResult(cmd.ResponseTopic, result)
	}
}

// validateCommand validates the command structure.
// OWASP ASVS L3: Input validation on all fields.
func (h *CommandHandler) validateCommand(cmd *Command) error {
	if cmd.ID == "" {
		return fmt.Errorf("command ID is required")
	}

	if cmd.Type == "" {
		return fmt.Errorf("command type is required")
	}

	// Validate command type
	validTypes := map[CommandType]bool{
		CommandReboot:          true,
		CommandSyncNow:         true,
		CommandDiscover:        true,
		CommandExec:            true,
		CommandGetStatus:       true,
		CommandUpdateCache:     true,
		CommandStartVPNSession: true,
		CommandStopVPNSession:  true,
	}

	if !validTypes[cmd.Type] {
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}

	// If target is specified, validate format (IP or MAC)
	if cmd.Target != "" {
		if !strings.Contains(cmd.Target, ":") && !strings.Contains(cmd.Target, ".") {
			return fmt.Errorf("invalid target format: %s", cmd.Target)
		}
	}

	return nil
}

// execute runs the appropriate command handler based on type.
func (h *CommandHandler) execute(cmd Command) CommandResult {
	switch cmd.Type {
	case CommandReboot:
		return h.handleReboot(cmd)
	case CommandSyncNow:
		return h.handleSyncNow(cmd)
	case CommandDiscover:
		return h.handleDiscover(cmd)
	case CommandExec:
		return h.handleExec(cmd)
	case CommandGetStatus:
		return h.handleGetStatus(cmd)
	case CommandUpdateCache:
		return h.handleUpdateCache(cmd)
	case CommandStartVPNSession:
		return h.handleStartVPNSession(cmd)
	case CommandStopVPNSession:
		return h.handleStopVPNSession(cmd)
	default:
		return CommandResult{
			ID:        cmd.ID,
			Status:    "error",
			Error:     fmt.Sprintf("unsupported command type: %s", cmd.Type),
			Timestamp: time.Now(),
		}
	}
}

func (h *CommandHandler) handleReboot(cmd Command) CommandResult {
	h.logger.Warn("reboot command received", "command_id", cmd.ID)

	// Schedule reboot after responding
	go func() {
		time.Sleep(2 * time.Second)
		// In production, this would trigger actual system reboot
		h.logger.Info("system reboot initiated")
	}()

	return CommandResult{
		ID:        cmd.ID,
		Status:    "ok",
		Data:      map[string]string{"message": "reboot scheduled"},
		Timestamp: time.Now(),
	}
}

func (h *CommandHandler) handleSyncNow(cmd Command) CommandResult {
	go h.agent.runSync()

	return CommandResult{
		ID:        cmd.ID,
		Status:    "ok",
		Data:      map[string]string{"message": "sync started"},
		Timestamp: time.Now(),
	}
}

func (h *CommandHandler) handleDiscover(cmd Command) CommandResult {
	go h.agent.runDiscovery()

	return CommandResult{
		ID:        cmd.ID,
		Status:    "ok",
		Data:      map[string]string{"message": "discovery started"},
		Timestamp: time.Now(),
	}
}

func (h *CommandHandler) handleExec(cmd Command) CommandResult {
	// Universal Protocol Interpreter execution
	// In production, this would route to the appropriate protocol adapter
	h.logger.Info("exec command",
		"target", cmd.Target,
		"payload", string(cmd.Payload),
	)

	return CommandResult{
		ID:     cmd.ID,
		Status: "ok",
		Data: map[string]interface{}{
			"message": "command forwarded to protocol interpreter",
			"target":  cmd.Target,
		},
		Timestamp: time.Now(),
	}
}

func (h *CommandHandler) handleGetStatus(cmd Command) CommandResult {
	h.agent.devicesMu.RLock()
	deviceCount := len(h.agent.devices)
	h.agent.devicesMu.RUnlock()

	return CommandResult{
		ID:     cmd.ID,
		Status: "ok",
		Data: map[string]interface{}{
			"agent_id":       h.agent.config.AgentID,
			"version":        h.agent.config.Version,
			"uptime":         time.Now().Unix(),
			"device_count":   deviceCount,
			"cache_count":    h.agent.protoCache.Count(),
			"mqtt_connected": h.agent.mqttClient.IsConnected(),
		},
		Timestamp: time.Now(),
	}
}

func (h *CommandHandler) handleUpdateCache(cmd Command) CommandResult {
	go h.agent.runSync()

	return CommandResult{
		ID:        cmd.ID,
		Status:    "ok",
		Data:      map[string]string{"message": "cache update triggered"},
		Timestamp: time.Now(),
	}
}

// handleStartVPNSession запускает WireGuard туннель на агенте.
//
// Ожидает payload с TunnelConfig в JSON.
//
// Compliance:
//   - IEC 62443-3-3 SL-4: Edge device security
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
func (h *CommandHandler) handleStartVPNSession(cmd Command) CommandResult {
	h.logger.Info("start_vpn_session command received",
		"command_id", cmd.ID,
	)

	if h.wgManager == nil {
		return CommandResult{
			ID:        cmd.ID,
			Status:    "error",
			Error:     "WireGuard manager not configured",
			Timestamp: time.Now(),
		}
	}

	// Парсим конфигурацию туннеля из payload
	var config struct {
		SessionID   string `json:"session_id"`
		DurationSec int    `json:"duration_sec"`
	}
	if err := json.Unmarshal(cmd.Payload, &config); err != nil {
		return CommandResult{
			ID:        cmd.ID,
			Status:    "error",
			Error:     "invalid vpn config: " + err.Error(),
			Timestamp: time.Now(),
		}
	}

	h.logger.Info("starting vpn tunnel",
		"session_id", config.SessionID,
		"duration_sec", config.DurationSec,
	)

	// Запускаем туннель в фоне с авто-остановкой по таймеру
	go func() {
		ctx := context.Background()

		// Здесь конфигурация туннеля должна быть получена от Backend
		// через отдельный запрос или передана в payload команды
		tunnelConfig := wireguard.DefaultTunnelConfig()
		tunnelConfig.Duration = config.DurationSec

		if err := h.wgManager.StartTunnel(ctx, tunnelConfig); err != nil {
			h.logger.Error("failed to start vpn tunnel",
				"session_id", config.SessionID,
				"error", err,
			)
			return
		}

		h.logger.Info("vpn tunnel started",
			"session_id", config.SessionID,
			"duration_sec", config.DurationSec,
		)
	}()

	return CommandResult{
		ID:     cmd.ID,
		Status: "ok",
		Data: map[string]interface{}{
			"message":    "vpn tunnel starting",
			"session_id": config.SessionID,
		},
		Timestamp: time.Now(),
	}
}

// handleStopVPNSession останавливает WireGuard туннель на агенте.
//
// Compliance:
//   - IEC 62443-3-3 SR 7.2: Session termination
//   - Приказ ОАЦ №66 п. 7.18.2: Отзыв доступа
func (h *CommandHandler) handleStopVPNSession(cmd Command) CommandResult {
	h.logger.Info("stop_vpn_session command received",
		"command_id", cmd.ID,
	)

	if h.wgManager == nil {
		return CommandResult{
			ID:        cmd.ID,
			Status:    "error",
			Error:     "WireGuard manager not configured",
			Timestamp: time.Now(),
		}
	}

	go func() {
		ctx := context.Background()
		if err := h.wgManager.StopTunnel(ctx); err != nil {
			h.logger.Error("failed to stop vpn tunnel",
				"command_id", cmd.ID,
				"error", err,
			)
		}
	}()

	return CommandResult{
		ID:     cmd.ID,
		Status: "ok",
		Data: map[string]interface{}{
			"message": "vpn tunnel stopping",
		},
		Timestamp: time.Now(),
	}
}

// sendError sends an error response for a command.
func (h *CommandHandler) sendError(cmd Command, errMsg string) {
	result := CommandResult{
		ID:        cmd.ID,
		Status:    "error",
		Error:     errMsg,
		Timestamp: time.Now(),
	}

	if cmd.ResponseTopic != "" {
		h.sendResult(cmd.ResponseTopic, result)
	}
}

// sendResult publishes command result to the response topic.
func (h *CommandHandler) sendResult(topic string, result CommandResult) {
	payload, err := json.Marshal(result)
	if err != nil {
		h.logger.Error("failed to marshal command result", "error", err)
		return
	}

	if !h.agent.mqttClient.IsConnected() {
		h.logger.Warn("MQTT not connected, queuing command result")
		if err := h.agent.offlineQueue.Enqueue(topic, payload); err != nil {
			h.logger.Error("failed to queue command result", "error", err)
		}
		return
	}

	token := h.agent.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		h.logger.Error("failed to publish command result",
			"topic", topic,
			"error", token.Error(),
		)
	}
}
