package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"edge-agent/internal/agent"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("edge-agent version %s\n", Version)
		os.Exit(0)
	}

	// Load configuration from environment variables
	// (Приказ ОАЦ №66 п. 7.18.1 — уникальная идентификация через AGENT_ID)
	cfg, err := agent.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Override version if set in config
	if cfg.Version == "" {
		cfg.Version = Version
	}

	// Create and run the agent
	agt, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create agent: %v\n", err)
		os.Exit(1)
	}

	slog.Info("edge agent starting",
		"agent_id", cfg.AgentID,
		"version", cfg.Version,
		"mqtt_broker", cfg.MQTTBrokerURL,
		"backend", cfg.BackendURL,
		"lan_subnet", cfg.LANSubnet,
	)

	if err := agt.Run(); err != nil {
		slog.Error("agent terminated with error", "error", err)
		os.Exit(1)
	}
}
