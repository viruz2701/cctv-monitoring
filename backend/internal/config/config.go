package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Debug            bool
	SIPPort          int
	SIPHost          string
	APIAddr          string
	WorkerCount      int
	ReaperInterval   time.Duration
	HeartbeatTimeout time.Duration
	ImagesDir        string

	// Настройки логирования
	LogFile       string `mapstructure:"log_file"`
	LogMaxSizeMB  int    `mapstructure:"log_max_size_mb"`
	LogMaxBackups int    `mapstructure:"log_max_backups"`
	LogMaxAgeDays int    `mapstructure:"log_max_age_days"`
	LogCompress   bool   `mapstructure:"log_compress"`
	LogServerPort int    `mapstructure:"log_server_port"`

	P2PGatewayURL string `mapstructure:"p2p_gateway_url"`
	P2PAPIKey     string `mapstructure:"p2p_api_key"`

	// CMMS Adapter configuration
	CMMSAdapter       string `mapstructure:"cmms_adapter"` // "internal" (default) | "atlas"
	AtlasURL          string `mapstructure:"atlas_url"`
	AtlasAPIKey       string `mapstructure:"atlas_api_key"`
	AtlasClientID     string `mapstructure:"atlas_client_id"`
	AtlasClientSecret string `mapstructure:"atlas_client_secret"`
	AtlasTokenURL     string `mapstructure:"atlas_token_url"`
	AtlasFallbackDir  string `mapstructure:"atlas_fallback_dir"`

	// Новые настройки для HTTP-приёма событий
	HTTPXMLEnabled  bool `mapstructure:"http_xml_enabled"`
	VigiEnabled     bool `mapstructure:"vigi_enabled"`
	SaveEventImages bool `mapstructure:"save_event_images"`

	// Протоколы
	Dahua     DahuaConfig
	Hisilicon HisiliconConfig
	TVT       TVTConfig
	FTP       FTPConfig
	Hikvision HikvisionConfig
	SNMP      SNMPConfig
	GB28181   GB28181Config // ДОБАВЛЕНО: GB28181 конфигурация

	// Telegram bot configuration
	Telegram TelegramConfig
}

// TelegramConfig — настройки Telegram бота
type TelegramConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
}

// GB28181Config — настройки GB/T 28181 сервера
type GB28181Config struct {
	Enabled           bool   `mapstructure:"enabled"`
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	ServerID          string `mapstructure:"server_id"`
	ServerIP          string `mapstructure:"server_ip"`
	Realm             string `mapstructure:"realm"`
	AuthEnabled       bool   `mapstructure:"auth_enabled"`
	AuthUser          string `mapstructure:"auth_user"`
	AuthPassword      string `mapstructure:"auth_password"`
	AutoCatalog       bool   `mapstructure:"auto_catalog"`
	AutoDeviceInfo    bool   `mapstructure:"auto_device_info"`
	KeepaliveInterval int    `mapstructure:"keepalive_interval"`
	KeepaliveTimeout  int    `mapstructure:"keepalive_timeout"`
	MaxSubChannels    int    `mapstructure:"max_sub_channels"`
	LogSIPMessages    bool   `mapstructure:"log_sip_messages"`
}

type DahuaConfig struct {
	Enabled bool  `mapstructure:"enabled"`
	Ports   []int `mapstructure:"ports"`
}

type HisiliconConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type TVTConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type FTPConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Port       int    `mapstructure:"port"`
	RootPath   string `mapstructure:"rootPath"`
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	AllowFiles bool   `mapstructure:"allowFiles"`
}

type HikvisionConfig struct {
	Enabled bool                       `mapstructure:"enabled"`
	Cameras map[string]HikCameraConfig `mapstructure:"cams"`
}

