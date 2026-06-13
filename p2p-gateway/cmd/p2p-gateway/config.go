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
	DahuaPythonPath      string `yaml:"dahua_python_path"`
	DahuaScriptPath      string `yaml:"dahua_script_path"`
	XiongmaiNodePath     string `yaml:"xiongmai_node_path"`
	XiongmaiScriptPath   string `yaml:"xiongmai_script_path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}
