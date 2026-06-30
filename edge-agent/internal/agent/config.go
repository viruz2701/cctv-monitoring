package agent

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	stdlibtls "crypto/tls"

	"edge-agent/internal/tls"
)

// Config holds all configuration for the Edge Agent.
// All values are loaded from environment variables — no hardcoded secrets.
type Config struct {
	// Agent identity (Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация)
	AgentID string
	Version string

	// LAN settings for device discovery
	LANSubnet    string
	LANInterface string

	// MQTT broker connection (mTLS required — IEC 62443 SL-3)
	MQTTBrokerURL string
	MQTTTopicRoot string
	MQTTTLSConfig *stdlibtls.Config

	// Backend API connection (digest auth)
	BackendURL     string
	BackendUser    string
	BackendTimeout time.Duration

	// Polling intervals
	PollInterval      time.Duration
	SyncInterval      time.Duration
	DiscoveryInterval time.Duration

	// Offline queue
	OfflineQueuePath string

	// Descriptor cache
	CachePath string

	// Log level
	LogLevel string
}

const (
	defaultPollInterval      = 30 * time.Second
	defaultSyncInterval      = 300 * time.Second
	defaultDiscoveryInterval = 600 * time.Second
	defaultBackendTimeout    = 10 * time.Second
	defaultMQTTTopicRoot     = "cctv/edge"
	envPrefix                = "EDGE_AGENT_"
)

// LoadConfig reads configuration from environment variables.
// Returns validated Config or error.
func LoadConfig() (*Config, error) {
	c := &Config{
		AgentID:          os.Getenv(envPrefix + "AGENT_ID"),
		Version:          os.Getenv(envPrefix + "VERSION"),
		LANSubnet:        os.Getenv(envPrefix + "LAN_SUBNET"),
		LANInterface:     os.Getenv(envPrefix + "LAN_INTERFACE"),
		MQTTBrokerURL:    os.Getenv(envPrefix + "MQTT_BROKER_URL"),
		MQTTTopicRoot:    envOrDefault("MQTT_TOPIC_ROOT", defaultMQTTTopicRoot),
		BackendURL:       os.Getenv(envPrefix + "BACKEND_URL"),
		BackendUser:      os.Getenv(envPrefix + "BACKEND_USER"),
		OfflineQueuePath: os.Getenv(envPrefix + "OFFLINE_QUEUE_PATH"),
		CachePath:        os.Getenv(envPrefix + "CACHE_PATH"),
		LogLevel:         os.Getenv(envPrefix + "LOG_LEVEL"),
	}

	// Parse durations
	c.PollInterval = parseDurationEnv("POLL_INTERVAL", defaultPollInterval)
	c.SyncInterval = parseDurationEnv("SYNC_INTERVAL", defaultSyncInterval)
	c.DiscoveryInterval = parseDurationEnv("DISCOVERY_INTERVAL", defaultDiscoveryInterval)
	c.BackendTimeout = parseDurationEnv("BACKEND_TIMEOUT", defaultBackendTimeout)

	// Validate required fields
	var errs []error
	if c.AgentID == "" {
		errs = append(errs, errors.New(envPrefix+"AGENT_ID is required"))
	}
	if c.MQTTBrokerURL == "" {
		errs = append(errs, errors.New(envPrefix+"MQTT_BROKER_URL is required"))
	}
	if c.LANSubnet == "" {
		errs = append(errs, errors.New(envPrefix+"LAN_SUBNET is required"))
	}
	if c.BackendURL == "" {
		errs = append(errs, errors.New(envPrefix+"BACKEND_URL is required"))
	}
	if c.BackendUser == "" {
		errs = append(errs, errors.New(envPrefix+"BACKEND_USER is required"))
	}

	// Validate subnet
	if c.LANSubnet != "" {
		if _, _, err := net.ParseCIDR(c.LANSubnet); err != nil {
			errs = append(errs, fmt.Errorf("invalid LAN_SUBNET %q: %w", c.LANSubnet, err))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// Load mTLS configuration (Приказ ОАЦ №66 п. 7.18.2 — mTLS 1.3)
	mqttCert := os.Getenv(envPrefix + "MQTT_CERT")
	mqttKey := os.Getenv(envPrefix + "MQTT_KEY")
	mqttCA := os.Getenv(envPrefix + "MQTT_CA")

	tlsConfig, err := tls.LoadClientTLS(mqttCert, mqttKey, mqttCA, c.AgentID)
	if err != nil {
		return nil, fmt.Errorf("mTLS config: %w", err)
	}
	c.MQTTTLSConfig = tlsConfig

	// Default log level
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	return c, nil
}

func envOrDefault(key, def string) string {
	val := os.Getenv(envPrefix + key)
	if val == "" {
		return def
	}
	return val
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
	val := os.Getenv(envPrefix + key)
	if val == "" {
		return def
	}
	d, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return time.Duration(d) * time.Second
}
