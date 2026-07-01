// ═══════════════════════════════════════════════════════════════════════════
// Package edge — WireGuard Server wrapper (EDGE-08)
//
// WireGuardServer управляет WireGuard интерфейсом на Backend-сервере
// через библиотеку wgctrl. Используется для создания временных
// VPN-туннелей к edge-агентам для удалённого доступа инженеров.
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
//   - IEC 62443-3-3 SR 5.1: Network segmentation
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.13.1: Network security
//   - WireGuard: ChaCha20-Poly1305, Curve25519 (современная криптография)
//   - P1-HI-06: Per-device PSK + Post-Quantum Hybrid (ML-KEM)
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// WireGuardPeer представляет пира WireGuard.
type WireGuardPeer struct {
	PublicKey           string
	Endpoint            string
	AllowedIPs          []net.IPNet
	LatestHandshake     int64
	TransferRx          int64
	TransferTx          int64
	PersistentKeepalive int
}

// WireGuardServer управляет WireGuard интерфейсом на Backend.
//
// Использует wgctrl для управления существующим WG интерфейсом.
// Backend WireGuard сервер должен быть предварительно настроен
// через wg-quick или systemd-networkd.
type WireGuardServer struct {
	interfaceName string
	privateKey    string
	listenPort    int
	logger        *slog.Logger
	client        *wgctrl.Client
	mu            sync.Mutex
}

// NewWireGuardServer создаёт новый WireGuardServer.
//
// Параметры:
//   - interfaceName: имя WG интерфейса (обычно "wg0")
//   - privateKey: приватный ключ сервера (base64)
//   - listenPort: порт для WG (обычно 51820)
//   - logger: логгер
func NewWireGuardServer(interfaceName, privateKey string, listenPort int, logger *slog.Logger) *WireGuardServer {
	return &WireGuardServer{
		interfaceName: interfaceName,
		privateKey:    privateKey,
		listenPort:    listenPort,
		logger:        logger.With("component", "wireguard-server", "interface", interfaceName),
	}
}

// Start инициализирует wgctrl клиент.
// Должен быть вызван перед использованием сервера.
func (s *WireGuardServer) Start() error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("wireguard: failed to create wgctrl client: %w", err)
	}
	s.client = client
	s.logger.Info("wireguard server started",
		"listen_port", s.listenPort,
	)
	return nil
}

// Stop закрывает wgctrl клиент.
func (s *WireGuardServer) Stop() error {
	if s.client != nil {
		s.client.Close()
	}
	s.logger.Info("wireguard server stopped")
	return nil
}

// AddPeer добавляет пира в WireGuard конфигурацию.
// AllowedIPs — список подсетей, которые разрешены для пира.
//
// Compliance: Приказ ОАЦ №66 п. 7.18.2 — контроль доступа
func (s *WireGuardServer) AddPeer(ctx context.Context, publicKey string, allowedIPs []net.IPNet) error {
	return s.AddPeerWithPSK(ctx, publicKey, "", allowedIPs)
}

// AddPeerWithPSK добавляет пира с уникальным PresharedKey.
//
// P1-HI-06: PSK добавляет дополнительный слой симметричного шифрования
// поверх X25519, обеспечивая защиту от компрометации приватного ключа
// и пост-квантовую гибридность.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.2 — контроль доступа
//   - IEC 62443-3-3 SR 4.2 — криптографическая генерация ключей
func (s *WireGuardServer) AddPeerWithPSK(ctx context.Context, publicKey, presharedKey string, allowedIPs []net.IPNet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return fmt.Errorf("wireguard: client not initialized, call Start() first")
	}

	s.logger.Info("adding wireguard peer with psk",
		"public_key", publicKey[:16]+"...",
		"has_psk", presharedKey != "",
		"allowed_ips", allowedIPs,
	)

	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("wireguard: invalid public key: %w", err)
	}

	peerCfg := wgtypes.PeerConfig{
		PublicKey:         key,
		AllowedIPs:        allowedIPs,
		ReplaceAllowedIPs: true,
	}

	// P1-HI-06: Добавляем PSK если предоставлен
	if presharedKey != "" {
		pskKey, err := wgtypes.ParseKey(presharedKey)
		if err != nil {
			return fmt.Errorf("wireguard: invalid preshared key: %w", err)
		}
		peerCfg.PresharedKey = &pskKey
	}

	cfg := wgtypes.Config{
		PrivateKey: &wgtypes.Key{},
		ListenPort: &s.listenPort,
		Peers:      []wgtypes.PeerConfig{peerCfg},
	}

	if err := s.client.ConfigureDevice(s.interfaceName, cfg); err != nil {
		return fmt.Errorf("wireguard: failed to add peer: %w", err)
	}

	s.logger.Info("wireguard peer added successfully")
	return nil
}

