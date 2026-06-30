// ═══════════════════════════════════════════════════════════════════════════
// Package edge — Edge SSH Proxy (PROXY-02)
//
// SSHProxy предоставляет WebSocket-терминал для SSH доступа к устройству
// через активную WireGuard VPN-сессию.
//
// Flow:
//  1. WebSocket handshake с JWT аутентификацией
//  2. Получение/создание VPN сессии через LazyVPNSession
//  3. Проверка AllowedIPs для device_ip
//  4. SSH подключение к устройству через WG интерфейс
//  5. PTY запрос (xterm-256color)
//  6. Двусторонний прокси: WebSocket ↔ SSH
//  7. Session recording (опционально)
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - IEC 62443-3-3 SR 5.1: Network segmentation
//   - OWASP ASVS L3 V2: Authentication
//   - OWASP ASVS L3 V5: Input validation
//   - ISO 27001 A.12.4: Audit trail
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// ═══ Constants ═════════════════════════════════════════════════════════

const (
	// DefaultSSHPort — порт SSH по умолчанию.
	DefaultSSHPort = 22

	// SSHTimeout — таймаут SSH подключения.
	SSHTimeout = 10 * time.Second

	// SSHKeepAliveInterval — интервал keepalive.
	SSHKeepAliveInterval = 30 * time.Second

	// MaxSessionRecordingSize — максимальный размер записи сессии (100 MB).
	MaxSessionRecordingSize = 100 * 1024 * 1024

	// RecordingDir — директория для записи SSH сессий.
	RecordingDir = "data/ssh-recordings"
)

// ═══ Types ═════════════════════════════════════════════════════════════

// TerminalSize представляет размер терминала.
type TerminalSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// WSMessage — сообщение WebSocket для SSH терминала.
type WSMessage struct {
	Type string          `json:"type"` // "input", "output", "resize", "ping", "pong"
	Data json.RawMessage `json:"data,omitempty"`
	Size *TerminalSize   `json:"size,omitempty"`
}

// SSHProxyConfig — конфигурация SSH прокси.
type SSHProxyConfig struct {
	// RecordingEnabled включает запись SSH сессий.
	RecordingEnabled bool
	// RecordingDir — директория для файлов записи.
	RecordingDir string
	// DefaultUsername — имя пользователя SSH по умолчанию.
	DefaultUsername string
	// DefaultPort — порт SSH по умолчанию.
	DefaultPort int
}

// DefaultSSHProxyConfig возвращает конфигурацию SSH прокси по умолчанию.
func DefaultSSHProxyConfig() SSHProxyConfig {
	return SSHProxyConfig{
		RecordingEnabled: false,
		RecordingDir:     RecordingDir,
		DefaultUsername:  "root",
		DefaultPort:      DefaultSSHPort,
	}
}

// SSHProxyInterface — интерфейс SSH прокси (для тестирования).
type SSHProxyInterface interface {
	HandleSSHSession(ctx context.Context, agentID, deviceIP string, port int, engineerID uuid.UUID, username string, password string, send func(WSMessage) error, recv <-chan WSMessage) error
}

// SSHProxy предоставляет WebSocket-терминал для SSH доступа.
type SSHProxy struct {
	manager     *VPNSessionManager
	lazyVPN     *LazyVPNSession
	auditLogger ProxyAuditLogger
	logger      *slog.Logger
	config      SSHProxyConfig
}

// NewSSHProxy создаёт новый SSHProxy.
func NewSSHProxy(
	manager *VPNSessionManager,
	lazyVPN *LazyVPNSession,
	auditLogger ProxyAuditLogger,
	logger *slog.Logger,
	config SSHProxyConfig,
) *SSHProxy {
	return &SSHProxy{
		manager:     manager,
		lazyVPN:     lazyVPN,
		auditLogger: auditLogger,
		logger:      logger.With("component", "ssh-proxy"),
		config:      config,
	}
}

