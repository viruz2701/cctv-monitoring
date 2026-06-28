// Package agent — tests for SSH, SNMP, ISAPI, ONVIF actions.
// Соответствие: ISO 27001 A.9.4.3, СТБ 34.101.27 п. 5.1, OWASP ASVS V2.1
// IEC 62443 SR 7.1 — Remote access security, audit trail verification
package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ── Test Helpers ──────────────────────────────────────────────────────────

// testActionExecutor creates an ActionExecutor with test-friendly defaults.
func testActionExecutor(opts ...func(*ActionExecutor)) *ActionExecutor {
	e := NewActionExecutor("", "", slog.Default())
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// withP2PGateway configures P2P gateway for tests.
func withP2PGateway(url, apiKey string) func(*ActionExecutor) {
	return func(e *ActionExecutor) {
		e.p2pGatewayURL = url
		e.p2pAPIKey = apiKey
	}
}

// extractHost returns host:port from an httptest server URL (strips "http://").
func extractHost(serverURL string) string {
	return strings.TrimPrefix(serverURL, "http://")
}

// ── S1-01 Compliance Tests: Password Management ──────────────────────────

// TestBuildSSHAuthMethodsNoPassword проверяет, что не создаются password-методы.
func TestBuildSSHAuthMethodsNoPassword(t *testing.T) {
	tmpKey := createTempSSHKey(t)
	if tmpKey != "" {
		defer os.Remove(tmpKey)
		t.Setenv("SSH_PRIVATE_KEY_PATH", tmpKey)
	}

	methods := buildSSHAuthMethods()

	if methods == nil {
		t.Fatal("buildSSHAuthMethods returned nil")
	}

	for _, m := range methods {
		typeStr := fmt.Sprintf("%T", m)
		if typeStr == "ssh.keyboardAuthMethod" || typeStr == "ssh.passwordAuthMethod" {
			t.Errorf("unexpected password auth method: %s", typeStr)
		}
	}
}

// TestBuildSSHAuthMethodsEmptyWithoutKey проверяет, что без ключа нет методов.
func TestBuildSSHAuthMethodsEmptyWithoutKey(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")

	methods := buildSSHAuthMethods()

	if methods == nil {
		t.Fatal("buildSSHAuthMethods returned nil, expected empty slice")
	}
	if len(methods) != 0 {
		t.Errorf("expected 0 methods without key, got %d", len(methods))
	}
}

// TestNewHostKeyCallback проверяет host key verification.
func TestNewHostKeyCallback(t *testing.T) {
	t.Setenv("SSH_KNOWN_HOSTS_PATH", "")
	t.Setenv("HOME", "/nonexistent-home-dir-for-test")

	callback, err := newHostKeyCallback()
	if err == nil {
		t.Error("expected error without known_hosts, got nil callback")
	}
	if callback != nil {
		t.Error("expected nil callback on error")
	}
}

// TestSSHClientConfigNoInsecureIgnore проверяет, что InsecureIgnoreHostKey не используется.
func TestSSHClientConfigNoInsecureIgnore(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_KNOWN_HOSTS_PATH", "/nonexistent/known_hosts")

	config, err := sshClientConfig("testuser", 10)
	if err == nil {
		t.Error("expected error with nonexistent known_hosts (Fail Secure)")
	}
	if config != nil {
		t.Error("expected nil config on error (no InsecureIgnoreHostKey fallback)")
	}
}

// TestSSHClientConfigHostKeyCallbackType проверяет наличие host key callback.
func TestSSHClientConfigHostKeyCallbackType(t *testing.T) {
	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	config, err := sshClientConfig("testuser", 10)
	if err != nil {
		t.Fatalf("sshClientConfig error: %v", err)
	}
	if config == nil {
		t.Fatal("sshClientConfig returned nil")
	}
	if config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback is nil, expected known_hosts verification")
	}
}

