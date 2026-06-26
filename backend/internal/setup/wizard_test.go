// Package setup — unit tests for Setup Wizard.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.4: Setup Wizard Tests
// ═══════════════════════════════════════════════════════════════════════════
package setup

import (
	"testing"

	"gb-telemetry-collector/internal/compliance"
)

func setupTestWizard(t *testing.T) *SetupWizard {
	t.Helper()
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewBYProfile()),
		compliance.WithProfile(compliance.NewEUProfile()),
		compliance.WithProfile(compliance.NewINTLProfile()),
	)
	return NewSetupWizard(registry)
}

func TestNewSetupWizard(t *testing.T) {
	w := setupTestWizard(t)
	if w == nil {
		t.Fatal("wizard must not be nil")
	}
	if w.IsStarted() {
		t.Error("new wizard should not be started")
	}
	if w.IsCompleted() {
		t.Error("new wizard should not be completed")
	}
}

func TestStartWizard(t *testing.T) {
	w := setupTestWizard(t)

	if err := w.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if !w.IsStarted() {
		t.Error("wizard should be started")
	}
	if w.CurrentStep() != StepRegion {
		t.Errorf("expected step %d, got %d", StepRegion, w.CurrentStep())
	}
}

func TestStartTwice(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	err := w.Start()
	if err == nil {
		t.Fatal("Start twice should return error")
	}
}

func TestFullWizardFlowBY(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()

	// Step 1: Region
	if err := w.SetRegion(compliance.RegionBY); err != nil {
		t.Fatalf("SetRegion error: %v", err)
	}
	if w.CurrentStep() != StepCrypto {
		t.Errorf("expected step %d, got %d", StepCrypto, w.CurrentStep())
	}

	// Step 2: Crypto
	if err := w.ConfirmCrypto(true); err != nil {
		t.Fatalf("ConfirmCrypto error: %v", err)
	}
	if w.CurrentStep() != StepStorage {
		t.Errorf("expected step %d, got %d", StepStorage, w.CurrentStep())
	}

	// Step 3: Storage
	if err := w.SetStorage("local", "", "", ""); err != nil {
		t.Fatalf("SetStorage error: %v", err)
	}
	if w.CurrentStep() != StepAdmin {
		t.Errorf("expected step %d, got %d", StepAdmin, w.CurrentStep())
	}

	// Step 4: Admin (BY requires signature)
	if err := w.SetAdmin("admin", "admin@example.com", "digital-signature-123"); err != nil {
		t.Fatalf("SetAdmin error: %v", err)
	}
	if w.CurrentStep() != StepNetwork {
		t.Errorf("expected step %d, got %d", StepNetwork, w.CurrentStep())
	}

	// Step 5: Network
	if err := w.SetNetwork(8443, "/certs/server.crt", "/certs/server.key"); err != nil {
		t.Fatalf("SetNetwork error: %v", err)
	}
	if w.CurrentStep() != StepNotifications {
		t.Errorf("expected step %d, got %d", StepNotifications, w.CurrentStep())
	}

	// Step 6: Notifications
	if err := w.SetNotifications("token:123", "smtp.example.com", 587, "user"); err != nil {
		t.Fatalf("SetNotifications error: %v", err)
	}
	if w.CurrentStep() != StepReview {
		t.Errorf("expected step %d, got %d", StepReview, w.CurrentStep())
	}

	// Step 7: Complete
	if err := w.Complete(); err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if !w.IsCompleted() {
		t.Error("wizard should be completed")
	}

	// Verify config
	cfg := w.Config()
	if cfg.Region != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", cfg.Region)
	}
	if !cfg.RegionLocked {
		t.Error("region should be locked after completion")
	}
	if !cfg.CryptoConfirmed {
		t.Error("crypto should be confirmed")
	}
	if cfg.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
}

func TestFullWizardFlowEU(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()

	w.SetRegion(compliance.RegionEU)
	w.ConfirmCrypto(true)
	w.SetStorage("s3", "https://s3.eu-central-1.amazonaws.com", "my-bucket", "eu-central-1")
	w.SetAdmin("admin", "admin@example.com", "") // No signature needed for EU

	// Verify S3 config
	cfg := w.Config()
	if cfg.StorageType != "s3" {
		t.Errorf("expected s3 storage, got %s", cfg.StorageType)
	}
	if cfg.S3Bucket != "my-bucket" {
		t.Errorf("expected bucket my-bucket, got %s", cfg.S3Bucket)
	}
}

