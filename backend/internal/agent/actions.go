// Package agent — remediation actions: ISAPI, ONVIF, SNMP, SSH.
package agent

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/icholy/digest"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ActionExecutor выполняет remediation-действия на устройствах.
type ActionExecutor struct {
	p2pGatewayURL string
	p2pAPIKey     string
	logger        *slog.Logger
	httpClient    *http.Client
}

// NewActionExecutor создаёт новый ActionExecutor.
func NewActionExecutor(p2pGatewayURL, p2pAPIKey string, logger *slog.Logger) *ActionExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &ActionExecutor{
		p2pGatewayURL: p2pGatewayURL,
		p2pAPIKey:     p2pAPIKey,
		logger:        logger,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ── ISAPI Actions (Hikvision) ──────────────────────────────────────

// ISAPIReboot перезагружает камеру Hikvision через ISAPI /System/reboot.
func (e *ActionExecutor) ISAPIReboot(ctx context.Context, deviceIP, username, password string) error {
	url := fmt.Sprintf("http://%s/ISAPI/System/reboot", deviceIP)
	return e.isapiPut(ctx, url, username, password, nil)
}

// ISAPIReset выполняет сброс Hikvision через ISAPI /System/factoryReset.
// resetMode: "basic" (сохранить сеть), "full" (полный сброс).
func (e *ActionExecutor) ISAPIReset(ctx context.Context, deviceIP, username, password, resetMode string) error {
	if resetMode == "" {
		resetMode = "basic"
	}
	url := fmt.Sprintf("http://%s/ISAPI/System/factoryReset?mode=%s", deviceIP, resetMode)
	return e.isapiPut(ctx, url, username, password, nil)
}

// ISAPIRestore восстанавливает конфигурацию Hikvision из бэкапа.
func (e *ActionExecutor) ISAPIRestore(ctx context.Context, deviceIP, username, password string, configXML []byte) error {
	url := fmt.Sprintf("http://%s/ISAPI/System/configurationData?mode=restore", deviceIP)
	return e.isapiPut(ctx, url, username, password, configXML)
}

// ISAPIRebootViaP2P перезагружает камеру через P2P Gateway (прокси).
func (e *ActionExecutor) ISAPIRebootViaP2P(ctx context.Context, p2pSerial string) error {
	if e.p2pGatewayURL == "" {
		return fmt.Errorf("p2p gateway URL not configured")
	}

	payload := fmt.Sprintf(`{"serial":"%s","command":"reboot"}`, p2pSerial)
	req, err := http.NewRequestWithContext(ctx, "POST", e.p2pGatewayURL+"/api/v1/device/command", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("p2p request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", e.p2pAPIKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("p2p call: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("p2p reboot failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	e.logger.Info("isapi reboot via p2p", "serial", p2pSerial, "status", resp.StatusCode)
	return nil
}

func (e *ActionExecutor) isapiPut(ctx context.Context, url, username, password string, body []byte) error {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, reqBody)
	if err != nil {
		return fmt.Errorf("isapi request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml")

	// Digest auth через icholy/digest
	transport := &digest.Transport{
		Username: username,
		Password: password,
	}

	client := &http.Client{
		Timeout:   e.httpClient.Timeout,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("isapi call %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("isapi %s failed: status=%d body=%s", url, resp.StatusCode, string(respBody))
	}

	e.logger.Info("isapi action", "url", url, "status", resp.StatusCode)
	return nil
}

// ── ONVIF Actions ──────────────────────────────────────────────────

// ONVIFReboot перезагружает камеру через ONVIF SystemReboot.
func (e *ActionExecutor) ONVIFReboot(ctx context.Context, deviceIP, username, password string) error {
	soapBody := fmt.Sprintf(onvifRebootTemplate, username, password)
	return e.onvifSoapCall(ctx, deviceIP, soapBody, "SystemReboot", username, password)
}

// ONVIFPTZHome отправляет PTZ в home-позицию.
func (e *ActionExecutor) ONVIFPTZHome(ctx context.Context, deviceIP, username, password string) error {
	soapBody := fmt.Sprintf(onvifPTZHomeTemplate, username, password)
	return e.onvifSoapCall(ctx, deviceIP, soapBody, "PTZHome", username, password)
}

func (e *ActionExecutor) onvifSoapCall(ctx context.Context, deviceIP, soapBody, action, username, password string) error {
	url := fmt.Sprintf("http://%s/onvif/device_service", deviceIP)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(soapBody))
	if err != nil {
		return fmt.Errorf("onvif request: %w", err)
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", fmt.Sprintf(`"http://www.onvif.org/ver10/device/wsdl/%s"`, action))

	// WS-Digest аутентификация
	transport := &digest.Transport{
		Username: username,
		Password: password,
	}

	client := &http.Client{
		Timeout:   e.httpClient.Timeout,
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("onvif %s: %w", action, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("onvif %s failed: status=%d body=%s", action, resp.StatusCode, string(respBody))
	}

	// Проверяем, что SOAP ответ не содержит Fault
	if strings.Contains(string(respBody), "SOAP-ENV:Fault") {
		return fmt.Errorf("onvif %s SOAP fault: %s", action, extractFaultString(respBody))
	}

	e.logger.Info("onvif action", "ip", deviceIP, "action", action, "status", resp.StatusCode)
	return nil
}

func extractFaultString(body []byte) string {
	// Простой парсинг <faultstring>...</faultstring>
	start := bytes.Index(body, []byte("<faultstring"))
	if start < 0 {
		return "unknown fault"
	}
	end := bytes.IndexByte(body[start:], '>')
	if end < 0 {
		return string(body[start:])
	}
	closeTag := bytes.Index(body[start+end+1:], []byte("</faultstring>"))
	if closeTag < 0 {
		return string(body[start+end+1:])
	}
	return string(body[start+end+1 : start+end+1+closeTag])
}

// ── SNMP Actions ───────────────────────────────────────────────────

// SNMPReset отправляет SNMP SET для перезагрузки устройства.
// Использует OID 1.3.6.1.4.1.39165.1.5.0 (Hikvision reboot OID) или 1.3.6.1.4.1.1004849.2.1.2.99.0 (Dahua).
func (e *ActionExecutor) SNMPReset(ctx context.Context, deviceIP string, snmpConfig SNMPActionConfig) error {
	rebootOID := snmpConfig.RebootOID
	if rebootOID == "" {
		rebootOID = "1.3.6.1.4.1.39165.1.5.0" // Hikvision default
	}

	gs := &gosnmp.GoSNMP{
		Target:    deviceIP,
		Port:      uint16(snmpConfig.Port),
		Community: snmpConfig.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(snmpConfig.TimeoutSec) * time.Second,
		Retries:   snmpConfig.Retries,
	}

	if snmpConfig.Version == "3" {
		gs.Version = gosnmp.Version3
		gs.SecurityModel = gosnmp.UserSecurityModel
		gs.MsgFlags = gosnmp.AuthPriv
		gs.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 snmpConfig.Username,
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: snmpConfig.AuthPassword,
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        snmpConfig.PrivPassword,
		}
	}

	if err := gs.Connect(); err != nil {
		return fmt.Errorf("snmp connect %s: %w", deviceIP, err)
	}
	defer gs.Conn.Close()

	// SNMP SET: i = 1 (integer) с value = 1 (reboot)
	pdu := gosnmp.SnmpPDU{
		Name:  rebootOID,
		Type:  gosnmp.Integer,
		Value: 1,
	}

	result, err := gs.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return fmt.Errorf("snmp set %s: %w", deviceIP, err)
	}

	if result.Error != 0 {
		return fmt.Errorf("snmp set error: %s (code %d)", result.Error, result.Error)
	}

	e.logger.Info("snmp reset", "ip", deviceIP, "oid", rebootOID)
	return nil
}

// SNMPColdStart отправляет SNMP cold start trap.
func (e *ActionExecutor) SNMPColdStart(ctx context.Context, deviceIP string, snmpConfig SNMPActionConfig) error {
	coldStartOID := "1.3.6.1.6.3.1.1.5.1" // coldStart trap OID

	gs := &gosnmp.GoSNMP{
		Target:    deviceIP,
		Port:      uint16(snmpConfig.Port),
		Community: snmpConfig.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(snmpConfig.TimeoutSec) * time.Second,
		Retries:   snmpConfig.Retries,
	}

	if err := gs.Connect(); err != nil {
		return fmt.Errorf("snmp connect %s: %w", deviceIP, err)
	}
	defer gs.Conn.Close()

	// Используем SNMP SET для cold start OID
	pdu := gosnmp.SnmpPDU{
		Name:  coldStartOID,
		Type:  gosnmp.Integer,
		Value: 1,
	}

	result, err := gs.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return fmt.Errorf("snmp cold start %s: %w", deviceIP, err)
	}

	if result.Error != 0 {
		return fmt.Errorf("snmp cold start error: %s (code %d)", result.Error, result.Error)
	}

	e.logger.Info("snmp cold start", "ip", deviceIP)
	return nil
}

// SNMPActionConfig — конфигурация SNMP-действия.
type SNMPActionConfig struct {
	Port         int
	Community    string
	Version      string // "2c" или "3"
	Username     string
	AuthPassword string
	PrivPassword string
	RebootOID    string
	TimeoutSec   int
	Retries      int
}

// DefaultSNMPConfig возвращает дефолтную конфигурацию.
func DefaultSNMPConfig() SNMPActionConfig {
	return SNMPActionConfig{
		Port:       161,
		Community:  "public",
		Version:    "2c",
		TimeoutSec: 5,
		Retries:    2,
	}
}

// ── SSH Actions ────────────────────────────────────────────────────

// SSHCommandResult — результат выполнения SSH команды.
type SSHCommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// sshClientConfig создаёт конфигурацию SSH клиента.
// Приоритет: ключ из SSH_PRIVATE_KEY_PATH (или SSH_PRIVATE_KEY), затем password.
// Host key verification: используется known_hosts из SSH_KNOWN_HOSTS_PATH или ~/.ssh/known_hosts.
func sshClientConfig(username, password string, timeout time.Duration) *ssh.ClientConfig {
	authMethods := buildSSHAuthMethods(username, password)

	hostKeyCallback, err := newHostKeyCallback()
	if err != nil {
		hostKeyCallback = ssh.InsecureIgnoreHostKey() // fallback only if no known_hosts available
	}

	return &ssh.ClientConfig{
		User:              username,
		Auth:              authMethods,
		HostKeyCallback:   hostKeyCallback,
		Timeout:           timeout,
		HostKeyAlgorithms: nil, // use defaults
	}
}

// buildSSHAuthMethods строит цепочку методов аутентификации.
// 1. Публичный ключ из SSH_PRIVATE_KEY_PATH или SSH_PRIVATE_KEY (env).
// 2. Пароль (если не пустой).
func buildSSHAuthMethods(username, password string) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// Key-based auth from env vars
	if signer := loadSSHSignerFromEnv(); signer != nil {
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// Password auth fallback
	if password != "" {
		methods = append(methods, ssh.Password(password))
	}

	// Keyboard-interactive fallback (для устройств, которым нужен challenge-response)
	if len(methods) == 0 {
		methods = append(methods, ssh.Password(password))
	}

	return methods
}

// loadSSHSignerFromEnv загружает SSH-ключ из переменных окружения.
// SSH_PRIVATE_KEY_PATH — путь к файлу с приватным ключом.
// SSH_PRIVATE_KEY — содержимое ключа в PEM-формате (напрямую).
func loadSSHSignerFromEnv() ssh.Signer {
	keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	if keyPath != "" {
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil
		}
		return signer
	}

	keyData := os.Getenv("SSH_PRIVATE_KEY")
	if keyData != "" {
		signer, err := ssh.ParsePrivateKey([]byte(keyData))
		if err != nil {
			return nil
		}
		return signer
	}

	return nil
}

// newHostKeyCallback создаёт callback для проверки host key через known_hosts.
func newHostKeyCallback() (ssh.HostKeyCallback, error) {
	knownHostsPath := os.Getenv("SSH_KNOWN_HOSTS_PATH")
	if knownHostsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			knownHostsPath = filepath.Join(homeDir, ".ssh", "known_hosts")
		}
	}

	if knownHostsPath == "" {
		return nil, fmt.Errorf("no known_hosts path available")
	}

	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts: %w", err)
	}

	return hostKeyCallback, nil
}

