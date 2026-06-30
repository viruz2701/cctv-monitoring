package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"edge-agent/internal/discovery"
	"edge-agent/internal/protocols"
)

// Agent is the main orchestrator for the Edge Agent.
// It manages MQTT connection, device discovery, protocol sync,
// telemetry polling, and command handling.
//
// Compliance: IEC 62443-3-3 SL-3 (Zone 5 — Edge)
//
//	Приказ ОАЦ №66 п. 7.18 (уникальная идентификация, mTLS)
type Agent struct {
	config *Config
	logger *slog.Logger

	// MQTT client
	mqttClient mqtt.Client

	// Sub-components
	discoverer   *discovery.Orchestrator
	protoCache   *protocols.DescriptorCache
	protoSync    *protocols.ProtocolSync
	cmdHandler   *CommandHandler
	poller       *Poller
	offlineQueue *OfflineQueue

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Device registry (discovered devices)
	devicesMu sync.RWMutex
	devices   []discovery.Device
}

// New creates a new Agent instance.
func New(cfg *Config) (*Agent, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize sub-components
	discOrch := discovery.NewOrchestrator(cfg.LANSubnet, cfg.LANInterface, logger)
	protoCache := protocols.NewDescriptorCache(cfg.CachePath, logger)
	protoSync := protocols.NewProtocolSync(cfg.BackendURL, cfg.BackendUser, cfg.BackendTimeout, logger)
	offQueue := NewOfflineQueue(cfg.OfflineQueuePath, logger)
	poller := NewPoller(cfg.PollInterval, logger)

	a := &Agent{
		config:       cfg,
		logger:       logger,
		discoverer:   discOrch,
		protoCache:   protoCache,
		protoSync:    protoSync,
		poller:       poller,
		offlineQueue: offQueue,
		ctx:          ctx,
		cancel:       cancel,
	}

	// CommandHandler needs reference to Agent for executing commands
	a.cmdHandler = NewCommandHandler(a, logger)

	return a, nil
}

// Run starts the agent and blocks until shutdown.
func (a *Agent) Run() error {
	a.logger.Info("starting edge agent",
		"agent_id", a.config.AgentID,
		"version", a.config.Version,
		"lan_subnet", a.config.LANSubnet,
	)

	// Connect to MQTT broker (mTLS required — IEC 62443 SL-3)
	if err := a.connectMQTT(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	a.logger.Info("connected to MQTT broker", "broker", a.config.MQTTBrokerURL)

	// Load protocol descriptors from cache
	if err := a.protoCache.Load(); err != nil {
		a.logger.Warn("failed to load descriptor cache", "error", err)
	}

	// Start background workers
	a.startWorkers()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCh:
		a.logger.Info("received signal, shutting down", "signal", sig)
	case <-a.ctx.Done():
		a.logger.Info("agent context cancelled")
	}

	a.shutdown()
	return nil
}

// Stop gracefully stops the agent.
func (a *Agent) Stop() {
	a.cancel()
}

// connectMQTT establishes mTLS connection to the MQTT broker.
func (a *Agent) connectMQTT() error {
	opts := mqtt.NewClientOptions().
		AddBroker(a.config.MQTTBrokerURL).
		SetClientID(a.config.AgentID).
		SetTLSConfig(a.config.MQTTTLSConfig).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetConnectionLostHandler(a.onConnectionLost).
		SetOnConnectHandler(a.onConnected).
		SetOrderMatters(false)

	// Subscribe to command topic (Приказ ОАЦ №66 п. 7.18.4 — управление)
	cmdTopic := fmt.Sprintf("%s/%s/commands/#", a.config.MQTTTopicRoot, a.config.AgentID)
	opts.SetDefaultPublishHandler(a.cmdHandler.HandleMessage)

	a.mqttClient = mqtt.NewClient(opts)
	if token := a.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	// Subscribe to command topic
	if token := a.mqttClient.Subscribe(cmdTopic, 1, nil); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe to %s: %w", cmdTopic, token.Error())
	}

	return nil
}

func (a *Agent) onConnected(c mqtt.Client) {
	a.logger.Info("connected to MQTT broker")

	// Flush offline queue on reconnect
	go a.flushOfflineQueue()
}

func (a *Agent) onConnectionLost(c mqtt.Client, err error) {
	a.logger.Warn("MQTT connection lost", "error", err)
}

// startWorkers launches background goroutines.
func (a *Agent) startWorkers() {
	// Device discovery loop
	a.wg.Add(1)
	go a.discoveryLoop()

	// Protocol sync loop
	a.wg.Add(1)
	go a.syncLoop()

	// Telemetry polling loop
	a.wg.Add(1)
	go a.pollingLoop()

	a.logger.Info("background workers started")
}

