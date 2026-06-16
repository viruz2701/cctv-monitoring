package main

import (
	"log"
	"net/http"

	"p2p-gateway/pkg/adapters"
	"p2p-gateway/pkg/jftech"
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
	}

	// Регистрация адаптера Xiongmai (Jftech)
	if cfg.Jftech != nil && cfg.Jftech.UUID != "" {
		jfCfg := &jftech.Config{
			UUID:      cfg.Jftech.UUID,
			AppKey:    cfg.Jftech.AppKey,
			AppSecret: cfg.Jftech.AppSecret,
			MoveCard:  cfg.Jftech.MoveCard,
			Endpoint:  cfg.Jftech.Endpoint,
		}
		region := cfg.Jftech.Region
		if region == "" {
			region = "Local"
		}
		dm.RegisterAdapter("xiongmai", adapters.NewXiongmaiAdapter(jfCfg, region))
		dm.RegisterAdapter("jftech", adapters.NewXiongmaiAdapter(jfCfg, region))
		log.Println("Xiongmai (Jftech) adapter registered")
	} else {
		log.Println("Jftech config not found or incomplete – skipping Xiongmai adapter")
	}

	router := NewRouter(dm)
	log.Printf("P2P gateway listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, router))
}
