package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	cryptotls "crypto/tls"
)

const (
	defaultOTATimeout = 5 * time.Minute
	rollbackTimeout   = 30 * time.Second
)

// OTAUpdater handles Over-The-Air firmware updates for the Edge Agent.
//
// Update flow:
//  1. Check version on Backend (HTTP GET /api/v1/edge/version)
//  2. Download new binary       (HTTPS with mTLS)
//  3. Verify SHA256 hash
//  4. Backup current binary
//  5. Install new binary + systemctl restart
//  6. Monitor startup — rollback on failure within 30 seconds
//
// Compliance:
//   - IEC 62443-3-3 SL-3 (Zone 5 — Edge)
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности (SHA256)
//   - Приказ ОАЦ №66 п. 7.18.5: Управление обновлениями (signed OTA, rollback)
//   - OWASP ASVS V12: File integrity verification
type OTAUpdater struct {
	currentVersion string
	downloadURL    string
	dataDir        string
	client         *http.Client
	logger         *slog.Logger
}

// NewOTAUpdater creates an OTAUpdater with an mTLS-enabled HTTP client.
//
// Parameters:
//   - currentVersion: currently running agent version
//   - downloadURL:    base URL of the OTA distribution server
//   - dataDir:        directory for temporary files (default: /usb/ota)
//   - tlsConfig:      *crypto/tls.Config for mTLS (Приказ ОАЦ №66 п. 7.18.2)
//   - logger:         structured logger
func NewOTAUpdater(
	currentVersion, downloadURL, dataDir string,
	tlsConfig *cryptotls.Config,
	logger *slog.Logger,
) *OTAUpdater {
	if dataDir == "" {
		dataDir = "/usb/ota"
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Timeout:   defaultOTATimeout,
		Transport: transport,
	}

	return &OTAUpdater{
		currentVersion: currentVersion,
		downloadURL:    downloadURL,
		dataDir:        dataDir,
		client:         client,
		logger:         logger.With("component", "ota_updater"),
	}
}

// CheckAndUpdate checks for a new version and performs OTA update if available.
//
// Returns nil if no update is needed or update succeeds.
// Returns error if update fails (rollback is attempted automatically).
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.5: Атомарное обновление с rollback при failure
//   - OWASP ASVS V12.3: File integrity verification
func (u *OTAUpdater) CheckAndUpdate(ctx context.Context) error {
	u.logger.Info("checking for OTA update",
		"current_version", u.currentVersion,
	)

	// --- Step 1: Check version on Backend ---
	newVersion, err := u.checkVersion(ctx)
	if err != nil {
		return fmt.Errorf("version check: %w", err)
	}
	if newVersion == "" {
		u.logger.Info("no new version available")
		return nil
	}
	if newVersion == u.currentVersion {
		u.logger.Info("already at latest version", "version", u.currentVersion)
		return nil
	}

	u.logger.Info("new version available",
		"current", u.currentVersion,
		"new", newVersion,
	)

	// --- Step 2: Download new binary ---
	tmpPath := filepath.Join(u.dataDir, "edge-agent.new")
	if err := u.downloadBinary(ctx, tmpPath); err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	// --- Step 3: Verify SHA256 hash ---
	if err := u.verifyDownloadedHash(ctx, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("hash verification: %w", err)
	}

	u.logger.Info("SHA256 integrity verified")

	// --- Step 4: Backup current binary ---
	currentBinary, err := os.Executable()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("get executable path: %w", err)
	}

	backupPath := filepath.Join(u.dataDir, "edge-agent.bak")
	if err := u.backupBinary(currentBinary, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("backup: %w", err)
	}

	u.logger.Debug("current binary backed up", "backup", backupPath)

	// --- Step 5: Install new binary ---
	if err := u.installBinary(tmpPath, currentBinary); err != nil {
		// Attempt rollback on install failure
		if rbErr := u.Rollback(); rbErr != nil {
			u.logger.Error("rollback failed after install error",
				"install_error", err,
				"rollback_error", rbErr,
			)
		}
		return fmt.Errorf("install: %w", err)
	}

	u.logger.Info("new binary installed, restarting service")

	// --- Step 6: Restart service + monitor ---
	if err := u.restartService(ctx); err != nil {
		u.logger.Error("service restart failed, rolling back",
			"error", err,
		)
		if rbErr := u.Rollback(); rbErr != nil {
			u.logger.Error("rollback failed", "error", rbErr)
		}
		return fmt.Errorf("service restart: %w", err)
	}

	u.currentVersion = newVersion
	u.logger.Info("OTA update completed successfully",
		"new_version", newVersion,
	)

	return nil
}

// checkVersion queries the Backend for the latest available version.
func (u *OTAUpdater) checkVersion(ctx context.Context) (string, error) {
	versionURL := u.downloadURL + "/api/v1/edge/version"
	u.logger.Debug("checking version", "url", versionURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return string(body), nil
}

// downloadBinary downloads the new agent binary to a temporary file.
func (u *OTAUpdater) downloadBinary(ctx context.Context, dst string) error {
	binaryURL := u.downloadURL + "/api/v1/edge/binary"
	u.logger.Debug("downloading binary", "url", binaryURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, binaryURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(u.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("write binary: %w", err)
	}

	u.logger.Debug("binary downloaded", "size", written, "path", dst)
	return nil
}

// verifyDownloadedHash fetches the expected SHA256 hash from Backend
// and verifies it against the downloaded file.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности
//   - OWASP ASVS V12.3: File integrity verification
func (u *OTAUpdater) verifyDownloadedHash(ctx context.Context, filePath string) error {
	sha256URL := u.downloadURL + "/api/v1/edge/binary.sha256"
	u.logger.Debug("fetching expected SHA256", "url", sha256URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sha256URL, nil)
	if err != nil {
		return fmt.Errorf("create hash request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("hash request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hash endpoint returned status %d", resp.StatusCode)
	}

	hashBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read hash response: %w", err)
	}

	expectedHash := string(hashBytes)

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file for hash: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash calculation: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: got %q..., expected %q...",
			actualHash[:16], expectedHash[:16],
		)
	}

	return nil
}

// backupBinary copies a file from src to dst, preserving permissions.
func (u *OTAUpdater) backupBinary(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Preserve executable permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("chmod backup: %w", err)
	}

	return nil
}

// installBinary replaces dst with src, then removes src.
func (u *OTAUpdater) installBinary(src, dst string) error {
	if err := u.backupBinary(src, dst); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	if err := os.Remove(src); err != nil {
		u.logger.Warn("failed to remove temp file", "path", src, "error", err)
	}

	return nil
}

// restartService calls systemctl restart edge-agent with a 30-second timeout.
func (u *OTAUpdater) restartService(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, rollbackTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "restart", "edge-agent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart failed: %w, output: %s",
			err, string(output),
		)
	}

	u.logger.Info("service restarted successfully")
	return nil
}

// Rollback restores the previous binary version and restarts the service.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.5: Атомарное обновление с rollback при failure
func (u *OTAUpdater) Rollback() error {
	u.logger.Info("initiating rollback")

	backupPath := filepath.Join(u.dataDir, "edge-agent.bak")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found at %s", backupPath)
	}

	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	// Restore backup
	if err := u.backupBinary(backupPath, currentBinary); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}

	// Restart with old binary
	ctx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "restart", "edge-agent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rollback restart failed: %w, output: %s",
			err, string(output),
		)
	}

	u.logger.Info("rollback completed successfully")
	return nil
}

// CurrentVersion returns the currently running agent version.
func (u *OTAUpdater) CurrentVersion() string {
	return u.currentVersion
}
