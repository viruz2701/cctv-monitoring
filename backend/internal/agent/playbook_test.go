package agent

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPlaybookRegistry(t *testing.T) {
	r := NewPlaybookRegistry(nil)
	if r == nil {
		t.Fatal("NewPlaybookRegistry returned nil")
	}
	if len(r.List()) != 0 {
		t.Errorf("expected 0 playbooks, got %d", len(r.List()))
	}
}

func TestPlaybookRegistryLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_playbook.yml")

	content := `name: test_reboot
description: Test reboot playbook
version: "1.0"
max_retries: 2
cooldown: "10m"
applicable:
  - vendor_type: Hikvision
    alarm_method: 5
    device_type: camera
steps:
  - name: wait_before
    action: wait
    timeout: "5s"
    on_failure: continue
    params:
      duration: "5s"
  - name: reboot_device
    action: onvif_reboot
    timeout: "30s"
    retries: 2
    retry_delay: "10s"
    on_failure: escalate
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewPlaybookRegistry(slog.Default())
	if err := r.LoadFile(path); err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	pb, ok := r.Get("test_reboot")
	if !ok {
		t.Fatal("playbook 'test_reboot' not found")
	}
	if pb.Version != "1.0" {
		t.Errorf("expected version '1.0', got %q", pb.Version)
	}
	if pb.MaxRetries != 2 {
		t.Errorf("expected MaxRetries=2, got %d", pb.MaxRetries)
	}
	if pb.cooldownDur != 10*time.Minute {
		t.Errorf("expected cooldown=10m, got %v", pb.cooldownDur)
	}
	if len(pb.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(pb.Steps))
	}
	if pb.Steps[0].Action != ActionWait {
		t.Errorf("expected step 0 action='wait', got %q", pb.Steps[0].Action)
	}
	if pb.Steps[1].Action != ActionONVIFReboot {
		t.Errorf("expected step 1 action='onvif_reboot', got %q", pb.Steps[1].Action)
	}
	if pb.Steps[1].Retries != 2 {
		t.Errorf("expected step 1 retries=2, got %d", pb.Steps[1].Retries)
	}
	if pb.Steps[1].retryDelay != 10*time.Second {
		t.Errorf("expected step 1 retry_delay=10s, got %v", pb.Steps[1].retryDelay)
	}
}

func TestPlaybookRegistryLoadDir(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "playbook_a.yml"), []byte(`name: playbook_a
version: "1.0"
steps: []`), 0644)

	os.WriteFile(filepath.Join(dir, "playbook_b.yaml"), []byte(`name: playbook_b