type HikCameraConfig struct {
	Address  string `mapstructure:"address"`
	HTTPS    bool   `mapstructure:"https"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	RawTCP   bool   `mapstructure:"rawTcp"`
}

type SNMPConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Port            int    `mapstructure:"port"`
	Community       string `mapstructure:"community"`
	Version         string `mapstructure:"version"`
	User            string `mapstructure:"user"`
	AuthProtocol    string `mapstructure:"authProtocol"`
	AuthPassword    string `mapstructure:"authPassword"`
	PrivProtocol    string `mapstructure:"privProtocol"`
	PrivPassword    string `mapstructure:"privPassword"`
	EngineID        string `mapstructure:"engineID"`
	ContextEngineID string `mapstructure:"contextEngineID"`
	ContextName     string `mapstructure:"contextName"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/etc/gb-telemetry/")

	// Значения по умолчанию
	viper.SetDefault("debug", false)
	viper.SetDefault("sip_port", 5060)
	viper.SetDefault("sip_host", "0.0.0.0")
	viper.SetDefault("api_addr", ":8080")
	viper.SetDefault("worker_count", 100)
	viper.SetDefault("reaper_interval", "15s")
	viper.SetDefault("heartbeat_timeout", "65s")
	viper.SetDefault("images_dir", "/var/lib/gb-telemetry/images")
	viper.SetDefault("log_file", "/var/log/gb-telemetry/collector.log")
	viper.SetDefault("log_max_size_mb", 50)
	viper.SetDefault("log_max_backups", 7)
	viper.SetDefault("log_max_age_days", 30)
	viper.SetDefault("log_compress", true)
	viper.SetDefault("log_server_port", 515)

	// CMMS Atlas defaults
	viper.SetDefault("cmms_adapter", "internal")
	viper.SetDefault("atlas_fallback_dir", "/var/lib/gb-telemetry/fallback")

	// Новые настройки
	viper.SetDefault("http_xml_enabled", true)
	viper.SetDefault("vigi_enabled", true)
	viper.SetDefault("save_event_images", true)

	viper.SetDefault("dahua.enabled", true)
	viper.SetDefault("dahua.ports", []int{37777, 37778})
	viper.SetDefault("hisilicon.enabled", true)
	viper.SetDefault("hisilicon.port", 15002)
	viper.SetDefault("tvt.enabled", true)
	viper.SetDefault("tvt.port", 15003)
	viper.SetDefault("ftp.enabled", false)
	viper.SetDefault("ftp.port", 2121)
	viper.SetDefault("ftp.rootPath", "./ftp")
	viper.SetDefault("ftp.user", "alarm")
	viper.SetDefault("ftp.password", "alarm_pass")
	viper.SetDefault("ftp.allowFiles", true)
	viper.SetDefault("hikvision.enabled", false)
	viper.SetDefault("snmp.enabled", false)
	viper.SetDefault("snmp.port", 162)
	viper.SetDefault("snmp.community", "public")
	viper.SetDefault("snmp.version", "v2c")

	// GB28181 defaults
	viper.SetDefault("gb28181.enabled", true)
	viper.SetDefault("gb28181.host", "0.0.0.0")
	viper.SetDefault("gb28181.port", 5060)
	viper.SetDefault("gb28181.server_id", "34020000002000000001")
	viper.SetDefault("gb28181.server_ip", "")
	viper.SetDefault("gb28181.realm", "3402000000")
	viper.SetDefault("gb28181.auth_enabled", false)
	viper.SetDefault("gb28181.auth_user", "admin")
	viper.SetDefault("gb28181.auth_password", "")
	viper.SetDefault("gb28181.auto_catalog", true)
	viper.SetDefault("gb28181.auto_device_info", true)
	viper.SetDefault("gb28181.keepalive_interval", 60)
	viper.SetDefault("gb28181.keepalive_timeout", 180)
	viper.SetDefault("gb28181.max_sub_channels", 64)
	viper.SetDefault("gb28181.log_sip_messages", false)

	bindEnv("debug", "GB_DEBUG")
	bindEnv("sip_port", "GB_SIP_PORT")
	bindEnv("sip_host", "GB_SIP_HOST")
	bindEnv("api_addr", "GB_API_ADDR")
	bindEnv("worker_count", "GB_WORKER_COUNT")
	bindEnv("reaper_interval", "GB_REAPER_INTERVAL")
	bindEnv("heartbeat_timeout", "GB_HEARTBEAT_TIMEOUT")
	bindEnv("images_dir", "GB_IMAGES_DIR")
	bindEnv("log_file", "GB_LOG_FILE")
	bindEnv("log_max_size_mb", "GB_LOG_MAX_SIZE_MB")
	bindEnv("log_max_backups", "GB_LOG_MAX_BACKUPS")
	bindEnv("log_max_age_days", "GB_LOG_MAX_AGE_DAYS")
	bindEnv("log_compress", "GB_LOG_COMPRESS")
	bindEnv("log_server_port", "GB_LOG_SERVER_PORT")
	bindEnv("http_xml_enabled", "GB_HTTP_XML_ENABLED")
	bindEnv("vigi_enabled", "GB_VIGI_ENABLED")
	bindEnv("save_event_images", "GB_SAVE_EVENT_IMAGES")
	bindEnv("dahua.enabled", "GB_DAHUA_ENABLED")
	bindEnv("hisilicon.enabled", "GB_HISILICON_ENABLED")
	bindEnv("hisilicon.port", "GB_HISILICON_PORT")
	bindEnv("tvt.enabled", "GB_TVT_ENABLED")
	bindEnv("tvt.port", "GB_TVT_PORT")
	bindEnv("ftp.enabled", "GB_FTP_ENABLED")
	bindEnv("ftp.port", "GB_FTP_PORT")
	bindEnv("ftp.rootPath", "GB_FTP_ROOT")
	bindEnv("ftp.user", "GB_FTP_USER")
	bindEnv("ftp.password", "GB_FTP_PASSWORD")
	bindEnv("ftp.allowFiles", "GB_FTP_ALLOW_FILES")
	bindEnv("hikvision.enabled", "GB_HIKVISION_ENABLED")
	bindEnv("snmp.enabled", "GB_SNMP_ENABLED")
	bindEnv("snmp.port", "GB_SNMP_PORT")
	bindEnv("snmp.community", "GB_SNMP_COMMUNITY")
	bindEnv("snmp.version", "GB_SNMP_VERSION")

	// GB28181 env bindings
	bindEnv("gb28181.enabled", "GB_GB28181_ENABLED")
	bindEnv("gb28181.host", "GB_GB28181_HOST")
	bindEnv("gb28181.port", "GB_GB28181_PORT")
	bindEnv("gb28181.server_id", "GB_GB28181_SERVER_ID")
	bindEnv("gb28181.server_ip", "GB_GB28181_SERVER_IP")
	bindEnv("gb28181.realm", "GB_GB28181_REALM")
	bindEnv("gb28181.auth_enabled", "GB_GB28181_AUTH_ENABLED")
	bindEnv("gb28181.auth_user", "GB_GB28181_AUTH_USER")
	bindEnv("gb28181.auth_password", "GB_GB28181_AUTH_PASSWORD")
	bindEnv("gb28181.auto_catalog", "GB_GB28181_AUTO_CATALOG")
	bindEnv("gb28181.auto_device_info", "GB_GB28181_AUTO_DEVICE_INFO")
	bindEnv("gb28181.keepalive_interval", "GB_GB28181_KEEPALIVE_INTERVAL")
	bindEnv("gb28181.keepalive_timeout", "GB_GB28181_KEEPALIVE_TIMEOUT")
	bindEnv("gb28181.max_sub_channels", "GB_GB28181_MAX_SUB_CHANNELS")
	bindEnv("gb28181.log_sip_messages", "GB_GB28181_LOG_SIP_MESSAGES")

	// P2P Gateway
	bindEnv("p2p_gateway_url", "GB_P2P_GATEWAY_URL")
	bindEnv("p2p_api_key", "GB_P2P_API_KEY")

	// CMMS Adapter
	bindEnv("cmms_adapter", "GB_CMMS_ADAPTER")
	bindEnv("atlas_url", "GB_ATLAS_URL")
	bindEnv("atlas_api_key", "GB_ATLAS_API_KEY")
	bindEnv("atlas_client_id", "GB_ATLAS_CLIENT_ID")
	bindEnv("atlas_client_secret", "GB_ATLAS_CLIENT_SECRET")
	bindEnv("atlas_token_url", "GB_ATLAS_TOKEN_URL")
	bindEnv("atlas_fallback_dir", "GB_ATLAS_FALLBACK_DIR")

	// Telegram
	bindEnv("telegram.enabled", "GB_TELEGRAM_ENABLED")
	bindEnv("telegram.token", "GB_TELEGRAM_TOKEN")

	// Database
	bindEnv("database.url", "DATABASE_URL")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults and environment variables")
		} else {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}

	cfg := &Config{
		Debug:            viper.GetBool("debug"),
		SIPPort:          viper.GetInt("sip_port"),
		SIPHost:          viper.GetString("sip_host"),
		APIAddr:          viper.GetString("api_addr"),
		WorkerCount:      viper.GetInt("worker_count"),
		ReaperInterval:   viper.GetDuration("reaper_interval"),
		HeartbeatTimeout: viper.GetDuration("heartbeat_timeout"),
		ImagesDir:        viper.GetString("images_dir"),
		LogServerPort:    viper.GetInt("log_server_port"),
		LogFile:          viper.GetString("log_file"),
		LogMaxSizeMB:     viper.GetInt("log_max_size_mb"),
		LogMaxBackups:    viper.GetInt("log_max_backups"),
		LogMaxAgeDays:    viper.GetInt("log_max_age_days"),
		LogCompress:      viper.GetBool("log_compress"),
		HTTPXMLEnabled:   viper.GetBool("http_xml_enabled"),
		VigiEnabled:      viper.GetBool("vigi_enabled"),
		SaveEventImages:  viper.GetBool("save_event_images"),
		Dahua: DahuaConfig{
			Enabled: viper.GetBool("dahua.enabled"),
			Ports:   viper.GetIntSlice("dahua.ports"),
		},
		Hisilicon: HisiliconConfig{
			Enabled: viper.GetBool("hisilicon.enabled"),
			Port:    viper.GetInt("hisilicon.port"),
		},
		TVT: TVTConfig{
			Enabled: viper.GetBool("tvt.enabled"),
			Port:    viper.GetInt("tvt.port"),
		},
		FTP: FTPConfig{
			Enabled:    viper.GetBool("ftp.enabled"),
			Port:       viper.GetInt("ftp.port"),
			RootPath:   viper.GetString("ftp.rootPath"),
			User:       viper.GetString("ftp.user"),
			Password:   viper.GetString("ftp.password"),
			AllowFiles: viper.GetBool("ftp.allowFiles"),
		},
		Hikvision: HikvisionConfig{
			Enabled: viper.GetBool("hikvision.enabled"),
			Cameras: make(map[string]HikCameraConfig),
		},
		SNMP: SNMPConfig{
			Enabled:         viper.GetBool("snmp.enabled"),
			Port:            viper.GetInt("snmp.port"),
			Community:       viper.GetString("snmp.community"),
			Version:         viper.GetString("snmp.version"),
			User:            viper.GetString("snmp.user"),
			AuthProtocol:    viper.GetString("snmp.authProtocol"),
			AuthPassword:    viper.GetString("snmp.authPassword"),
			PrivProtocol:    viper.GetString("snmp.privProtocol"),
			PrivPassword:    viper.GetString("snmp.privPassword"),
			EngineID:        viper.GetString("snmp.engineID"),
			ContextEngineID: viper.GetString("snmp.contextEngineID"),
			ContextName:     viper.GetString("snmp.contextName"),
		},
		GB28181: GB28181Config{
			Enabled:           viper.GetBool("gb28181.enabled"),
			Host:              viper.GetString("gb28181.host"),
			Port:              viper.GetInt("gb28181.port"),
			ServerID:          viper.GetString("gb28181.server_id"),
			ServerIP:          viper.GetString("gb28181.server_ip"),
			Realm:             viper.GetString("gb28181.realm"),
			AuthEnabled:       viper.GetBool("gb28181.auth_enabled"),
			AuthUser:          viper.GetString("gb28181.auth_user"),
			AuthPassword:      viper.GetString("gb28181.auth_password"),
			AutoCatalog:       viper.GetBool("gb28181.auto_catalog"),
			AutoDeviceInfo:    viper.GetBool("gb28181.auto_device_info"),
			KeepaliveInterval: viper.GetInt("gb28181.keepalive_interval"),
			KeepaliveTimeout:  viper.GetInt("gb28181.keepalive_timeout"),
			MaxSubChannels:    viper.GetInt("gb28181.max_sub_channels"),
			LogSIPMessages:    viper.GetBool("gb28181.log_sip_messages"),
		},
		P2PGatewayURL:     viper.GetString("p2p_gateway_url"),
		P2PAPIKey:         viper.GetString("p2p_api_key"),
		CMMSAdapter:       viper.GetString("cmms_adapter"),
		AtlasURL:          viper.GetString("atlas_url"),
		AtlasAPIKey:       viper.GetString("atlas_api_key"),
		AtlasClientID:     viper.GetString("atlas_client_id"),
		AtlasClientSecret: viper.GetString("atlas_client_secret"),
		AtlasTokenURL:     viper.GetString("atlas_token_url"),
		AtlasFallbackDir:  viper.GetString("atlas_fallback_dir"),
		Telegram: TelegramConfig{
			Enabled: viper.GetBool("telegram.enabled"),
			Token:   viper.GetString("telegram.token"),
		},
	}

	if cfg.Hikvision.Enabled {
		camsRaw := viper.Get("hikvision.cams")
		if camsRaw != nil {
			if camsMap, ok := camsRaw.(map[string]interface{}); ok {
				for name, camIntf := range camsMap {
					camMap, ok := camIntf.(map[string]interface{})
					if !ok {
						continue
					}
					camCfg := HikCameraConfig{}
					if v, ok := camMap["address"].(string); ok {
						camCfg.Address = v
					}
					if v, ok := camMap["https"].(bool); ok {
						camCfg.HTTPS = v
					}
					if v, ok := camMap["username"].(string); ok {
						camCfg.Username = v
					}
					if v, ok := camMap["password"].(string); ok {
						camCfg.Password = v
					}
					if v, ok := camMap["rawTcp"].(bool); ok {
						camCfg.RawTCP = v
					}
					if camCfg.Address != "" && camCfg.Username != "" {
						cfg.Hikvision.Cameras[name] = camCfg
					}
				}
			}
		}
	}

	return cfg
}

func bindEnv(key, envVar string) {
	if envVar != "" {
		_ = viper.BindEnv(key, envVar)
	}
}

// Вспомогательные функции
func getEnvString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
