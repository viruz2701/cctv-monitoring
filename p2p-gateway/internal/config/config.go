package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Alarm     AlarmConfig
	Hikvision HikvisionConfig
}

type ServerConfig struct {
	Port   string
	ApiKey string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type AlarmConfig struct {
	CollectorURL string
	ApiKey       string
}

type HikvisionConfig struct {
	Go2rtcBinaryPath string
	Go2rtcApiPort    int
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	viper.AutomaticEnv()

	viper.SetDefault("server.port", "8081")
	viper.SetDefault("server.apiKey", "change-me")
	viper.SetDefault("alarm.collectorURL", "http://localhost:8080")
	viper.SetDefault("alarm.apiKey", "internal-secret")
	viper.SetDefault("hikvision.go2rtcBinaryPath", "./bin/go2rtc")
	viper.SetDefault("hikvision.go2rtcApiPort", 1984)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Config file not found, using defaults: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}
	return &cfg
}
