package agent

import (
	"context"
	"crypto/ed25519"
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
	defaultOTATimeout  = 5 * time.Minute
	healthCheckTimeout = 30 * time.Second

	// Dual-boot A/B slots (Приказ ОАЦ №66 п. 7.18.3 — контроль целостности)
	slotA   = "/usr/local/bin/edge-agent.a"
	slotB   = "/usr/local/bin/edge-agent.b"
	symlink = "/usr/local/bin/edge-agent"
)

// EdgeAgentPublicKey is the Ed25519 public key embedded at build time via -ldflags.
// Format: hex-encoded 32-byte Ed25519 public key.
// Default is a test key — REPLACE in production with:
//
//	go build -ldflags '-X edge-agent/internal/agent.EdgeAgentPublicKey=<hex>'
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности (подпись bign/Ed25519)
//   - IEC 62443-3-3 SL-3: Signed firmware updates
var EdgeAgentPublicKey = "3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a3d3a" // test key — 32 zero bytes

// ed25519PublicKey is the decoded Ed25519 public key used for signature verification.
var ed25519PublicKey ed25519.PublicKey

func init() {
	decoded, err := hex.DecodeString(EdgeAgentPublicKey)
	if err != nil {
		// If the default test key fails, log but don't crash —
		// the agent will reject all updates until a valid key is embedded.
		ed25519PublicKey = make(ed25519.PublicKey, ed25519.PublicKeySize)
		return
	}
	if len(decoded) != ed25519.PublicKeySize {
		ed25519PublicKey = make(ed25519.PublicKey, ed25519.PublicKeySize)
		return
	}
	ed25519PublicKey = ed25519.PublicKey(decoded)
}

// OTAUpdater handles Over-The-Air firmware updates for the Edge Agent
// using dual-boot A/B partition approach.
//
// Architecture:
//
//	┌──────────────────────────────────────────────────────┐
//	│  /usr/local/bin/                                    │
//	│  ├── edge-agent.a       ← Slot A (binary)           │
//	│  ├── edge-agent.b       ← Slot B (binary)           │
//	│  └── edge-agent         ← Symlink → active slot     │
//	└──────────────────────────────────────────────────────┘
//
// Update flow:
//  1. Check version on Backend (HTTP GET /api/v1/edge/version)
//  2. Download new binary + .sig to inactive slot
//  3. Verify Ed25519 signature (embedded public key)
//  4. Atomic: switch symlink to inactive slot
//  5. systemctl restart edge-agent
//  6. Health check (30s timeout) — auto-rollback symlink on failure
//
// Rollback (no backup file):
//   - Switch symlink to the other slot (previous active version)
//   - No backup file needed — both slots preserve their binaries
//
// Compliance:
//   - IEC 62443-3-3 SL-3 (Zone 5 — Edge)
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности (Ed25519 signature)
//   - Приказ ОАЦ №66 п. 7.18.5: Управление обновлениями (signed OTA, rollback)
//   - OWASP ASVS V12: File integrity verification
//   - ISO 27001 A.12.4: Audit trail
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

// Slot identifies an A/B partition.
type Slot int

const (
	slotUnknown Slot = iota
	SlotA
	SlotB
)

// String returns the slot name.
func (s Slot) String() string {
	switch s {
	case SlotA:
		return "A"
	case SlotB:
		return "B"
	default:
		return "unknown"
	}
}

// Path returns the absolute binary path for the slot.
func (s Slot) Path() string {
	switch s {
	case SlotA:
		return slotA
	case SlotB:
		return slotB
	default:
		return ""
	}
}

// CurrentSlot determines which slot is currently active by following the symlink.
func CurrentSlot() Slot {
	target, err := os.Readlink(symlink)
	if err != nil {
		return slotUnknown
	}

	abs, err := filepath.Abs(target)
	if err != nil {
		return slotUnknown
	}

	switch abs {
	case slotA:
		return SlotA
	case slotB:
		return SlotB
	default:
		return slotUnknown
	}
}

// InactiveSlot returns the slot that is NOT currently active.
// If no slot is active, defaults to SlotA.
func InactiveSlot() Slot {
	switch CurrentSlot() {
	case SlotA:
		return SlotB
	case SlotB:
		return SlotA
	default:
		// No active slot — try to determine which has a valid binary.
		// Default to Slot A as target.
		return SlotA
	}
}

// activeSymlinkTarget returns the absolute path the symlink should point to for a given slot.
func activeSymlinkTarget(slot Slot) string {
	return slot.Path()
}

