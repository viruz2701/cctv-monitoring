package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr           string `yaml:"listen_addr"`
	BackendAPIURL        string `yaml:"backend_api_url"`
	BackendAPIKey        string `yaml:"backend_api_key"`
	ProxyBinPath         string `yaml:"proxy_bin_path"`
	ProxyBaseRTSPPort    int    `yaml:"proxy_base_rtsp_port"`
	ProxyBaseONVIFPort   int    `yaml:"proxy_base_onvif_port"`
	DeviceStatusInterval int    `yaml:"device_status_interval_sec"`

	// Hikvision
	HikvisionUsername string `yaml:"hikvision_username"`
	HikvisionPassword string `yaml:"hikvision_password"`
	FFmpegPath        string `yaml:"ffmpeg_path"`

	// Dahua
	DahuaPythonPath string `yaml:"dahua_python_path"`
	DahuaScriptPath string `yaml:"dahua_script_path"`

	// Xiongmai (старый)
	XiongmaiNodePath   string `yaml:"xiongmai_node_path"`
	XiongmaiScriptPath string `yaml:"xiongmai_script_path"`

	// Jftech (новый)
	Jftech *JftechConfig `yaml:"jftech"`
}

type JftechConfig struct {
	UUID      string `yaml:"uuid"`
	AppKey    string `yaml:"app_key"`
	AppSecret string `yaml:"app_secret"`
	MoveCard  int    `yaml:"move_card"`
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Env vars override секретов (безопасность: никогда не хардкодить)
	if v := os.Getenv("P2P_BACKEND_API_KEY"); v != "" {
		cfg.BackendAPIKey = v
	}
	if v := os.Getenv("P2P_HIKVISION_USERNAME"); v != "" {
		cfg.HikvisionUsername = v
	}
	if v := os.Getenv("P2P_HIKVISION_PASSWORD"); v != "" {
		cfg.HikvisionPassword = v
	}
	if v := os.Getenv("P2P_JFTECH_APP_KEY"); v != "" {
		if cfg.Jftech == nil {
			cfg.Jftech = &JftechConfig{}
		}
		cfg.Jftech.AppKey = v
	}
	if v := os.Getenv("P2P_JFTECH_APP_SECRET"); v != "" {
		if cfg.Jftech == nil {
			cfg.Jftech = &JftechConfig{}
		}
		cfg.Jftech.AppSecret = v
	}
	if v := os.Getenv("P2P_JFTECH_UUID"); v != "" {
		if cfg.Jftech == nil {
			cfg.Jftech = &JftechConfig{}
		}
		cfg.Jftech.UUID = v
	}

	return &cfg, nil
}
