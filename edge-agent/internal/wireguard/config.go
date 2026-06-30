// ═══════════════════════════════════════════════════════════════════════════
// Package wireguard — WireGuard Tunnel Configuration (EDGE-08)
//
// Типы конфигурации для управления WireGuard туннелем на edge-агенте.
//
// Соответствие:
//   - IEC 62443-3-3 SL-4: Edge device security
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - WireGuard: ChaCha20-Poly1305, Curve25519
// ═══════════════════════════════════════════════════════════════════════════

package wireguard

// TunnelConfig — конфигурация WireGuard туннеля для edge-агента.
//
// Получается от Backend через MQTT команду start_vpn_session.
type TunnelConfig struct {
	// BackendEndpoint — адрес WireGuard сервера на Backend (host:port).
	BackendEndpoint string `json:"backend_endpoint"`

	// BackendPubKey — публичный ключ WireGuard сервера Backend.
	BackendPubKey string `json:"backend_pubkey"`

	// Address — IP адрес для WireGuard интерфейса на агенте (CIDR).
	// Пример: "10.0.0.2/32"
	Address string `json:"address"`

	// AllowedIPs — список разрешённых подсетей (LAN инженера).
	// Пример: ["192.168.1.0/24", "10.0.0.0/8"]
	AllowedIPs []string `json:"allowed_ips"`

	// DNS — DNS сервера для использования через туннель (опционально).
	DNS []string `json:"dns,omitempty"`

	// PersistentKeepalive — интервал keepalive в секундах.
	// Рекомендуется 25 для NAT.
	PersistentKeepalive int `json:"persistent_keepalive,omitempty"`

	// Duration — длительность сессии в секундах.
	Duration int `json:"duration_sec"`

	// PrivateKey — приватный ключ WireGuard для агента.
	// Генерируется на агенте или передаётся от Backend.
	PrivateKey string `json:"private_key,omitempty"`
}

// DefaultTunnelConfig возвращает конфигурацию туннеля по умолчанию.
func DefaultTunnelConfig() TunnelConfig {
	return TunnelConfig{
		PersistentKeepalive: 25,
		Duration:            3600, // 1 час
	}
}
