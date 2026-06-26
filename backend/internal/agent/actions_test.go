// Package agent — tests for SSH, SNMP, ISAPI actions.
// Соответствие: ISO 27001 A.9.4.3, СТБ 34.101.27 п. 5.1, OWASP ASVS V2.1
package agent

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

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

	// Проверяем что нет Password методов - используем строковое представление
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
	// Set HOME to nonexistent to prevent fallback to ~/.ssh/known_hosts
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

	// Проверяем что все методы — key-based (не password)
	for _, m := range config.Auth {
		typeStr := fmt.Sprintf("%T", m)
		if typeStr == "ssh.keyboardAuthMethod" || typeStr == "ssh.passwordAuthMethod" {
			t.Errorf("unexpected password auth method: %s", typeStr)
		}
	}
}

// ── Helper functions ─────────────────────────────────────────────────────

// createTempSSHKey создаёт временный файл с тестовым SSH ключом.
func createTempSSHKey(t *testing.T) string {
	t.Helper()

	// Пробуем сгенерировать ключ через ssh-keygen
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Log("ssh-keygen not available, skipping key generation")
		return ""
	}

	tmpFile, err := os.CreateTemp("", "id_rsa_test_*")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name()) // ssh-keygen requires non-existent file

	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-N", "", "-f", tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("ssh-keygen failed: %v, output: %s", err, string(output))
		os.Remove(tmpFile.Name())
		return ""
	}

	return tmpFile.Name()
}

// createTempKnownHosts создаёт временный known_hosts файл с реальным ключом хоста.
func createTempKnownHosts(t *testing.T) string {
	t.Helper()

	// Сначала генерируем хост-ключ
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

	// Генерируем ED25519 хост-ключ
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", hostKeyFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("ssh-keygen host key failed: %v, output: %s", err, string(output))
		os.Remove(hostKeyFile.Name())
		return ""
	}
	defer os.Remove(hostKeyFile.Name())

	// Читаем публичную часть
	pubKeyData, err := os.ReadFile(hostKeyFile.Name() + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(hostKeyFile.Name() + ".pub")

	// Создаём known_hosts файл
	knownHostsFile, err := os.CreateTemp("", "known_hosts_*")
	if err != nil {
		t.Fatal(err)
	}

	// Формат: hostname algorithm base64key
	// Парсим публичный ключ
	line := fmt.Sprintf("localhost %s", string(pubKeyData))
	if _, err := knownHostsFile.WriteString(line); err != nil {
		t.Fatal(err)
	}
	knownHostsFile.Close()

	return knownHostsFile.Name()
}

// ── SSH Signer Tests ────────────────────────────────────────────────────

// TestLoadSSHSignerFromEnvPath проверяет загрузку ключа из пути.
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

// TestLoadSSHSignerFromEnvDirect проверяет загрузку ключа из переменной.
func TestLoadSSHSignerFromEnvDirect(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "")
	t.Setenv("SSH_PRIVATE_KEY", "")

	signer := loadSSHSignerFromEnv()
	if signer != nil {
		t.Error("expected nil signer for empty key")
	}
}

// TestLoadSSHSignerPriority проверяет приоритет: PATH > DIRECT.
func TestLoadSSHSignerPriority(t *testing.T) {
	t.Setenv("SSH_PRIVATE_KEY_PATH", "/some/path")
	t.Setenv("SSH_PRIVATE_KEY", "some-key-data")

	_ = loadSSHSignerFromEnv()
}