version: "1.0"
steps: []`), 0644)

	os.WriteFile(filepath.Join(dir, "not_a_playbook.txt"), []byte(`not yaml`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	if err := r.LoadDir(dir); err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}

	names := r.List()
	if len(names) != 2 {
		t.Errorf("expected 2 playbooks, got %d: %v", len(names), names)
	}
}

func TestPlaybookRegistryLoadFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yml")
	os.WriteFile(path, []byte(`name: [[invalid yaml!!!`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	err := r.LoadFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestPlaybookRegistryLoadFileNoName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noname.yml")
	os.WriteFile(path, []byte(`version: "1.0"
steps: []`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	err := r.LoadFile(path)
	if err == nil {
		t.Error("expected error for playbook without name, got nil")
	}
}

func TestPlaybookRegistryLoadFileInvalidCooldown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_cooldown.yml")
	os.WriteFile(path, []byte(`name: bad_cooldown
version: "1.0"
cooldown: "not_a_duration"
steps: []`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	err := r.LoadFile(path)
	if err == nil {
		t.Error("expected error for invalid cooldown, got nil")
	}
}

func TestPlaybookRegistryGetMissing(t *testing.T) {
	r := NewPlaybookRegistry(slog.Default())
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get should return false for missing playbook")
	}
}

func TestPlaybookRegistryFindApplicable(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "hikvision_reboot.yml"), []byte(`name: hikvision_reboot
version: "1.0"
applicable:
  - vendor_type: Hikvision
    alarm_method: 5
    device_type: camera
steps: []`), 0644)

	os.WriteFile(filepath.Join(dir, "dahua_reboot.yml"), []byte(`name: dahua_reboot
version: "1.0"
applicable:
  - vendor_type: Dahua
    alarm_method: 5
    device_type: camera
steps: []`), 0644)

	os.WriteFile(filepath.Join(dir, "universal.yml"), []byte(`name: universal
version: "1.0"
steps: []`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	r.LoadDir(dir)

	// Hikvision VideoLoss camera
	pbs := r.FindApplicable("Hikvision", 5, "camera", 1)
	if len(pbs) != 2 {
		t.Errorf("expected 2 playbooks for Hikvision/VideoLoss/camera, got %d", len(pbs))
	}

	// Dahua VideoLoss camera
	pbs = r.FindApplicable("Dahua", 5, "camera", 1)
	if len(pbs) != 2 {
		t.Errorf("expected 2 playbooks for Dahua/VideoLoss/camera, got %d", len(pbs))
	}

	// Generic EquipmentFault switch — no specific match
	pbs = r.FindApplicable("Generic", 6, "switch", 1)
	if len(pbs) != 1 {
		t.Errorf("expected 1 universal playbook for Generic/EquipmentFault/switch, got %d", len(pbs))
	}
}

func TestPlaybookRegistryCanRun(t *testing.T) {
	r := NewPlaybookRegistry(slog.Default())

	// Can't run non-existent playbook
	if r.CanRun("nonexistent", "cam-001") {
		t.Error("CanRun should return false for non-existent playbook")
	}

	// Register a playbook with cooldown
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "cooldown_test.yml"), []byte(`name: cooldown_test
version: "1.0"
cooldown: "500ms"
steps: []`), 0644)
	r.LoadDir(dir)

	// First run should be allowed
	if !r.CanRun("cooldown_test", "cam-001") {
		t.Error("CanRun should return true before first run")
	}

	// Mark run
	r.MarkRun("cooldown_test", "cam-001")

	// Immediately after — should be blocked by cooldown
	if r.CanRun("cooldown_test", "cam-001") {
		t.Error("CanRun should return false immediately after run (cooldown active)")
	}

	// Wait for cooldown
	time.Sleep(600 * time.Millisecond)

	// After cooldown — should be allowed again
	if !r.CanRun("cooldown_test", "cam-001") {
		t.Error("CanRun should return true after cooldown expires")
	}
}

func TestPlaybookRegistryCanRunNoCooldown(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "no_cooldown.yml"), []byte(`name: no_cooldown
version: "1.0"
steps: []`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	r.LoadDir(dir)

	if !r.CanRun("no_cooldown", "cam-001") {
		t.Error("CanRun should return true for playbook without cooldown")
	}

	r.MarkRun("no_cooldown", "cam-001")

	if !r.CanRun("no_cooldown", "cam-001") {
		t.Error("CanRun should always return true when cooldown is 0")
	}
}

func TestPlaybookRegistryMarkRun(t *testing.T) {
	r := NewPlaybookRegistry(slog.Default())

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mark_test.yml"), []byte(`name: mark_test
version: "1.0"
steps: []`), 0644)
	r.LoadDir(dir)

	r.MarkRun("mark_test", "cam-001")
	// MarkRun should not panic even if playbook doesn't exist
	r.MarkRun("nonexistent", "cam-001")

	// Verify it's tracked via CanRun
	if !r.CanRun("mark_test", "cam-001") {
		t.Error("CanRun should return true for playbook with no cooldown after MarkRun")
	}
}

func TestPlaybookStepDefaults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "defaults.yml"), []byte(`name: defaults_test
version: "1.0"
steps:
  - name: step_with_defaults
    action: wait
    on_failure: continue
    retries: 0
    retry_delay: "5s"
`), 0644)

	r := NewPlaybookRegistry(slog.Default())
	if err := r.LoadDir(dir); err != nil {
		t.Fatal(err)
	}

	pb, _ := r.Get("defaults_test")
	step := pb.Steps[0]
	if step.Retries != 0 {
		t.Errorf("expected Retries=0, got %d", step.Retries)
	}
	if step.retryDelay != 5*time.Second {
		t.Errorf("expected retryDelay=5s, got %v", step.retryDelay)
	}
}
