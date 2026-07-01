// ═══════════════════════════════════════════════════════════════════════════
// Package edge — WireGuard Config Generator (SELFSERV-01)
//
// WGConfigGenerator генерирует стандартные wg-quick конфигурационные
// файлы для WireGuard VPN клиентов. Поддерживает HMAC подпись конфига
// для обеспечения целостности и защиты от подмены.
//
// Self-Service Flow:
//   1. Инженер создаёт VPN сессию через admin/support
//   2. Получает ссылку на скачивание конфига
//   3. Скачивает .conf файл или копирует в буфер обмена
//   4. Импортирует в WireGuard клиент
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 → engineer client)
//   - IEC 62443-3-3 SR 4.2: Cryptographic key generation
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.12.4: Audit trail (HMAC подпись)
//   - OWASP ASVS V2.1: Session management
//   - P1-HI-06: Per-device PSK + Post-Quantum Hybrid (ML-KEM)
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// WGConfigGenerator генерирует WireGuard конфигурации для клиентов.
type WGConfigGenerator struct{}

// WGClientConfig представляет полную конфигурацию WireGuard клиента.
type WGClientConfig struct {
	Interface WGInterfaceConfig `json:"interface"`
	Peer      WGPeerConfig      `json:"peer"`
	// PostQuantumHybridKey — публичный ключ ML-KEM (Kyber) для PQ-гибрида.
	// P1-HI-06: Post-Quantum Cryptography Hybrid (X25519 + ML-KEM).
	PostQuantumHybridKey string `json:"post_quantum_hybrid_key,omitempty"`
}

// WGInterfaceConfig — настройки [Interface] секции wg-quick конфига.
type WGInterfaceConfig struct {
	// PrivateKey — приватный ключ клиента (base64).
	PrivateKey string `json:"private_key"`
	// Address — IP адрес WireGuard интерфейса клиента (CIDR).
	Address string `json:"address"`
	// DNS — DNS сервера для клиента.
	DNS []string `json:"dns,omitempty"`
}

// WGPeerConfig — настройки [Peer] секции wg-quick конфига.
type WGPeerConfig struct {
	// PublicKey — публичный ключ сервера (base64).
	PublicKey string `json:"public_key"`
	// PresharedKey — PSK для дополнительного слоя симметричного шифрования.
	// P1-HI-06: Уникальный PSK на каждое устройство.
	PresharedKey string `json:"preshared_key,omitempty"`
	// AllowedIPs — список разрешённых подсетей (CIDR).
	AllowedIPs []string `json:"allowed_ips"`
	// Endpoint — адрес WireGuard сервера (host:port).
	Endpoint string `json:"endpoint"`
	// PersistentKeepalive — интервал keepalive в секундах (для NAT).
	PersistentKeepalive int `json:"persistent_keepalive"`
}

// NewWGConfigGenerator создаёт новый генератор конфигов.
func NewWGConfigGenerator() *WGConfigGenerator {
	return &WGConfigGenerator{}
}

// GenerateConfig генерирует структурированную конфигурацию WireGuard клиента.
//
// Параметры:
//   - session: VPN сессия с ключами и allowed IPs
//   - serverPubKey: публичный ключ WireGuard сервера
//   - serverEndpoint: публичный endpoint сервера (host:port)
//   - clientAddress: IP адрес для WG интерфейса клиента
//   - dns: DNS сервера для клиента
//
// Compliance:
//   - IEC 62443-3-3 SR 4.2: Cryptographic key generation
//   - Приказ ОАЦ №66 п. 7.18.2: Контроль удалённого доступа
func (g *WGConfigGenerator) GenerateConfig(
	session *VPNSession,
	serverPubKey, serverEndpoint, clientAddress string,
	dns []string,
) (*WGClientConfig, error) {
	if session == nil {
		return nil, fmt.Errorf("wg-config: session is nil")
	}
	if serverPubKey == "" {
		return nil, fmt.Errorf("wg-config: server public key is required")
	}
	if serverEndpoint == "" {
		return nil, fmt.Errorf("wg-config: server endpoint is required")
	}
	if clientAddress == "" {
		return nil, fmt.Errorf("wg-config: client address is required")
	}

	if session.Status != "active" {
		return nil, fmt.Errorf("wg-config: session is not active (status: %s)", session.Status)
	}

	// Конвертируем allowed IPs в строки CIDR
	allowedIPs := make([]string, 0, len(session.AllowedIPs))
	for _, ipNet := range session.AllowedIPs {
		allowedIPs = append(allowedIPs, ipNet.String())
	}

	if len(allowedIPs) == 0 {
		// Если allowed IPs пустые, добавляем default route через VPN
		allowedIPs = append(allowedIPs, "0.0.0.0/0")
	}

	if dns == nil {
		dns = []string{}
	}

	config := &WGClientConfig{
		Interface: WGInterfaceConfig{
			PrivateKey: session.PrivateKey,
			Address:    clientAddress,
			DNS:        dns,
		},
		Peer: WGPeerConfig{
			PublicKey:           serverPubKey,
			PresharedKey:        session.PresharedKey,
			AllowedIPs:          allowedIPs,
			Endpoint:            serverEndpoint,
			PersistentKeepalive: 25,
		},
		PostQuantumHybridKey: session.PQHybridPublicKey,
	}

	return config, nil
}