// discoveryLoop periodically discovers devices on the LAN.
func (a *Agent) discoveryLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.config.DiscoveryInterval)
	defer ticker.Stop()

	// Run initial discovery
	a.runDiscovery()

	for {
		select {
		case <-ticker.C:
			a.runDiscovery()
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Agent) runDiscovery() {
	a.logger.Info("starting device discovery")

	devices, err := a.discoverer.Scan(a.ctx)
	if err != nil {
		a.logger.Error("device discovery failed", "error", err)
		return
	}

	a.devicesMu.Lock()
	a.devices = devices
	a.devicesMu.Unlock()

	a.logger.Info("device discovery completed", "count", len(devices))
	for _, d := range devices {
		a.logger.Debug("discovered device",
			"ip", d.IP,
			"mac", d.MAC,
			"vendor", d.Vendor,
			"type", d.DeviceType,
		)
	}
}

// syncLoop periodically syncs protocol descriptors with Backend.
func (a *Agent) syncLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.config.SyncInterval)
	defer ticker.Stop()

	// Run initial sync
	a.runSync()

	for {
		select {
		case <-ticker.C:
			a.runSync()
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Agent) runSync() {
	a.logger.Info("starting protocol sync")

	descriptors, err := a.protoSync.Sync(a.ctx)
	if err != nil {
		a.logger.Error("protocol sync failed", "error", err)
		return
	}

	if err := a.protoCache.Update(descriptors); err != nil {
		a.logger.Error("failed to update descriptor cache", "error", err)
		return
	}

	a.logger.Info("protocol sync completed", "count", len(descriptors))
}

// pollingLoop periodically collects telemetry from discovered devices.
func (a *Agent) pollingLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.runPolling()
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Agent) runPolling() {
	a.devicesMu.RLock()
	devices := make([]discovery.Device, len(a.devices))
	copy(devices, a.devices)
	a.devicesMu.RUnlock()

	if len(devices) == 0 {
		return
	}

	results := a.poller.Collect(a.ctx, devices)
	for _, res := range results {
		a.publishTelemetry(res)
	}
}

// publishTelemetry publishes telemetry data to MQTT.
func (a *Agent) publishTelemetry(data TelemetryResult) {
	topic := fmt.Sprintf("%s/%s/telemetry/%s",
		a.config.MQTTTopicRoot, a.config.AgentID, data.DeviceID)

	payload, err := data.Marshal()
	if err != nil {
		a.logger.Error("failed to marshal telemetry", "error", err)
		return
	}

	if !a.mqttClient.IsConnected() {
		// Queue for later delivery
		if err := a.offlineQueue.Enqueue(topic, payload); err != nil {
			a.logger.Error("failed to enqueue offline message", "error", err)
		}
		return
	}

	token := a.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		a.logger.Error("failed to publish telemetry", "error", token.Error())
	}
}

// publishEvent publishes an event to MQTT.
func (a *Agent) publishEvent(eventType string, payload []byte) {
	topic := fmt.Sprintf("%s/%s/event/%s",
		a.config.MQTTTopicRoot, a.config.AgentID, eventType)

	if !a.mqttClient.IsConnected() {
		if err := a.offlineQueue.Enqueue(topic, payload); err != nil {
			a.logger.Error("failed to enqueue offline event", "error", err)
		}
		return
	}

	token := a.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		a.logger.Error("failed to publish event", "error", token.Error())
	}
}

// flushOfflineQueue replays queued messages after reconnection.
func (a *Agent) flushOfflineQueue() {
	messages, err := a.offlineQueue.DequeueAll()
	if err != nil {
		a.logger.Error("failed to dequeue offline messages", "error", err)
		return
	}

	for _, msg := range messages {
		token := a.mqttClient.Publish(msg.Topic, 1, false, msg.Payload)
		token.Wait()
		if token.Error() != nil {
			a.logger.Error("failed to publish queued message",
				"topic", msg.Topic,
				"error", token.Error(),
			)
			// Re-enqueue failed messages
			if err := a.offlineQueue.Enqueue(msg.Topic, msg.Payload); err != nil {
				a.logger.Error("failed to re-enqueue", "error", err)
			}
		}
	}
}

// shutdown performs graceful shutdown of all components.
func (a *Agent) shutdown() {
	a.logger.Info("shutting down agent")

	// Cancel context to stop all loops
	a.cancel()

	// Wait for goroutines to finish
	a.wg.Wait()

	// Disconnect MQTT
	if a.mqttClient != nil && a.mqttClient.IsConnected() {
		a.mqttClient.Disconnect(1000)
	}

	// Close offline queue
	if err := a.offlineQueue.Close(); err != nil {
		a.logger.Error("failed to close offline queue", "error", err)
	}

	// Save descriptor cache
	if err := a.protoCache.Save(); err != nil {
		a.logger.Error("failed to save descriptor cache", "error", err)
	}

	a.logger.Info("agent shutdown complete")
}

// GetConfig returns the agent configuration.
func (a *Agent) GetConfig() *Config {
	return a.config
}

// GetMQTTClient returns the MQTT client for use by sub-components.
func (a *Agent) GetMQTTClient() mqtt.Client {
	return a.mqttClient
}

// getLogger returns the agent logger.
func (a *Agent) getLogger() *slog.Logger {
	return a.logger
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