// switchSymlink atomically updates the symlink to point to the given slot.
// Uses a temporary symlink + rename for atomicity.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.5: Атомарное обновление
func switchSymlink(slot Slot) error {
	target := activeSymlinkTarget(slot)
	if target == "" {
		return fmt.Errorf("invalid slot: %v", slot)
	}

	// Verify target binary exists
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("slot binary not found: %s", target)
	}

	// Atomic: create temp symlink, then rename over the real one
	tmpSymlink := symlink + ".tmp"
	if err := os.Remove(tmpSymlink); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old tmp symlink: %w", err)
	}

	if err := os.Symlink(target, tmpSymlink); err != nil {
		return fmt.Errorf("create temp symlink: %w", err)
	}

	if err := os.Rename(tmpSymlink, symlink); err != nil {
		// Cleanup temp on failure
		os.Remove(tmpSymlink)
		return fmt.Errorf("atomic rename symlink: %w", err)
	}

	return nil
}

// CheckAndUpdate checks for a new version and performs OTA update if available.
//
// Returns nil if no update is needed or update succeeds.
// Returns error if update fails (rollback is attempted automatically).
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3: Ed25519 signature verification
//   - Приказ ОАЦ №66 п. 7.18.5: Атомарное обновление с rollback при failure
//   - OWASP ASVS V12.3: File integrity verification
func (u *OTAUpdater) CheckAndUpdate(ctx context.Context) error {
	u.logger.Info("checking for OTA update",
		"current_version", u.currentVersion,
		"active_slot", CurrentSlot().String(),
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

	inactiveSlot := InactiveSlot()
	u.logger.Info("new version available, updating inactive slot",
		"current", u.currentVersion,
		"new", newVersion,
		"inactive_slot", inactiveSlot.String(),
	)

	// --- Step 2: Download binary + signature to inactive slot ---
	binaryPath := inactiveSlot.Path()
	sigPath := binaryPath + ".sig"

	u.logger.Debug("downloading binary", "slot", inactiveSlot.String(), "path", binaryPath)
	if err := u.downloadFile(ctx, "/api/v1/edge/binary", binaryPath); err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	u.logger.Debug("downloading signature", "path", sigPath)
	if err := u.downloadFile(ctx, "/api/v1/edge/binary.sig", sigPath); err != nil {
		os.Remove(binaryPath)
		return fmt.Errorf("download signature: %w", err)
	}

	// --- Step 3: Verify Ed25519 signature ---
	if err := u.verifyEd25519Signature(binaryPath, sigPath); err != nil {
		os.Remove(binaryPath)
		os.Remove(sigPath)
		return fmt.Errorf("Ed25519 signature verification: %w", err)
	}

	u.logger.Info("Ed25519 signature verified",
		"slot", inactiveSlot.String(),
	)

	// --- Step 4: Atomic symlink switch ---
	previousSlot := CurrentSlot()
	u.logger.Debug("switching symlink",
		"from", previousSlot.String(),
		"to", inactiveSlot.String(),
	)

	if err := switchSymlink(inactiveSlot); err != nil {
		// Clean up the new binary — we didn't switch
		os.Remove(binaryPath)
		os.Remove(sigPath)
		return fmt.Errorf("symlink switch: %w", err)
	}

	u.logger.Info("symlink switched, restarting service")

	// --- Step 5: Restart service ---
	if err := u.restartService(ctx); err != nil {
		u.logger.Error("service restart failed, rolling back",
			"error", err,
		)
		// Rollback symlink
		if rbErr := switchSymlink(previousSlot); rbErr != nil {
			u.logger.Error("rollback symlink switch failed",
				"error", rbErr,
			)
		}
		return fmt.Errorf("service restart: %w", err)
	}

	// --- Step 6: Health check with auto-rollback ---
	if err := u.healthCheck(ctx); err != nil {
		u.logger.Error("health check failed after update, rolling back",
			"error", err,
		)
		if rbErr := switchSymlink(previousSlot); rbErr != nil {
			u.logger.Error("rollback symlink switch failed",
				"error", rbErr,
			)
		}
		// Restart old version
		if rbErr := u.restartService(ctx); rbErr != nil {
			u.logger.Error("rollback restart failed",
				"error", rbErr,
			)
		}
		return fmt.Errorf("health check: %w", err)
	}

	u.currentVersion = newVersion

	// --- Cleanup old signature file (keep binary in inactive slot) ---
	os.Remove(sigPath)

	u.logger.Info("OTA update completed successfully",
		"new_version", newVersion,
		"active_slot", inactiveSlot.String(),
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

// downloadFile downloads a file from the OTA server to a local path.
func (u *OTAUpdater) downloadFile(ctx context.Context, endpoint, dst string) error {
	fileURL := u.downloadURL + endpoint
	u.logger.Debug("downloading file", "url", fileURL, "dst", dst)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, endpoint)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(u.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("write file: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(dst, 0755); err != nil {
		os.Remove(dst)
		return fmt.Errorf("chmod: %w", err)
	}

	u.logger.Debug("file downloaded", "size", written, "path", dst)
	return nil
}

// verifyEd25519Signature verifies the Ed25519 signature of a binary file.
//
// The signature file (.sig) contains a raw 64-byte Ed25519 signature.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности — подпись Ed25519
//   - IEC 62443-3-3 SL-3: Signed firmware updates
//   - OWASP ASVS V12.3: File integrity verification
func (u *OTAUpdater) verifyEd25519Signature(binaryPath, sigPath string) error {
	// Read binary
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read binary: %w", err)
	}

	// Read signature
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("read signature: %w", err)
	}

	// Ed25519 signature is exactly 64 bytes
	if len(sigData) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: got %d, want %d",
			len(sigData), ed25519.SignatureSize,
		)
	}

	// Verify signature
	if !ed25519.Verify(ed25519PublicKey, binaryData, sigData) {
		return fmt.Errorf("Ed25519 signature verification FAILED — binary may be tampered")
	}

	return nil
}

