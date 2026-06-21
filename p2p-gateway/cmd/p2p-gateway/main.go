package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
	}

	// Канал для сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	go func() {
		log.Printf("P2P gateway listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Ожидание сигнала
	sig := <-quit
	log.Printf("Received signal %v, starting graceful shutdown...", sig)

	// Даём 10 секунд на завершение активных соединений
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}

	// Останавливаем все P2P-устройства
	log.Println("Stopping all P2P devices...")
	errs := dm.ShutdownAll()
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("shutdown device error: %v", e)
		}
	} else {
		log.Println("All P2P devices stopped")
	}

	log.Println("P2P gateway shutdown complete")
}
