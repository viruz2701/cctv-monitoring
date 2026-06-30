// ═══════════════════════════════════════════════════════════════════════════
// Package wireguard — WireGuard Tunnel Manager (EDGE-08)
//
// WireGuardManager управляет WireGuard туннелем на edge-агенте.
// Использует exec.Command для вызова wg-quick/wg или напрямую ip команды.
//
// Flow StartTunnel:
//   1. Create WG interface (if not exists) — ip link add
//   2. Assign IP address — ip addr add
//   3. Add peer (Backend WG server) — wg set
//   4. Set AllowedIPs (LAN of client) — wg set
//   5. Bring interface up — ip link set up
//
// Flow StopTunnel:
//   1. Remove peer — wg set peer remove
//   2. Bring interface down — ip link set down
//   3. Delete interface — ip link delete
//   4. Clean routes — ip route flush
//
// Соответствие:
//   - IEC 62443-3-3 SL-4: Edge device security
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.13.1: Network security
// ═══════════════════════════════════════════════════════════════════════════

package wireguard

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
)

// WireGuardManager управляет WireGuard интерфейсом на edge-агенте.
//
// На edge-агентах используется exec.Command для управления WG,
// так как wgctrl требует доступ к Netlink, что не всегда доступно
// на OpenWRT/embedded устройствах.
type WireGuardManager struct {
	interfaceName string
	privateKey    string
	logger        *slog.Logger
	mu            sync.Mutex
	isActive      bool
}

// NewWireGuardManager создаёт новый WireGuardManager.
//
// Параметры:
//   - interfaceName: имя WG интерфейса (обычно "wg0")
//   - privateKey: приватный ключ агента (base64, генерируется при старте)
//   - logger: логгер
func NewWireGuardManager(interfaceName, privateKey string, logger *slog.Logger) *WireGuardManager {
	return &WireGuardManager{
		interfaceName: interfaceName,
		privateKey:    privateKey,
		logger:        logger.With("component", "wireguard-manager", "interface", interfaceName),
	}
}

// StartTunnel создаёт и запускает WireGuard туннель к Backend.
//
// Flow:
//  1. Create WG interface (if not exists)
//  2. Assign IP address
//  3. Configure private key
//  4. Add peer (Backend WG server) with AllowedIPs
//  5. Bring interface up
//
// Compliance:
//   - IEC 62443-3-3 SR 5.1: Network segmentation
//   - Приказ ОАЦ №66 п. 7.18.2: Удалённый доступ
func (m *WireGuardManager) StartTunnel(ctx context.Context, config TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isActive {
		return fmt.Errorf("wireguard: tunnel already active on %s", m.interfaceName)
	}

	logger := m.logger.With("backend", config.BackendEndpoint)
	logger.Info("starting wireguard tunnel")

	// 1. Create WG interface
	if err := m.execCmd(ctx, "ip", "link", "add", m.interfaceName, "type", "wireguard"); err != nil {
		// Interface may already exist — that's OK
		logger.Debug("interface may already exist", "error", err)
	}

	// 2. Configure private key
	privKey := m.privateKey
	if config.PrivateKey != "" {
		privKey = config.PrivateKey
	}

	if err := m.wgSet(ctx, "private-key", privKey); err != nil {
		return fmt.Errorf("wireguard: failed to set private key: %w", err)
	}

	// 3. Assign IP address
	if config.Address != "" {
		if err := m.execCmd(ctx, "ip", "addr", "add", config.Address, "dev", m.interfaceName); err != nil {
			// Address may already be assigned
			logger.Debug("address assignment info", "error", err)
		}
	}

	// 4. Generate public key and add peer
	// On edge-agent, we derive pubkey via 'wg pubkey' command
	pubKey, err := m.getPublicKey(ctx)
	if err != nil {
		return fmt.Errorf("wireguard: failed to get public key: %w", err)
	}
	logger.Info("agent public key", "public_key", pubKey)

	// 5. Add peer (Backend WG server)
	keepalive := config.PersistentKeepalive
	if keepalive <= 0 {
		keepalive = 25
	}

	peerArgs := []string{
		"set", m.interfaceName,
		"peer", config.BackendPubKey,
		"endpoint", config.BackendEndpoint,
		"allowed-ips", "0.0.0.0/0", // Route all traffic through tunnel
		"persistent-keepalive", fmt.Sprintf("%d", keepalive),
	}

	if err := m.execCmd(ctx, "wg", peerArgs...); err != nil {
		return fmt.Errorf("wireguard: failed to add peer: %w", err)
	}

	// 6. Add specific allowed IPs (if different from 0.0.0.0/0)
	if len(config.AllowedIPs) > 0 {
		allowedIPs := ""
		for i, ip := range config.AllowedIPs {
			if i > 0 {
				allowedIPs += ","
			}
			allowedIPs += ip
		}

		peerArgs = []string{
			"set", m.interfaceName,
			"peer", config.BackendPubKey,
			"allowed-ips", allowedIPs,
		}
		if err := m.execCmd(ctx, "wg", peerArgs...); err != nil {
			logger.Warn("failed to set specific allowed IPs", "error", err)
		}
	}

	// 7. Bring interface up
	if err := m.execCmd(ctx, "ip", "link", "set", m.interfaceName, "up"); err != nil {
		return fmt.Errorf("wireguard: failed to bring interface up: %w", err)
	}

	m.isActive = true
	logger.Info("wireguard tunnel started successfully")

	return nil
}

