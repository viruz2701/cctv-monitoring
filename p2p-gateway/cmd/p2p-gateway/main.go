package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

	dm := NewDeviceManager(cfg.ProxyBaseRTSPPort)

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

	router := NewRouter(dm, cfg)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ═══════════════════════════════════════════════════════════════════
	// mTLS Configuration (Приказ ОАЦ №66 п. 7.18.2)
	// ═══════════════════════════════════════════════════════════════════
	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13, // TLS 1.3 mandatory
		}

		// Загрузка серверного сертификата
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			log.Fatalf("Failed to load TLS cert/key: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		// Настройка mTLS (mutual TLS)
		if cfg.MTLSRequired && cfg.TLSClientCA != "" {
			caCert, err := os.ReadFile(cfg.TLSClientCA)
			if err != nil {
				log.Fatalf("Failed to load client CA: %v", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			log.Println("mTLS enabled: client certificate required")
		} else if cfg.MTLSRequired {
			log.Fatal("mTLS required but TLSClientCA not set")
		}

		srv.TLSConfig = tlsConfig
	}

	// Канал для сигналов ОС
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	go func() {
		proto := "HTTP"
		if cfg.TLSEnabled {
			proto = "HTTPS (mTLS)"
		}
		log.Printf("P2P gateway listening on %s [%s]", cfg.ListenAddr, proto)

		var err error
		if cfg.TLSEnabled {
			err = srv.ListenAndServeTLS("", "") // Используем cert из TLSConfig
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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