// SSHRestartDevice перезагружает устройство по SSH (crypto/ssh).
func (e *ActionExecutor) SSHRestartDevice(ctx context.Context, deviceIP, username, password string, sshPort int) error {
	return e.runSSHSession(ctx, deviceIP, username, password, sshPort, "reboot")
}

// SSHServiceRestart перезапускает сервис на устройстве по SSH (crypto/ssh).
func (e *ActionExecutor) SSHServiceRestart(ctx context.Context, deviceIP, username, password, serviceName string, sshPort int) error {
	remoteCmd := fmt.Sprintf("systemctl restart %s || service %s restart", serviceName, serviceName)
	return e.runSSHSession(ctx, deviceIP, username, password, sshPort, remoteCmd)
}

// runSSHSession выполняет команду через нативный SSH (crypto/ssh).
func (e *ActionExecutor) runSSHSession(ctx context.Context, host, username, password string, port int, remoteCmd string) error {
	if port == 0 {
		port = 22
	}

	config := sshClientConfig(username, password, 10*time.Second)
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		e.logger.Error("ssh dial failed", "host", addr, "error", err)
		return fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("ssh new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(remoteCmd); err != nil {
		e.logger.Error("ssh command failed",
			"host", host,
			"command", remoteCmd,
			"stdout", stdout.String(),
			"stderr", stderr.String(),
			"error", err,
		)
		return fmt.Errorf("ssh command failed: %w (stderr: %s)", err, stderr.String())
	}

	e.logger.Info("ssh command succeeded", "host", host, "command", remoteCmd, "output", stdout.String())
	return nil
}

// ── ONVIF SOAP Templates ───────────────────────────────────────────

const onvifRebootTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Header>
    <Security xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
      <UsernameToken>
        <Username>%s</Username>
        <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
      </UsernameToken>
    </Security>
  </s:Header>
  <s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <SystemReboot xmlns="http://www.onvif.org/ver10/device/wsdl"/>
  </s:Body>
</s:Envelope>`

const onvifPTZHomeTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Header>
    <Security xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
      <UsernameToken>
        <Username>%s</Username>
        <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">%s</Password>
      </UsernameToken>
    </Security>
  </s:Header>
  <s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <GotoHomePosition xmlns="http://www.onvif.org/ver20/ptz/wsdl">
      <ProfileToken>default</ProfileToken>
    </GotoHomePosition>
  </s:Body>
</s:Envelope>`

// ── XML helpers ────────────────────────────────────────────────────

// ISAPIRebootResponse XML структура.
type ISAPIRebootResponse struct {
	XMLName      xml.Name `xml:"ResponseStatus"`
	StatusCode   int      `xml:"statusCode"`
	StatusString string   `xml:"statusString"`
}
