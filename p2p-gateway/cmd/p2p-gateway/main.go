package main

import (
	"log"
	"net/http"

	"p2p-gateway/pkg/adapters"
)

func main() {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Printf("Config not found, using defaults: %v", err)
		cfg = &Config{ListenAddr: ":8082"}
	}

	dm := NewDeviceManager(8554)

	// Регистрация адаптеров
	dm.RegisterAdapter("hikvision", adapters.NewHikvisionAdapter())
	dm.RegisterAdapter("reolink", adapters.NewReolinkAdapter(cfg.ProxyBinPath))

	if cfg.DahuaPythonPath != "" && cfg.DahuaScriptPath != "" {
		dm.RegisterAdapter("dahua", adapters.NewDahuaAdapter(cfg.DahuaPythonPath, cfg.DahuaScriptPath))
		log.Println("Dahua adapter registered")
	} else {
		log.Println("Dahua adapter not configured – skipping")
	}

	// Xiongmai временно отключён
	// if cfg.XiongmaiNodePath != "" {
	//     dm.RegisterAdapter("xiongmai", adapters.NewXiongmaiAdapter(cfg.XiongmaiNodePath, cfg.XiongmaiScriptPath))
	// }

	router := NewRouter(dm)
	log.Printf("P2P gateway listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, router))
}