// TestSSHClientConfigAuthMethods проверяет что методы аутентификации — key-based.
func TestSSHClientConfigAuthMethods(t *testing.T) {
	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	config, err := sshClientConfig("testuser", 10)
	if err != nil {
		t.Fatalf("sshClientConfig error: %v", err)
	}

	for _, m := range config.Auth {
		typeStr := fmt.Sprintf("%T", m)
		if typeStr == "ssh.keyboardAuthMethod" || typeStr == "ssh.passwordAuthMethod" {
			t.Errorf("unexpected password auth method: %s", typeStr)
		}
	}
}

// ── ISAPI Tests ───────────────────────────────────────────────────────────
// IEC 62443 SR 7.1 — Remote access must verify device response
// OWASP ASVS V3.1 — Session management for device APIs

func TestActionExecutor_ISAPIReboot_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/ISAPI/System/reboot" {
			t.Errorf("expected /ISAPI/System/reboot, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/xml" {
			t.Errorf("expected application/xml Content-Type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReboot(context.Background(), host, "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ISAPIReboot_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReboot(context.Background(), host, "admin", "password")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got: %v", err)
	}
}

func TestActionExecutor_ISAPIReboot_ContextTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReboot(ctx, host, "admin", "password")
	if err == nil {
		t.Fatal("expected context deadline exceeded error")
	}
}

