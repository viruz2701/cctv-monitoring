package main

import (
	"log"
	"net/http"

	"p2p-gateway/pkg/adapters" // убедитесь, что путь соответствует вашей структуре
)

func main() {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dm := NewDeviceManager(cfg)

	// Регистрация адаптера Reolink
	reolinkAdapter := NewReolinkAdapter(cfg.ProxyBinPath)
	dm.RegisterAdapter("reolink", reolinkAdapter)

	// Регистрация адаптера Dahua (если настроен)
	if cfg.DahuaPythonPath != "" && cfg.DahuaScriptPath != "" {
		dahuaAdapter := adapters.NewDahuaAdapter(cfg.DahuaPythonPath, cfg.DahuaScriptPath)
		dm.RegisterAdapter("dahua", dahuaAdapter)
	} else {
		log.Println("Dahua adapter not configured – skipping")
	}

	// Регистрация адаптера Xiongmai (если настроен)
	if cfg.XiongmaiNodePath != "" && cfg.XiongmaiScriptPath != "" {
		xiongmaiAdapter := adapters.NewXiongmaiAdapter(cfg.XiongmaiNodePath, cfg.XiongmaiScriptPath)
		dm.RegisterAdapter("xiongmai", xiongmaiAdapter)
	} else {
		log.Println("Xiongmai adapter not configured – skipping")
	}

	router := NewRouter(dm, cfg)

	log.Printf("p2p-gateway listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatal(err)
	}
}