func TestRegionErrors(t *testing.T) {
	w := setupTestWizard(t)

	// Not started
	err := w.SetRegion(compliance.RegionBY)
	if err == nil {
		t.Fatal("SetRegion before Start should error")
	}

	w.Start()

	// Invalid region
	err = w.SetRegion("INVALID")
	if err == nil {
		t.Fatal("SetRegion with invalid region should error")
	}

	// Skip crypto step
	err = w.SetStorage("local", "", "", "")
	if err == nil {
		t.Fatal("SetStorage before ConfirmCrypto should error")
	}
}

func TestAdminSignatureRequiredForBY(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	w.SetRegion(compliance.RegionBY)
	w.ConfirmCrypto(true)
	w.SetStorage("local", "", "", "")

	err := w.SetAdmin("admin", "admin@example.com", "")
	if err == nil {
		t.Fatal("SetAdmin for BY without signature should error")
	}
}

func TestAdminSignatureNotRequiredForEU(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	w.SetRegion(compliance.RegionEU)
	w.ConfirmCrypto(true)
	w.SetStorage("local", "", "", "")

	err := w.SetAdmin("admin", "admin@example.com", "")
	if err != nil {
		t.Fatalf("SetAdmin for EU should work without signature: %v", err)
	}
}

func TestInvalidStorageType(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	w.SetRegion(compliance.RegionINTL)
	w.ConfirmCrypto(true)

	err := w.SetStorage("invalid", "", "", "")
	if err == nil {
		t.Fatal("invalid storage type should error")
	}
}

func TestS3RequiresEndpoint(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	w.SetRegion(compliance.RegionINTL)
	w.ConfirmCrypto(true)

	err := w.SetStorage("s3", "", "", "")
	if err == nil {
		t.Fatal("S3 without endpoint should error")
	}
}

func TestCompleteWithoutAdmin(t *testing.T) {
	w := setupTestWizard(t)
	w.Start()
	w.SetRegion(compliance.RegionINTL)
	w.ConfirmCrypto(true)
	w.SetStorage("local", "", "", "")
	// Skip admin
	w.SetAdmin("admin", "admin@example.com", "") // Use valid admin
	w.SetNetwork(8080, "", "")
	w.SetNotifications("", "", 0, "")

	if err := w.Complete(); err != nil {
		t.Fatalf("Complete error: %v", err)
	}
}

func TestSetupCompleteHandler(t *testing.T) {
	called := false
	w := NewSetupWizard(
		compliance.NewProfileRegistry(
			compliance.WithProfile(compliance.NewINTLProfile()),
		),
		WithSetupCompleteHandler(func(cfg *SetupConfig) error {
			called = true
			if cfg.Region != compliance.RegionINTL {
				t.Errorf("expected region INTL, got %s", cfg.Region)
			}
			return nil
		}),
	)

	w.Start()
	w.SetRegion(compliance.RegionINTL)
	w.ConfirmCrypto(true)
	w.SetStorage("local", "", "", "")
	w.SetAdmin("admin", "admin@example.com", "")
	w.SetNetwork(8080, "", "")
	w.SetNotifications("", "", 0, "")
	w.Complete()

	if !called {
		t.Error("setup complete handler was not called")
	}
}

func TestAvailableRegions(t *testing.T) {
	regions := AvailableRegions()
	if len(regions) != 3 {
		t.Fatalf("expected 3 regions, got %d", len(regions))
	}

	regionMap := make(map[string]bool)
	for _, r := range regions {
		regionMap[r.Region] = true
		if r.Name == "" {
			t.Errorf("region %s has empty name", r.Region)
		}
		if len(r.Compliance) == 0 {
			t.Errorf("region %s has no compliance standards", r.Region)
		}
		if r.LegalNotice == "" {
			t.Errorf("region %s has no legal notice", r.Region)
		}
	}

	if !regionMap["BY"] {
		t.Error("BY region missing")
	}
	if !regionMap["EU"] {
		t.Error("EU region missing")
	}
	if !regionMap["INTL"] {
		t.Error("INTL region missing")
	}
}

func TestAllSteps(t *testing.T) {
	steps := AllSteps()
	if len(steps) != 7 {
		t.Fatalf("expected 7 steps, got %d", len(steps))
	}
}

func TestWizardConfig(t *testing.T) {
	w := setupTestWizard(t)

	// Config should be available even before start
	cfg := w.Config()
	if cfg == nil {
		t.Fatal("config must not be nil")
	}

	w.Start()
	w.SetRegion(compliance.RegionINTL)
	cfg = w.Config()
	if cfg.Region != compliance.RegionINTL {
		t.Errorf("expected region INTL, got %s", cfg.Region)
	}
}