func TestActionExecutor_ISAPIReboot_ContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReboot(ctx, host, "admin", "password")
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestActionExecutor_ISAPIReset_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ISAPI/System/factoryReset" {
			t.Errorf("expected /ISAPI/System/factoryReset, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("mode") != "basic" {
			t.Errorf("expected mode=basic, got %s", r.URL.Query().Get("mode"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReset(context.Background(), host, "admin", "password", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ISAPIReset_FullMode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") != "full" {
			t.Errorf("expected mode=full, got %s", r.URL.Query().Get("mode"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReset(context.Background(), host, "admin", "password", "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ISAPIReset_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "access denied")
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIReset(context.Background(), host, "admin", "password", "basic")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in error, got: %v", err)
	}
}

func TestActionExecutor_ISAPIRestore_Success(t *testing.T) {
	t.Parallel()

	configXML := []byte("<Configuration><param>value</param></Configuration>")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/ISAPI/System/configurationData" {
			t.Errorf("expected /ISAPI/System/configurationData, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("mode") != "restore" {
			t.Errorf("expected mode=restore, got %s", r.URL.Query().Get("mode"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != string(configXML) {
			t.Errorf("expected body %s, got %s", string(configXML), string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIRestore(context.Background(), host, "admin", "password", configXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ISAPIRestore_EmptyBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ISAPIRestore(context.Background(), host, "admin", "password", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── P2P Gateway Tests ─────────────────────────────────────────────────────
// IEC 62443 SR 2.1 — Authorization for P2P gateway access

func TestActionExecutor_ISAPIRebootViaP2P_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/device/command" {
			t.Errorf("expected /api/v1/device/command, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("expected X-API-Key=test-api-key, got %s", r.Header.Get("X-API-Key"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"serial":"SN12345"`) {
			t.Errorf("expected serial SN12345 in body, got %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	e := testActionExecutor(withP2PGateway(server.URL, "test-api-key"))

	err := e.ISAPIRebootViaP2P(context.Background(), "SN12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ISAPIRebootViaP2P_NoGateway(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()

	err := e.ISAPIRebootViaP2P(context.Background(), "SN12345")
	if err == nil {
		t.Fatal("expected error for empty gateway URL")
	}
	if !strings.Contains(err.Error(), "gateway URL not configured") {
		t.Errorf("expected 'gateway URL not configured', got: %v", err)
	}
}

func TestActionExecutor_ISAPIRebootViaP2P_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"error":"upstream timeout"}`)
	}))
	defer server.Close()

	e := testActionExecutor(withP2PGateway(server.URL, "test-api-key"))

	err := e.ISAPIRebootViaP2P(context.Background(), "SN12345")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("expected 502 in error, got: %v", err)
	}
}

func TestActionExecutor_ISAPIRebootViaP2P_ContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := testActionExecutor(withP2PGateway(server.URL, "test-api-key"))

	err := e.ISAPIRebootViaP2P(ctx, "SN12345")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ── ONVIF Tests ───────────────────────────────────────────────────────────
// IEC 62443 SR 3.1 — SOAP message integrity verification

func TestActionExecutor_ONVIFReboot_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/onvif/device_service" {
			t.Errorf("expected /onvif/device_service, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/soap+xml; charset=utf-8" {
			t.Errorf("expected SOAP content type, got %s", r.Header.Get("Content-Type"))
		}
		if !strings.Contains(r.Header.Get("SOAPAction"), "SystemReboot") {
			t.Errorf("expected SOAPAction SystemReboot, got %s", r.Header.Get("SOAPAction"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <SystemRebootResponse xmlns="http://www.onvif.org/ver10/device/wsdl">
      <Message>Rebooting...</Message>
    </SystemRebootResponse>
  </s:Body>
</s:Envelope>`)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFReboot(context.Background(), host, "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ONVIFReboot_SOAPFault(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
		<SOAP-ENV:Body>
		  <SOAP-ENV:Fault>
		    <faultstring>Action not supported</faultstring>
		  </SOAP-ENV:Fault>
		</SOAP-ENV:Body>
</SOAP-ENV:Envelope>`)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFReboot(context.Background(), host, "admin", "password")
	if err == nil {
		t.Fatal("expected error for SOAP Fault")
	}
	if !strings.Contains(err.Error(), "Action not supported") {
		t.Errorf("expected 'Action not supported' in error, got: %v", err)
	}
}

func TestActionExecutor_ONVIFReboot_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFReboot(context.Background(), host, "admin", "wrong-password")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

func TestActionExecutor_ONVIFPTZHome_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GotoHomePositionResponse xmlns="http://www.onvif.org/ver20/ptz/wsdl"/>
  </s:Body>
</s:Envelope>`)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFPTZHome(context.Background(), host, "admin", "password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActionExecutor_ONVIFPTZHome_SOAPFault(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
		<SOAP-ENV:Body>
		  <SOAP-ENV:Fault>
		    <faultstring>PTZ not available on this device</faultstring>
		  </SOAP-ENV:Fault>
		</SOAP-ENV:Body>
</SOAP-ENV:Envelope>`)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFPTZHome(context.Background(), host, "admin", "password")
	if err == nil {
		t.Fatal("expected error for SOAP Fault")
	}
	if !strings.Contains(err.Error(), "PTZ not available") {
		t.Errorf("expected 'PTZ not available' in error, got: %v", err)
	}
}

func TestActionExecutor_ONVIFPTZHome_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "Service Unavailable")
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)

	err := e.ONVIFPTZHome(context.Background(), host, "admin", "password")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected 503 in error, got: %v", err)
	}
}

// ── SNMP Tests ────────────────────────────────────────────────────────────
// IEC 62443 SR 5.1 — Network segmentation, SNMP v3 security

func TestActionExecutor_SNMPReset_InvalidConfig(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()

	err := e.SNMPReset(context.Background(), "192.0.2.1", SNMPActionConfig{})
	if err == nil {
		t.Fatal("expected error for invalid SNMP config")
	}
	t.Logf("SNMP error with empty config: %v", err)
}

func TestActionExecutor_SNMPReset_DefaultConfig(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()

	cfg := DefaultSNMPConfig()
	cfg.TimeoutSec = 1
	cfg.Retries = 0

	err := e.SNMPReset(context.Background(), "192.0.2.1", cfg)
	if err == nil {
		t.Fatal("expected error for unreachable device")
	}
	if !strings.Contains(err.Error(), "snmp") {
		t.Errorf("expected 'snmp' in error, got: %v", err)
	}
}

func TestActionExecutor_SNMPReset_ContextCancelled(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()
	cfg := DefaultSNMPConfig()
	cfg.TimeoutSec = 1
	cfg.Retries = 0
	// SNMP doesn't check context during Connect/Set,
	// so the request will still attempt but time out quickly.

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := e.SNMPReset(ctx, "192.0.2.1", cfg)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestActionExecutor_SNMPReset_CustomOID(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()
	cfg := DefaultSNMPConfig()
	cfg.TimeoutSec = 1
	cfg.Retries = 0
	cfg.RebootOID = "1.3.6.1.4.1.1004849.2.1.2.99.0"

	err := e.SNMPReset(context.Background(), "192.0.2.1", cfg)
	if err == nil {
		t.Fatal("expected error for unreachable device")
	}
	t.Logf("SNMP error with custom OID: %v", err)
}

func TestActionExecutor_SNMPReset_V3Config(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()
	cfg := DefaultSNMPConfig()
	cfg.Version = "3"
	cfg.Username = "snmpuser"
	cfg.AuthPassword = "authpass"
	cfg.PrivPassword = "privpass"
	cfg.TimeoutSec = 1
	cfg.Retries = 0

	err := e.SNMPReset(context.Background(), "192.0.2.1", cfg)
	if err == nil {
		t.Fatal("expected error for unreachable device")
	}
	t.Logf("SNMP v3 error: %v", err)
}

func TestActionExecutor_SNMPColdStart_InvalidConfig(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()

	err := e.SNMPColdStart(context.Background(), "192.0.2.1", SNMPActionConfig{})
	if err == nil {
		t.Fatal("expected error for invalid SNMP config")
	}
	t.Logf("SNMPColdStart error: %v", err)
}

func TestActionExecutor_SNMPColdStart_ContextCancelled(t *testing.T) {
	t.Parallel()

	e := testActionExecutor()
	cfg := DefaultSNMPConfig()
	cfg.TimeoutSec = 1
	cfg.Retries = 0

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := e.SNMPColdStart(ctx, "192.0.2.1", cfg)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ── SSH Action Tests (enhancement) ────────────────────────────────────────
// ISO 27001 A.9.4.3 — Key-based authentication only
// СТБ 34.101.27 п. 5.1 — Fail Secure для host key verification

func TestActionExecutor_SSHRestartDevice_NoConfig(t *testing.T) {

	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")
	t.Setenv("SSH_KNOWN_HOSTS_PATH", "")
	t.Setenv("HOME", "/nonexistent-home-for-test")

	e := testActionExecutor()

	err := e.SSHRestartDevice(context.Background(), "192.0.2.1", "testuser", 22)
	if err == nil {
		t.Fatal("expected error without SSH config")
	}
	if !strings.Contains(err.Error(), "ssh config") {
		t.Errorf("expected 'ssh config' in error, got: %v", err)
	}
}

func TestActionExecutor_SSHServiceRestart_NoConfig(t *testing.T) {

	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")
	t.Setenv("SSH_KNOWN_HOSTS_PATH", "")
	t.Setenv("HOME", "/nonexistent-home-for-test")

	e := testActionExecutor()

	err := e.SSHServiceRestart(context.Background(), "192.0.2.1", "testuser", "nginx", 22)
	if err == nil {
		t.Fatal("expected error without SSH config")
	}
	if !strings.Contains(err.Error(), "ssh config") {
		t.Errorf("expected 'ssh config' in error, got: %v", err)
	}
}

func TestActionExecutor_SSHRestartDevice_DefaultPort(t *testing.T) {

	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")
	t.Setenv("SSH_KNOWN_HOSTS_PATH", "")
	t.Setenv("HOME", "/nonexistent-home-for-test")

	e := testActionExecutor()

	err := e.SSHRestartDevice(context.Background(), "192.0.2.1", "testuser", 0)
	if err == nil {
		t.Fatal("expected error without SSH config")
	}
}

func TestActionExecutor_SSHRestartDevice_DialError(t *testing.T) {

	tmpKey := createTempSSHKey(t)
	if tmpKey == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKey)

	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("cannot create known_hosts, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_PRIVATE_KEY_PATH", tmpKey)
	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	e := testActionExecutor()

	err := e.SSHRestartDevice(context.Background(), "192.0.2.1", "testuser", 22)
	if err == nil {
		t.Fatal("expected dial error for unreachable host")
	}
	t.Logf("SSH dial error: %v", err)
}

func TestActionExecutor_SSHServiceRestart_DialError(t *testing.T) {

	tmpKey := createTempSSHKey(t)
	if tmpKey == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKey)

	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("cannot create known_hosts, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_PRIVATE_KEY_PATH", tmpKey)
	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	e := testActionExecutor()

	err := e.SSHServiceRestart(context.Background(), "192.0.2.1", "testuser", "nginx", 22)
	if err == nil {
		t.Fatal("expected dial error for unreachable host")
	}
	t.Logf("SSH service restart dial error: %v", err)
}

// ── SSH Client Config Tests (enhancement) ─────────────────────────────────
// ISO 27001 A.9.4.3 — Secure key management

func TestSSHClientConfig_Timeout(t *testing.T) {

	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	config, err := sshClientConfig("testuser", 30*time.Second)
	if err != nil {
		t.Fatalf("sshClientConfig error: %v", err)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}
}

func TestSSHClientConfig_ZeroTimeout(t *testing.T) {

	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	config, err := sshClientConfig("testuser", 0)
	if err != nil {
		t.Fatalf("sshClientConfig error: %v", err)
	}
	if config.Timeout != 0 {
		t.Errorf("expected zero timeout, got %v", config.Timeout)
	}
}

func TestSSHClientConfig_EmptyUsername(t *testing.T) {

	tmpKnownHosts := createTempKnownHosts(t)
	if tmpKnownHosts == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKnownHosts)

	t.Setenv("SSH_KNOWN_HOSTS_PATH", tmpKnownHosts)

	config, err := sshClientConfig("", 10*time.Second)
	if err != nil {
		t.Fatalf("sshClientConfig error: %v", err)
	}
	if config.User != "" {
		t.Errorf("expected empty username, got %s", config.User)
	}
}

// ── SSH Signer Tests ──────────────────────────────────────────────────────

func TestLoadSSHSignerFromEnvPath(t *testing.T) {
	tmpKey := createTempSSHKey(t)
	if tmpKey == "" {
		t.Skip("ssh-keygen not available, skipping")
	}
	defer os.Remove(tmpKey)

	t.Setenv("SSH_PRIVATE_KEY_PATH", tmpKey)
	t.Setenv("SSH_PRIVATE_KEY", "")

	signer := loadSSHSignerFromEnv()
	t.Logf("SSH signer loaded: %v", signer != nil)
}

func TestLoadSSHSignerFromEnvDirect(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")

	signer := loadSSHSignerFromEnv()
	if signer != nil {
		t.Error("expected nil signer for empty key")
	}
}

func TestLoadSSHSignerPriority(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "/some/path")
	t.Setenv("SSH_PRIVATE_KEY", "some-key-data")

	_ = loadSSHSignerFromEnv()
}

// ── Table-Driven: ISAPI Multi-Action ──────────────────────────────────────
// IEC 62443 SR 7.1 — Comprehensive testing of remote actions

func TestActionExecutor_ISAPIActionsTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(ctx context.Context, e *ActionExecutor, host string) error
		wantErr bool
	}{
		{
			name: "reboot success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ISAPIReboot(ctx, host, "admin", "pass")
			},
			wantErr: false,
		},
		{
			name: "reset (basic) success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ISAPIReset(ctx, host, "admin", "pass", "basic")
			},
			wantErr: false,
		},
		{
			name: "reset (full) success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ISAPIReset(ctx, host, "admin", "pass", "full")
			},
			wantErr: false,
		},
		{
			name: "restore success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ISAPIRestore(ctx, host, "admin", "pass", []byte("<cfg/>"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			e := testActionExecutor()
			host := extractHost(server.URL)

			err := tt.run(context.Background(), e, host)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestActionExecutor_ONVIFActionsTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(ctx context.Context, e *ActionExecutor, host string) error
		wantErr bool
	}{
		{
			name: "reboot success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ONVIFReboot(ctx, host, "admin", "pass")
			},
			wantErr: false,
		},
		{
			name: "ptz home success",
			run: func(ctx context.Context, e *ActionExecutor, host string) error {
				return e.ONVIFPTZHome(ctx, host, "admin", "pass")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <Response xmlns="http://www.onvif.org/ver10/device/wsdl"/>
  </s:Body>
</s:Envelope>`)
			}))
			defer server.Close()

			e := testActionExecutor()
			host := extractHost(server.URL)

			err := tt.run(context.Background(), e, host)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ── extractFaultString Tests ─────────────────────────────────────────────

func TestExtractFaultString_Standard(t *testing.T) {
	t.Parallel()

	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
	 <SOAP-ENV:Body>
	   <SOAP-ENV:Fault>
	     <faultstring>Action not supported by this device</faultstring>
	   </SOAP-ENV:Fault>
	 </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`)

	result := extractFaultString(body)
	if result != "Action not supported by this device" {
		t.Errorf("expected 'Action not supported by this device', got %q", result)
	}
}

func TestExtractFaultString_WithSubcode(t *testing.T) {
	t.Parallel()

	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
	 <SOAP-ENV:Body>
	   <SOAP-ENV:Fault>
	     <faultstring>Operation timeout</faultstring>
	     <faultcode>env:Sender</faultcode>
	   </SOAP-ENV:Fault>
	 </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`)

	result := extractFaultString(body)
	if result != "Operation timeout" {
		t.Errorf("expected 'Operation timeout', got %q", result)
	}
}

func TestExtractFaultString_NoFault(t *testing.T) {
	t.Parallel()

	body := []byte(`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
	 <s:Body>
	   <SystemRebootResponse xmlns="http://www.onvif.org/ver10/device/wsdl"/>
	 </s:Body>
</s:Envelope>`)

	result := extractFaultString(body)
	if result != "unknown fault" {
		t.Errorf("expected 'unknown fault', got %q", result)
	}
}

func TestExtractFaultString_EmptyBody(t *testing.T) {
	t.Parallel()

	result := extractFaultString([]byte{})
	if result != "unknown fault" {
		t.Errorf("expected 'unknown fault', got %q", result)
	}
}

func TestExtractFaultString_NoFaultstringTag(t *testing.T) {
	t.Parallel()

	body := []byte(`<SOAP-ENV:Fault><faultcode>env:Server</faultcode></SOAP-ENV:Fault>`)
	result := extractFaultString(body)
	if result != "unknown fault" {
		t.Errorf("expected 'unknown fault' when no faultstring tag, got %q", result)
	}
}

// ── DefaultSNMPConfig Tests ───────────────────────────────────────────────

func TestDefaultSNMPConfig_Values(t *testing.T) {
	t.Parallel()

	cfg := DefaultSNMPConfig()

	if cfg.Port != 161 {
		t.Errorf("expected Port=161, got %d", cfg.Port)
	}
	if cfg.Community != "public" {
		t.Errorf("expected Community='public', got %q", cfg.Community)
	}
	if cfg.Version != "2c" {
		t.Errorf("expected Version='2c', got %q", cfg.Version)
	}
	if cfg.TimeoutSec != 5 {
		t.Errorf("expected TimeoutSec=5, got %d", cfg.TimeoutSec)
	}
	if cfg.Retries != 2 {
		t.Errorf("expected Retries=2, got %d", cfg.Retries)
	}
	if cfg.RebootOID != "" {
		t.Errorf("expected empty RebootOID, got %q", cfg.RebootOID)
	}
}

func TestDefaultSNMPConfig_IndependentInstances(t *testing.T) {
	t.Parallel()

	cfg1 := DefaultSNMPConfig()
	cfg2 := DefaultSNMPConfig()

	cfg1.Port = 1161
	cfg1.Community = "private"

	if cfg2.Port == cfg1.Port {
		t.Error("cfg2.Port should not be affected by cfg1 mutation")
	}
	if cfg2.Community == cfg1.Community {
		t.Error("cfg2.Community should not be affected by cfg1 mutation")
	}
}

// ── Benchmark Tests ───────────────────────────────────────────────────────

func BenchmarkActionExecutor_New(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewActionExecutor("http://gateway:8080", "test-key", slog.Default())
	}
}

func BenchmarkActionExecutor_ISAPIReboot(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ISAPIReboot(ctx, host, "admin", "password")
	}
}

func BenchmarkActionExecutor_ONVIFReboot(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
	 <s:Body>
	   <SystemRebootResponse xmlns="http://www.onvif.org/ver10/device/wsdl"/>
	 </s:Body>
</s:Envelope>`)
	}))
	defer server.Close()

	e := testActionExecutor()
	host := extractHost(server.URL)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ONVIFReboot(ctx, host, "admin", "password")
	}
}

func BenchmarkActionExecutor_ISAPIRebootViaP2P(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	e := testActionExecutor(withP2PGateway(server.URL, "bench-api-key"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.ISAPIRebootViaP2P(ctx, "SN-BENCH-001")
	}
}

func BenchmarkExtractFaultString(b *testing.B) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
	 <SOAP-ENV:Body>
	   <SOAP-ENV:Fault>
	     <faultstring>Action not supported</faultstring>
	   </SOAP-ENV:Fault>
	 </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractFaultString(body)
	}
}

// ── Helper Functions ──────────────────────────────────────────────────────

// createTempSSHKey creates a temporary file with a test SSH key.
func createTempSSHKey(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Log("ssh-keygen not available, skipping key generation")
		return ""
	}

	tmpFile, err := os.CreateTemp("", "id_rsa_test_*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-N", "", "-f", tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("ssh-keygen failed: %v, output: %s", err, string(output))
		os.Remove(tmpFile.Name())
		return ""
	}

	return tmpFile.Name()
}

// createTempKnownHosts creates a temporary known_hosts file with a real host key.
func createTempKnownHosts(t *testing.T) string {
	t.Helper()

	hostKeyFile, err := os.CreateTemp("", "host_key_*")
	if err != nil {
		t.Fatal(err)
	}
	hostKeyFile.Close()
	os.Remove(hostKeyFile.Name())

	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Log("ssh-keygen not available, skipping")
		return ""
	}

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", hostKeyFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("ssh-keygen host key failed: %v, output: %s", err, string(output))
		os.Remove(hostKeyFile.Name())
		return ""
	}
	defer os.Remove(hostKeyFile.Name())

	pubKeyData, err := os.ReadFile(hostKeyFile.Name() + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(hostKeyFile.Name() + ".pub")

	knownHostsFile, err := os.CreateTemp("", "known_hosts_*")
	if err != nil {
		t.Fatal(err)
	}

	line := fmt.Sprintf("localhost %s", string(pubKeyData))
	if _, err := knownHostsFile.WriteString(line); err != nil {
		t.Fatal(err)
	}
	knownHostsFile.Close()

	return knownHostsFile.Name()
}