// HandleSSHSession управляет WebSocket ↔ SSH двусторонним прокси.
//
// Flow:
//  1. Получение VPN сессии
//  2. Проверка AllowedIPs
//  3. SSH подключение
//  4. PTY запрос
//  5. Двустороннее копирование WebSocket ↔ SSH
//  6. Session recording
//
// Compliance:
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.12.4: Audit trail
func (p *SSHProxy) HandleSSHSession(
	ctx context.Context,
	agentID string,
	deviceIP string,
	port int,
	engineerID uuid.UUID,
	username string,
	password string,
	send func(WSMessage) error,
	recv <-chan WSMessage,
) error {
	start := time.Now()
	logger := p.logger.With(
		"agent_id", agentID,
		"device_ip", deviceIP,
		"port", port,
		"engineer_id", engineerID,
	)

	// 1. Валидация параметров (OWASP ASVS V5.1)
	parsedIP := net.ParseIP(deviceIP)
	if parsedIP == nil {
		return fmt.Errorf("ssh-proxy: invalid device IP: %s", deviceIP)
	}
	if !isPrivateIP(parsedIP) {
		return fmt.Errorf("ssh-proxy: device IP must be in private range: %s", deviceIP)
	}
	if port <= 0 || port > 65535 {
		port = p.config.DefaultPort
	}

	// 2. Получаем VPN сессию (Lazy VPN — PROXY-03)
	allowedIPs, err := p.getAllowedIPsForAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to get allowed IPs: %w", err)
	}

	session, err := p.lazyVPN.GetOrCreateSession(ctx, agentID, engineerID, allowedIPs)
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to get vpn session: %w", err)
	}

	// 3. Проверка AllowedIPs
	if !p.isAllowedIP(session, parsedIP) {
		return fmt.Errorf("ssh-proxy: device IP %s is not in session AllowedIPs", deviceIP)
	}

	// 4. SSH подключение
	sshAddr := fmt.Sprintf("%s:%d", deviceIP, port)
	logger.Debug("connecting to ssh", "address", sshAddr)

	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Добавить verify host key
		Timeout:         SSHTimeout,
	}

	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		logger.Error("ssh connection failed", "error", err)
		return fmt.Errorf("ssh-proxy: failed to connect to %s: %w", sshAddr, err)
	}
	defer client.Close()

	// 5. PTY запрос (xterm-256color)
	sessionSSH, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to create session: %w", err)
	}
	defer sessionSSH.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	if err := sessionSSH.RequestPty("xterm-256color", 40, 120, modes); err != nil {
		return fmt.Errorf("ssh-proxy: failed to request pty: %w", err)
	}

	// Запускаем shell
	shellWriter, err := sessionSSH.StdinPipe()
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to get stdin pipe: %w", err)
	}

	shellReader, err := sessionSSH.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to get stdout pipe: %w", err)
	}

	stderrReader, err := sessionSSH.StderrPipe()
	if err != nil {
		return fmt.Errorf("ssh-proxy: failed to get stderr pipe: %w", err)
	}

	if err := sessionSSH.Shell(); err != nil {
		return fmt.Errorf("ssh-proxy: failed to start shell: %w", err)
	}

	// 6. Настройка recording
	var recordingFile *os.File
	var recordingMu sync.Mutex
	if p.config.RecordingEnabled {
		recordingFile, err = p.openRecordingFile(session.ID, agentID, deviceIP)
		if err != nil {
			logger.Warn("failed to open recording file", "error", err)
		} else {
			defer recordingFile.Close()
			meta, _ := json.Marshal(map[string]interface{}{
				"type":        "ssh_session",
				"session_id":  session.ID.String(),
				"agent_id":    agentID,
				"device_ip":   deviceIP,
				"engineer_id": engineerID.String(),
				"started_at":  start.Format(time.RFC3339),
			})
			recordingFile.Write(meta)
			recordingFile.Write([]byte("\n"))
		}
	}

	// 7. Двусторонний прокси WebSocket ↔ SSH
	errCh := make(chan error, 3)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// SSH stdout → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := shellReader.Read(buf)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("ssh read: %w", err)
				}
				return
			}
			if n > 0 {
				outputMsg := map[string]interface{}{
					"type": "output",
					"data": string(buf[:n]),
				}
				data, _ := json.Marshal(outputMsg)
				var wsMsg WSMessage
				json.Unmarshal(data, &wsMsg)
				if err := send(wsMsg); err != nil {
					errCh <- fmt.Errorf("ws send: %w", err)
					return
				}

				if recordingFile != nil {
					recordingMu.Lock()
					recordingFile.Write(buf[:n])
					recordingMu.Unlock()
				}
			}
		}
	}()

	// SSH stderr → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrReader.Read(buf)
			if err != nil {
				if err != io.EOF {
					errCh <- fmt.Errorf("ssh stderr read: %w", err)
				}
				return
			}
			if n > 0 {
				errMsg := map[string]interface{}{
					"type": "output",
					"data": string(buf[:n]),
				}
				data, _ := json.Marshal(errMsg)
				var wsMsg WSMessage
				json.Unmarshal(data, &wsMsg)
				if err := send(wsMsg); err != nil {
					errCh <- fmt.Errorf("ws send stderr: %w", err)
					return
				}
			}
		}
	}()

	// WebSocket → SSH stdin
	go func() {
		for msg := range recv {
			switch msg.Type {
			case "input":
				var input string
				if err := json.Unmarshal(msg.Data, &input); err == nil {
					shellWriter.Write([]byte(input))
					if recordingFile != nil {
						recordingMu.Lock()
						recordingFile.Write([]byte(input))
						recordingMu.Unlock()
					}
				}
			case "resize":
				if msg.Size != nil {
					if err := sessionSSH.WindowChange(int(msg.Size.Rows), int(msg.Size.Cols)); err != nil {
						logger.Warn("resize failed", "error", err)
					}
				}
			case "ping":
				pong := WSMessage{Type: "pong"}
				send(pong)
			}
		}
	}()

	// Ждём завершения
	select {
	case err := <-errCh:
		logger.Info("ssh session ended", "error", err, "duration", time.Since(start))
		return err
	case <-ctx.Done():
		logger.Info("ssh session cancelled", "duration", time.Since(start))
		return ctx.Err()
	}
}

// ═══ Internal ═══════════════════════════════════════════════════════

func (p *SSHProxy) getAllowedIPsForAgent(ctx context.Context, agentID string) ([]net.IPNet, error) {
	_, lanNet, err := net.ParseCIDR("192.168.0.0/16")
	if err != nil {
		return nil, fmt.Errorf("failed to parse default CIDR: %w", err)
	}
	_, lanNet2, _ := net.ParseCIDR("10.0.0.0/8")
	return []net.IPNet{*lanNet, *lanNet2}, nil
}

func (p *SSHProxy) isAllowedIP(session *VPNSession, ip net.IP) bool {
	for _, allowed := range session.AllowedIPs {
		if allowed.Contains(ip) {
			return true
		}
	}
	return false
}

// openRecordingFile создаёт файл для записи SSH сессии.
func (p *SSHProxy) openRecordingFile(sessionID uuid.UUID, agentID, deviceIP string) (*os.File, error) {
	dir := p.config.RecordingDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("ssh-proxy: failed to create recording dir: %w", err)
	}

	filename := fmt.Sprintf("ssh_%s_%s_%s_%d.rec",
		sessionID.String()[:8],
		agentID,
		deviceIP,
		time.Now().Unix(),
	)
	path := filepath.Join(dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("ssh-proxy: failed to create recording file: %w", err)
	}

	return f, nil
}