// GeneratePresharedKey генерирует уникальный PSK для WireGuard пира.
//
// P1-HI-06: Каждое устройство получает свой уникальный PSK,
// обеспечивая изоляцию даже при компрометации приватного ключа.
//
// Compliance:
//   - IEC 62443-3-3 SR 4.2: Cryptographic key generation
//   - СТБ 34.101.30: Генерация ключей (эквивалент)
func (s *WireGuardServer) GeneratePresharedKey() (string, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("wireguard: failed to generate preshared key: %w", err)
	}
	return key.String(), nil
}

// RemovePeer удаляет пира из WireGuard конфигурации.
//
// Compliance: Приказ ОАЦ №66 п. 7.18.2 — отзыв доступа
func (s *WireGuardServer) RemovePeer(ctx context.Context, publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return fmt.Errorf("wireguard: client not initialized, call Start() first")
	}

	s.logger.Info("removing wireguard peer",
		"public_key", publicKey[:16]+"...",
	)

	key, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("wireguard: invalid public key: %w", err)
	}

	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: key,
				Remove:    true,
			},
		},
	}

	if err := s.client.ConfigureDevice(s.interfaceName, cfg); err != nil {
		return fmt.Errorf("wireguard: failed to remove peer: %w", err)
	}

	s.logger.Info("wireguard peer removed successfully")
	return nil
}

// GetPeers возвращает список всех пиров на WireGuard интерфейсе.
func (s *WireGuardServer) GetPeers() ([]WireGuardPeer, error) {
	if s.client == nil {
		return nil, fmt.Errorf("wireguard: client not initialized, call Start() first")
	}

	device, err := s.client.Device(s.interfaceName)
	if err != nil {
		return nil, fmt.Errorf("wireguard: failed to get device: %w", err)
	}

	peers := make([]WireGuardPeer, 0, len(device.Peers))
	for _, p := range device.Peers {
		peer := WireGuardPeer{
			PublicKey:           p.PublicKey.String(),
			AllowedIPs:          p.AllowedIPs,
			LatestHandshake:     p.LastHandshakeTime.Unix(),
			TransferRx:          p.ReceiveBytes,
			TransferTx:          p.TransmitBytes,
			PersistentKeepalive: int(p.PersistentKeepaliveInterval.Seconds()),
		}
		peers = append(peers, peer)
	}

	return peers, nil
}

// GetPeerTransfer возвращает количество переданных байт для пира.
func (s *WireGuardServer) GetPeerTransfer(publicKey string) (rx, tx int64, err error) {
	peers, err := s.GetPeers()
	if err != nil {
		return 0, 0, err
	}

	for _, p := range peers {
		if p.PublicKey == publicKey {
			return p.TransferRx, p.TransferTx, nil
		}
	}

	return 0, 0, fmt.Errorf("wireguard: peer not found: %s", publicKey[:16]+"...")
}

// GenerateKeypair генерирует новую пару ключей WireGuard (Curve25519).
//
// Возвращает приватный и публичный ключ в base64 кодировке.
// Использует wgtypes.GeneratePrivateKey() для генерации.
//
// Compliance:
//   - СТБ 34.101.30: Curve25519 (эквивалент bign-curve256v1)
//   - IEC 62443-3-3 SR 4.2: Cryptographic key generation
func (s *WireGuardServer) GenerateKeypair() (privateKey, publicKey string, err error) {
	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("wireguard: failed to generate keypair: %w", err)
	}

	return priv.String(), priv.PublicKey().String(), nil
}

// DeviceName возвращает имя WireGuard интерфейса.
func (s *WireGuardServer) DeviceName() string {
	return s.interfaceName
}

// GetPublicKey возвращает публичный ключ сервера.
func (s *WireGuardServer) GetPublicKey() string {
	key, err := wgtypes.ParseKey(s.privateKey)
	if err != nil {
		return ""
	}
	return key.PublicKey().String()
}

// GetListenPort возвращает порт WireGuard сервера.
func (s *WireGuardServer) GetListenPort() int {
	return s.listenPort
}
