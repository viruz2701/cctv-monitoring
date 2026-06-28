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
	CMMSAdapter       string `mapstructure:"cmms_adapter"` // "internal" (default) | "atlas" | "servicenow" | "toir" | "jira"
	AtlasURL          string `mapstructure:"atlas_url"`
	AtlasAPIKey       string `mapstructure:"atlas_api_key"`
	AtlasClientID     string `mapstructure:"atlas_client_id"`
	AtlasClientSecret string `mapstructure:"atlas_client_secret"`
	AtlasTokenURL     string `mapstructure:"atlas_token_url"`
	AtlasFallbackDir  string `mapstructure:"atlas_fallback_dir"`

	// ServiceNow adapter
	ServiceNowInstanceURL  string `mapstructure:"servicenow_instance_url"`
	ServiceNowClientID     string `mapstructure:"servicenow_client_id"`
	ServiceNowClientSecret string `mapstructure:"servicenow_client_secret"`
	ServiceNowTokenURL     string `mapstructure:"servicenow_token_url"`
	ServiceNowUsername     string `mapstructure:"servicenow_username"`
	ServiceNowPassword     string `mapstructure:"servicenow_password"`
	ServiceNowFallbackDir  string `mapstructure:"servicenow_fallback_dir"`

	// 1С:ТОИР adapter
	TOIRBaseURL     string `mapstructure:"toir_base_url"`
	TOIRUsername    string `mapstructure:"toir_username"`
	TOIRPassword    string `mapstructure:"toir_password"`
	TOIRFallbackDir string `mapstructure:"toir_fallback_dir"`

	// Jira adapter
	JiraBaseURL     string `mapstructure:"jira_base_url"`
	JiraEmail       string `mapstructure:"jira_email"`
	JiraAPIToken    string `mapstructure:"jira_api_token"`
	JiraFallbackDir string `mapstructure:"jira_fallback_dir"`

	// NATS Event Bus
	// UseNATSKV — использовать NATS JetStream KV для распределённого состояния устройств (ARCH-01).
	// Если true и NATS доступен — DeviceStateManager работает через NATS KV вместо sync.Map.
	// Требуется для горизонтального масштабирования (2+ реплики backend).
	UseNATSKV    bool   `mapstructure:"use_nats_kv"`
	NATSEmbedded bool   `mapstructure:"nats_embedded"`
	NATSURL      string `mapstructure:"nats_url"`
	NATSCreds    string `mapstructure:"nats_creds"`
	NATSTLS      bool   `mapstructure:"nats_tls"`
	// NATSRequired — если true, startup фейлится при недоступности NATS.
	// Для production (КИИ РБ) ДОЛЖНО быть true.
	NATSRequired bool `mapstructure:"nats_required"`

	// P3-1: Multi-Region deployment
	DeploymentRegion string `mapstructure:"deployment_region"`

	// Webhook secrets for bi-directional ITSM sync
	ServiceNowWebhookSecret string `mapstructure:"servicenow_webhook_secret"`
	JiraWebhookSecret       string `mapstructure:"jira_webhook_secret"`
	TOIRWebhookSecret       string `mapstructure:"toir_webhook_secret"`

	// Sync interval for bi-directional ITSM state machine (default 5m)
	ITSMSyncInterval string `mapstructure:"itsm_sync_interval"`

	// CORS allowed origins (ISO 27001 MT-4)
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`

	// Audit log HMAC signing key (ISO 27001 MT-3)
	AuditHMACKey string `mapstructure:"audit_hmac_key"`

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
	ONVIF     ONVIFConfig   // CCTV-2.2.1: ONVIF Profile S/T

	// Event Store configuration (DM-1.2.2)
	EventStore EventStoreConfig

	// Telegram bot configuration
	Telegram TelegramConfig

	// reCAPTCHA configuration (WO-4.1.1 — public work request submission)
	RecaptchaSecretKey string `mapstructure:"recaptcha_secret_key"`
	RecaptchaSiteKey   string `mapstructure:"recaptcha_site_key"`
	RecaptchaEnabled   bool   `mapstructure:"recaptcha_enabled"`

	// INT-02: SAML 2.0 / LDAP SSO configuration
	LDAPEnabled        bool   `mapstructure:"ldap_enabled"`
	LDAPHost           string `mapstructure:"ldap_host"`
	LDAPPort           int    `mapstructure:"ldap_port"`
	LDAPUseTLS         bool   `mapstructure:"ldap_use_tls"`
	LDAPBaseDN         string `mapstructure:"ldap_base_dn"`
	LDAPBindDN         string `mapstructure:"ldap_bind_dn"`
	LDAPBindPassword   string `mapstructure:"ldap_bind_password"`
	LDAPUserFilter     string `mapstructure:"ldap_user_filter"`
	LDAPLoginAttribute string `mapstructure:"ldap_login_attribute"`
	LDAPMailAttribute  string `mapstructure:"ldap_mail_attribute"`
	LDAPNameAttribute  string `mapstructure:"ldap_name_attribute"`
	LDAPDefaultRole    string `mapstructure:"ldap_default_role"`

	SAMLEnabled        bool   `mapstructure:"saml_enabled"`
	SAMLIdPMetadataURL string `mapstructure:"saml_idp_metadata_url"`
	SAMLIdPEntityID    string `mapstructure:"saml_idp_entity_id"`
	SAMLIdPSSOURL      string `mapstructure:"saml_idp_sso_url"`
	SAMLSPEntityID     string `mapstructure:"saml_sp_entity_id"`
	SAMLAcsURL         string `mapstructure:"saml_acs_url"`
	SAMLDefaultRole    string `mapstructure:"saml_default_role"`
	SAMLMailAttribute  string `mapstructure:"saml_mail_attribute"`
	SAMLNameAttribute  string `mapstructure:"saml_name_attribute"`
	SAMLRoleAttribute  string `mapstructure:"saml_role_attribute"`

	// DeepSeek AI API key for AI Assistant Chat (P2-1.2)
	// Хранится только на сервере, не пробрасывается на клиент.
	DeepSeekAPIKey string `mapstructure:"deepseek_api_key"`
}

// EventStoreConfig — настройки Event Store (DM-1.2.2: NATS + S3 Cold Storage)
type EventStoreConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	NATSURL            string `mapstructure:"nats_url"`
	NATSCreds          string `mapstructure:"nats_creds"`
	NATSTLS            bool   `mapstructure:"nats_tls"`
	S3Endpoint         string `mapstructure:"s3_endpoint"`
	S3Region           string `mapstructure:"s3_region"`
	S3Bucket           string `mapstructure:"s3_bucket"`
	S3AccessKey        string `mapstructure:"s3_access_key"`
	S3SecretKey        string `mapstructure:"s3_secret_key"`
	S3UseTLS           bool   `mapstructure:"s3_use_tls"`
	HotRetentionHours  int    `mapstructure:"hot_retention_hours"`
	ColdRetentionHours int    `mapstructure:"cold_retention_hours"`
	ValidationEnabled  bool   `mapstructure:"validation_enabled"` // JSON Schema validation (default: true)
}

// TelegramConfig — настройки Telegram бота
type TelegramConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
}

// ONVIFConfig — настройки ONVIF Profile S/T (CCTV-2.2.1)
type ONVIFConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Discovery     bool   `mapstructure:"discovery"`
	DiscoveryPort int    `mapstructure:"discovery_port"`
	ConnectMode   string `mapstructure:"connect_mode"`
	EdgeAgentURL  string `mapstructure:"edge_agent_url"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
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