// StopTunnel останавливает и удаляет WireGuard туннель.
//
// Flow:
//  1. Remove peer
//  2. Bring interface down
//  3. Delete interface
//  4. Clean routes
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.2: Отзыв доступа
//   - IEC 62443-3-3 SR 7.2: Session termination
func (m *WireGuardManager) StopTunnel(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isActive {
		m.logger.Debug("tunnel not active, nothing to stop")
		return nil
	}

	logger := m.logger
	logger.Info("stopping wireguard tunnel")

	// 1. Bring interface down
	if err := m.execCmd(ctx, "ip", "link", "set", m.interfaceName, "down"); err != nil {
		logger.Warn("failed to bring interface down", "error", err)
	}

	// 2. Delete interface
	if err := m.execCmd(ctx, "ip", "link", "delete", m.interfaceName); err != nil {
		// Interface may not exist or already deleted
		logger.Debug("interface deletion info", "error", err)
	}

	// 3. Flush routes for the interface
	if err := m.execCmd(ctx, "ip", "route", "flush", "dev", m.interfaceName); err != nil {
		logger.Debug("route flush info", "error", err)
	}

	m.isActive = false
	logger.Info("wireguard tunnel stopped")

	return nil
}

// IsActive возвращает статус туннеля.
func (m *WireGuardManager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isActive
}

// ═══ Внутренние методы ═══════════════════════════════════════════════

// execCmd выполняет shell команду.
func (m *WireGuardManager) execCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %s: %w", name, args, string(output), err)
	}
	m.logger.Debug("exec OK",
		"cmd", name,
		"args", args,
	)
	return nil
}

// wgSet выполняет wg set команду.
func (m *WireGuardManager) wgSet(ctx context.Context, key, value string) error {
	return m.execCmd(ctx, "wg", "set", m.interfaceName, key, value)
}

// getPublicKey получает публичный ключ из приватного через wg pubkey.
func (m *WireGuardManager) getPublicKey(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "wg", "pubkey")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("wireguard: failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write([]byte(m.privateKey))
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wireguard: failed to get public key: %w", err)
	}

	// Remove trailing newline
	pubKey := string(output)
	if len(pubKey) > 0 && pubKey[len(pubKey)-1] == '\n' {
		pubKey = pubKey[:len(pubKey)-1]
	}

	return pubKey, nil
}