// restartService calls systemctl restart edge-agent with a 30-second timeout.
func (u *OTAUpdater) restartService(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
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

// healthCheck polls systemctl is-active with a timeout.
// Returns nil if the service is active within the timeout.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Health monitoring
//   - Приказ ОАЦ №66 п. 7.18.6: Мониторинг и реагирование
func (u *OTAUpdater) healthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timed out after %s", healthCheckTimeout)
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "systemctl", "is-active", "edge-agent")
			output, err := cmd.Output()
			if err == nil && string(output) == "active\n" {
				u.logger.Info("health check passed — service is active")
				return nil
			}
			u.logger.Debug("waiting for service to become active",
				"output", string(output),
			)
		}
	}
}

// Rollback switches to the inactive slot (previous version) without a backup file.
//
// The dual-boot design preserves both slot binaries, so rollback is simply:
//
//	Switch symlink → restart service → health check
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.5: Управление обновлениями (rollback)
//   - IEC 62443-3-3 SL-3: Graceful degradation
func (u *OTAUpdater) Rollback() error {
	previousSlot := CurrentSlot()
	targetSlot := InactiveSlot()

	u.logger.Info("initiating rollback via slot switch",
		"from", previousSlot.String(),
		"to", targetSlot.String(),
	)

	// Verify target slot has a valid binary
	targetPath := targetSlot.Path()
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("rollback target slot binary not found: %s", targetPath)
	}

	// Switch symlink
	if err := switchSymlink(targetSlot); err != nil {
		return fmt.Errorf("rollback symlink switch: %w", err)
	}

	// Restart with old binary
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	if err := u.restartService(ctx); err != nil {
		return fmt.Errorf("rollback restart: %w", err)
	}

	// Health check
	if err := u.healthCheck(ctx); err != nil {
		// Critical: try to switch back
		u.logger.Error("rollback health check failed, attempting recovery",
			"error", err,
		)
		if rbErr := switchSymlink(previousSlot); rbErr != nil {
			u.logger.Error("recovery symlink switch failed",
				"error", rbErr,
			)
		}
		return fmt.Errorf("rollback health check: %w", err)
	}

	u.logger.Info("rollback completed successfully",
		"active_slot", targetSlot.String(),
	)
	return nil
}

// CurrentVersion returns the currently running agent version.
func (u *OTAUpdater) CurrentVersion() string {
	return u.currentVersion
}

// SlotsStatus returns the status of both A/B slots.
// Useful for diagnostics and monitoring.
func (u *OTAUpdater) SlotsStatus() map[string]string {
	active := CurrentSlot()

	status := make(map[string]string)

	for _, slot := range []Slot{SlotA, SlotB} {
		path := slot.Path()
		info, err := os.Stat(path)
		if err != nil {
			status[slot.String()] = "absent"
			continue
		}

		label := "inactive"
		if slot == active {
			label = "active"
		}

		status[slot.String()] = fmt.Sprintf("%s (size=%d, mod=%s)",
			label, info.Size(), info.ModTime().Format(time.RFC3339),
		)
	}

	return status
}