// GenerateConfigFile генерирует стандартный wg-quick конфигурационный файл.
//
// Формат вывода:
//
//	[Interface]
//	PrivateKey = <base64>
//	Address = <CIDR>
//	DNS = <dns1>, <dns2>
//	# PostQuantumHybridKey = <base64> (ML-KEM, P1-HI-06)
//
//	[Peer]
//	PublicKey = <base64>
//	PresharedKey = <base64> (P1-HI-06)
//	AllowedIPs = <cidr1>, <cidr2>
//	Endpoint = <host:port>
//	PersistentKeepalive = <seconds>
//
// Возвращает готовый к сохранению текст конфигурационного файла.
func (g *WGConfigGenerator) GenerateConfigFile(
	session *VPNSession,
	serverPubKey, serverEndpoint, clientAddress string,
	dns []string,
) (string, error) {
	config, err := g.GenerateConfig(session, serverPubKey, serverEndpoint, clientAddress, dns)
	if err != nil {
		return "", fmt.Errorf("wg-config: failed to generate config: %w", err)
	}

	var sb strings.Builder

	// [Interface] секция
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", config.Interface.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", config.Interface.Address))
	if len(config.Interface.DNS) > 0 {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", strings.Join(config.Interface.DNS, ", ")))
	}
	// P1-HI-06: Post-Quantum Hybrid Key (ML-KEM) — комментарий в конфиге
	if config.PostQuantumHybridKey != "" {
		sb.WriteString(fmt.Sprintf("# PostQuantumHybridKey = %s\n", config.PostQuantumHybridKey))
	}
	sb.WriteString("\n")

	// [Peer] секция
	sb.WriteString("[Peer]\n")
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", config.Peer.PublicKey))
	// P1-HI-06: Per-device PresharedKey для дополнительного слоя шифрования
	if config.Peer.PresharedKey != "" {
		sb.WriteString(fmt.Sprintf("PresharedKey = %s\n", config.Peer.PresharedKey))
	}
	sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(config.Peer.AllowedIPs, ", ")))
	sb.WriteString(fmt.Sprintf("Endpoint = %s\n", config.Peer.Endpoint))
	sb.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", config.Peer.PersistentKeepalive))

	return sb.String(), nil
}

// SignConfig подписывает конфигурационный файл HMAC-SHA256 для обеспечения
// целостности. Подпись добавляется в виде комментария в начало файла.
//
// Compliance:
//   - ISO 27001 A.12.4: Защита целостности audit trail
//   - СТБ 34.101.27: Контроль целостности конфигурации
//
// Возвращает конфиг с добавленной HMAC подписью.
func (g *WGConfigGenerator) SignConfig(config string, hmacKey []byte) string {
	if len(hmacKey) == 0 {
		return config
	}

	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(config))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Добавляем подпись как комментарий в начало файла
	return fmt.Sprintf("# HMAC-SHA256: %s\n# Generated by CCTV Health Monitor Self-Service VPN\n%s", signature, config)
}

// VerifyConfig проверяет HMAC подпись конфигурационного файла.
//
// Возвращает true если подпись валидна.
func (g *WGConfigGenerator) VerifyConfig(signedConfig string, hmacKey []byte) bool {
	if len(hmacKey) == 0 {
		return true
	}

	lines := strings.SplitN(signedConfig, "\n", 2)
	if len(lines) < 2 {
		return false
	}

	// Извлекаем подпись из первой строки
	sigLine := strings.TrimPrefix(lines[0], "# HMAC-SHA256: ")
	sigLine = strings.TrimSpace(sigLine)

	// Проверяем подпись
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(lines[1]))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigLine), []byte(expectedSig))
}