// ValidateAuditHMACKey проверяет audit_hmac_key в зависимости от режима.
// Соответствует: ISO 27001 A.12.4.2, СТБ 34.101.30, СТБ 34.101.27 п. 7.2
func ValidateAuditHMACKey(key string, debug bool, logger interface {
	Error(string, ...interface{})
	Warn(string, ...interface{})
}) {
	const minKeyLength = 32 // 256 бит (СТБ 34.101.30)

	if debug {
		// Development mode: warn + dev-default
		if key == "" {
			logger.Warn("audit_hmac_key not set in development mode, using dev-default (INSECURE)")
		} else if len(key) < minKeyLength {
			logger.Warn("audit_hmac_key too short in development mode", "got_bytes", len(key), "min_bytes", minKeyLength)
		}
		return
	}

	// Production mode: log.Fatal если ключ отсутствует или слишком короткий
	if key == "" {
		logger.Error("FATAL: audit_hmac_key is required in production mode (ISO 27001 A.12.4.2)")
		fmt.Fprintf(os.Stderr, "FATAL: audit_hmac_key is required in production mode\n")
		os.Exit(1)
	}
	if len(key) < minKeyLength {
		logger.Error("FATAL: audit_hmac_key too short", "got_bytes", len(key), "min_bytes", minKeyLength)
		fmt.Fprintf(os.Stderr, "FATAL: audit_hmac_key must be at least %d bytes (got %d)\n", minKeyLength, len(key))
		os.Exit(1)
	}
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
	viper.SetDefault("log_server_port", 1514)

	// CMMS Atlas defaults
	viper.SetDefault("cmms_adapter", "internal")
	viper.SetDefault("atlas_fallback_dir", "/var/lib/gb-telemetry/fallback")

	// ServiceNow defaults
	viper.SetDefault("servicenow_fallback_dir", "/var/lib/gb-telemetry/fallback/servicenow")

	// 1С:ТОИР defaults
	viper.SetDefault("toir_fallback_dir", "/var/lib/gb-telemetry/fallback/toir")

	// Jira defaults
	viper.SetDefault("jira_fallback_dir", "/var/lib/gb-telemetry/fallback/jira")

	// NATS defaults (P0-BACKEND.1: JetStream mandatory for production)
	viper.SetDefault("use_nats_kv", true)   // P0-BACKEND.1: JetStream KV включён по умолчанию
	viper.SetDefault("nats_required", true) // P0-BACKEND.1: NATS обязателен для production
	viper.SetDefault("nats_embedded", false)
	viper.SetDefault("nats_url", "nats://localhost:4222")
	viper.SetDefault("nats_tls", false)

	// ITSM Sync defaults
	viper.SetDefault("itsm_sync_interval", "5m")
	viper.SetDefault("cors_allowed_origins", []string{"http://localhost:5173", "http://localhost:8080"})

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
	viper.SetDefault("ftp.user", "")     // W7: Должен быть задан через env/config
	viper.SetDefault("ftp.password", "") // W7: Должен быть задан через env/config
	viper.SetDefault("ftp.allowFiles", true)
	viper.SetDefault("hikvision.enabled", false)
	viper.SetDefault("snmp.enabled", false)
	viper.SetDefault("snmp.port", 1162)
	viper.SetDefault("snmp.community", "") // W6: Должен быть задан через env/config, не public
	viper.SetDefault("snmp.version", "v2c")

	// ONVIF defaults (CCTV-2.2.1)
	viper.SetDefault("onvif.enabled", false)
	viper.SetDefault("onvif.discovery", false)
	viper.SetDefault("onvif.discovery_port", 3702)
	viper.SetDefault("onvif.connect_mode", "direct")
	viper.SetDefault("onvif.edge_agent_url", "")
	viper.SetDefault("onvif.username", "")
	viper.SetDefault("onvif.password", "")

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

	// Event Store defaults (DM-1.2.2)
	viper.SetDefault("event_store.enabled", false)
	viper.SetDefault("event_store.nats_url", "nats://localhost:4222")
	viper.SetDefault("event_store.nats_creds", "")
	viper.SetDefault("event_store.nats_tls", false)
	viper.SetDefault("event_store.s3_endpoint", "")
	viper.SetDefault("event_store.s3_region", "us-east-1")
	viper.SetDefault("event_store.s3_bucket", "cctv-event-store")
	viper.SetDefault("event_store.s3_access_key", "")
	viper.SetDefault("event_store.s3_secret_key", "")
	viper.SetDefault("event_store.s3_use_tls", true)
	viper.SetDefault("event_store.hot_retention_hours", 8760)
	viper.SetDefault("event_store.cold_retention_hours", 43800)

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

	// ONVIF env bindings (CCTV-2.2.1)
	bindEnv("onvif.enabled", "GB_ONVIF_ENABLED")
	bindEnv("onvif.discovery", "GB_ONVIF_DISCOVERY")
	bindEnv("onvif.discovery_port", "GB_ONVIF_DISCOVERY_PORT")
	bindEnv("onvif.connect_mode", "GB_ONVIF_CONNECT_MODE")
	bindEnv("onvif.edge_agent_url", "GB_ONVIF_EDGE_AGENT_URL")
	bindEnv("onvif.username", "GB_ONVIF_USERNAME")
	bindEnv("onvif.password", "GB_ONVIF_PASSWORD")

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

	// ServiceNow
	bindEnv("servicenow_instance_url", "GB_SERVICENOW_INSTANCE_URL")
	bindEnv("servicenow_client_id", "GB_SERVICENOW_CLIENT_ID")
	bindEnv("servicenow_client_secret", "GB_SERVICENOW_CLIENT_SECRET")
	bindEnv("servicenow_token_url", "GB_SERVICENOW_TOKEN_URL")
	bindEnv("servicenow_username", "GB_SERVICENOW_USERNAME")
	bindEnv("servicenow_password", "GB_SERVICENOW_PASSWORD")
	bindEnv("servicenow_fallback_dir", "GB_SERVICENOW_FALLBACK_DIR")

	// 1С:ТОИР
	bindEnv("toir_base_url", "GB_TOIR_BASE_URL")
	bindEnv("toir_username", "GB_TOIR_USERNAME")
	bindEnv("toir_password", "GB_TOIR_PASSWORD")
	bindEnv("toir_fallback_dir", "GB_TOIR_FALLBACK_DIR")

	// Jira
	bindEnv("jira_base_url", "GB_JIRA_BASE_URL")
	bindEnv("jira_email", "GB_JIRA_EMAIL")
	bindEnv("jira_api_token", "GB_JIRA_API_TOKEN")
	bindEnv("jira_fallback_dir", "GB_JIRA_FALLBACK_DIR")

	// NATS
	bindEnv("use_nats_kv", "GB_USE_NATS_KV")
	bindEnv("nats_embedded", "GB_NATS_EMBEDDED")
	bindEnv("nats_url", "GB_NATS_URL")
	bindEnv("nats_creds", "GB_NATS_CREDS")
	bindEnv("nats_tls", "GB_NATS_TLS")
	bindEnv("nats_required", "GB_NATS_REQUIRED")

	// ITSM Webhook secrets
	bindEnv("servicenow_webhook_secret", "GB_SERVICENOW_WEBHOOK_SECRET")
	bindEnv("jira_webhook_secret", "GB_JIRA_WEBHOOK_SECRET")
	bindEnv("toir_webhook_secret", "GB_TOIR_WEBHOOK_SECRET")

	// ITSM Sync
	bindEnv("itsm_sync_interval", "GB_ITSM_SYNC_INTERVAL")
	bindEnv("audit_hmac_key", "GB_AUDIT_HMAC_KEY")

	// Telegram
	// Event Store env bindings
	bindEnv("event_store.enabled", "GB_EVENT_STORE_ENABLED")
	bindEnv("event_store.nats_url", "GB_EVENT_STORE_NATS_URL")
	bindEnv("event_store.nats_creds", "GB_EVENT_STORE_NATS_CREDS")
	bindEnv("event_store.nats_tls", "GB_EVENT_STORE_NATS_TLS")
	bindEnv("event_store.s3_endpoint", "GB_S3_ENDPOINT")
	bindEnv("event_store.s3_region", "GB_S3_REGION")
	bindEnv("event_store.s3_bucket", "GB_S3_BUCKET")
	bindEnv("event_store.s3_access_key", "GB_S3_ACCESS_KEY")
	bindEnv("event_store.s3_secret_key", "GB_S3_SECRET_KEY")
	bindEnv("event_store.s3_use_tls", "GB_S3_USE_TLS")
	bindEnv("event_store.hot_retention_hours", "GB_EVENT_STORE_HOT_RETENTION")
	bindEnv("event_store.cold_retention_hours", "GB_EVENT_STORE_COLD_RETENTION")

	bindEnv("telegram.enabled", "GB_TELEGRAM_ENABLED")
	bindEnv("telegram.token", "GB_TELEGRAM_TOKEN")

	// reCAPTCHA (WO-4.1.1)
	bindEnv("recaptcha_secret_key", "GB_RECAPTCHA_SECRET_KEY")
	bindEnv("recaptcha_site_key", "GB_RECAPTCHA_SITE_KEY")
	bindEnv("recaptcha_enabled", "GB_RECAPTCHA_ENABLED")

	// DeepSeek AI (P2-1.2)
	bindEnv("deepseek_api_key", "GB_DEEPSEEK_API_KEY")

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
		ONVIF: ONVIFConfig{
			Enabled:       viper.GetBool("onvif.enabled"),
			Discovery:     viper.GetBool("onvif.discovery"),
			DiscoveryPort: viper.GetInt("onvif.discovery_port"),
			ConnectMode:   viper.GetString("onvif.connect_mode"),
			EdgeAgentURL:  viper.GetString("onvif.edge_agent_url"),
			Username:      viper.GetString("onvif.username"),
			Password:      viper.GetString("onvif.password"),
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
		P2PGatewayURL:           viper.GetString("p2p_gateway_url"),
		P2PAPIKey:               viper.GetString("p2p_api_key"),
		CMMSAdapter:             viper.GetString("cmms_adapter"),
		AtlasURL:                viper.GetString("atlas_url"),
		AtlasAPIKey:             viper.GetString("atlas_api_key"),
		AtlasClientID:           viper.GetString("atlas_client_id"),
		AtlasClientSecret:       viper.GetString("atlas_client_secret"),
		AtlasTokenURL:           viper.GetString("atlas_token_url"),
		AtlasFallbackDir:        viper.GetString("atlas_fallback_dir"),
		ServiceNowInstanceURL:   viper.GetString("servicenow_instance_url"),
		ServiceNowClientID:      viper.GetString("servicenow_client_id"),
		ServiceNowClientSecret:  viper.GetString("servicenow_client_secret"),
		ServiceNowTokenURL:      viper.GetString("servicenow_token_url"),
		ServiceNowUsername:      viper.GetString("servicenow_username"),
		ServiceNowPassword:      viper.GetString("servicenow_password"),
		ServiceNowFallbackDir:   viper.GetString("servicenow_fallback_dir"),
		TOIRBaseURL:             viper.GetString("toir_base_url"),
		TOIRUsername:            viper.GetString("toir_username"),
		TOIRPassword:            viper.GetString("toir_password"),
		TOIRFallbackDir:         viper.GetString("toir_fallback_dir"),
		JiraBaseURL:             viper.GetString("jira_base_url"),
		JiraEmail:               viper.GetString("jira_email"),
		JiraAPIToken:            viper.GetString("jira_api_token"),
		JiraFallbackDir:         viper.GetString("jira_fallback_dir"),
		UseNATSKV:               viper.GetBool("use_nats_kv"),
		NATSEmbedded:            viper.GetBool("nats_embedded"),
		NATSURL:                 viper.GetString("nats_url"),
		NATSCreds:               viper.GetString("nats_creds"),
		NATSTLS:                 viper.GetBool("nats_tls"),
		NATSRequired:            viper.GetBool("nats_required"),
		ServiceNowWebhookSecret: viper.GetString("servicenow_webhook_secret"),
		JiraWebhookSecret:       viper.GetString("jira_webhook_secret"),
		TOIRWebhookSecret:       viper.GetString("toir_webhook_secret"),
		ITSMSyncInterval:        viper.GetString("itsm_sync_interval"),
		CORSAllowedOrigins:      viper.GetStringSlice("cors_allowed_origins"),
		AuditHMACKey:            viper.GetString("audit_hmac_key"),
		EventStore: EventStoreConfig{
			Enabled:            viper.GetBool("event_store.enabled"),
			NATSURL:            viper.GetString("event_store.nats_url"),
			NATSCreds:          viper.GetString("event_store.nats_creds"),
			NATSTLS:            viper.GetBool("event_store.nats_tls"),
			S3Endpoint:         viper.GetString("event_store.s3_endpoint"),
			S3Region:           viper.GetString("event_store.s3_region"),
			S3Bucket:           viper.GetString("event_store.s3_bucket"),
			S3AccessKey:        viper.GetString("event_store.s3_access_key"),
			S3SecretKey:        viper.GetString("event_store.s3_secret_key"),
			S3UseTLS:           viper.GetBool("event_store.s3_use_tls"),
			HotRetentionHours:  viper.GetInt("event_store.hot_retention_hours"),
			ColdRetentionHours: viper.GetInt("event_store.cold_retention_hours"),
			ValidationEnabled:  viper.GetBool("event_store.validation_enabled"),
		},
		Telegram: TelegramConfig{
			Enabled: viper.GetBool("telegram.enabled"),
			Token:   viper.GetString("telegram.token"),
		},
		RecaptchaSecretKey: viper.GetString("recaptcha_secret_key"),
		RecaptchaSiteKey:   viper.GetString("recaptcha_site_key"),
		RecaptchaEnabled:   viper.GetBool("recaptcha_enabled"),
		DeepSeekAPIKey:     viper.GetString("deepseek_api_key"),
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
