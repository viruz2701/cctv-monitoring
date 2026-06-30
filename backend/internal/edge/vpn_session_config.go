// ═══════════════════════════════════════════════════════════════════════════
// Package edge — VPN Session Config (EDGE-08)
//
// Настройки WireGuard VPN сессий и конфигурация сервера.
// ═══════════════════════════════════════════════════════════════════════════

package edge

// WireGuardConfig — конфигурация WireGuard сервера для VPN сессий.
type WireGuardConfig struct {
	// Enabled включает WireGuard VPN функциональность.
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	// InterfaceName — имя WireGuard интерфейса (wg0).
	InterfaceName string `mapstructure:"interface_name" yaml:"interface_name"`
	// ListenPort — UDP порт для WireGuard.
	ListenPort int `mapstructure:"listen_port" yaml:"listen_port"`
	// PrivateKey — приватный ключ сервера (base64).
	// Задаётся через GB_WIREGUARD_PRIVATE_KEY.
	PrivateKey string `mapstructure:"private_key" yaml:"private_key"`
	// DefaultDuration — длительность сессии по умолчанию.
	DefaultDuration string `mapstructure:"default_duration" yaml:"default_duration"`
	// MaxDuration — максимальная длительность сессии.
	MaxDuration string `mapstructure:"max_duration" yaml:"max_duration"`
	// CleanupInterval — интервал очистки истёкших сессий.
	CleanupInterval string `mapstructure:"cleanup_interval" yaml:"cleanup_interval"`
	// ServerEndpoint — публичный endpoint для WG (host:port).
	// Используется в конфиге для клиента.
	ServerEndpoint string `mapstructure:"server_endpoint" yaml:"server_endpoint"`
	// DNS — DNS сервера для клиента.
	DNS []string `mapstructure:"dns" yaml:"dns"`
}

// DefaultWireGuardConfig возвращает конфигурацию WireGuard по умолчанию.
func DefaultWireGuardConfig() WireGuardConfig {
	return WireGuardConfig{
		Enabled:         false,
		InterfaceName:   "wg0",
		ListenPort:      51820,
		DefaultDuration: "1h",
		MaxDuration:     "2h",
		CleanupInterval: "5m",
		ServerEndpoint:  "",
		DNS:             []string{},
	}
}
