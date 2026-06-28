package playbook

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// writePlaybook создаёт временный YAML-файл плейбука и возвращает путь.
func writePlaybook(t *testing.T, dir, filename string, schema *PlaybookSchema) string {
	t.Helper()

	data, err := yaml.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal playbook: %v", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

// mustLoad загружает файл и паникует при ошибке.
func mustLoad(t *testing.T, r *PlaybookRegistry, path string) {
	t.Helper()
	if err := r.LoadFile(path); err != nil {
		t.Fatalf("LoadFile(%s): %v", path, err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Versioning
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_Versioning(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Создаём первую версию
	v1Path := writePlaybook(t, dir, "test-v1.yml", &PlaybookSchema{
		Name:    "test-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "step1", Action: "check", Target: "device"}},
	})
	mustLoad(t, r, v1Path)

	// Проверяем, что версия загружена
	if v := r.GetVersion("test-playbook"); v != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", v)
	}

	// Создаём вторую версию
	v2Path := writePlaybook(t, dir, "test-v2.yml", &PlaybookSchema{
		Name:    "test-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "step1", Action: "restart", Target: "device"}},
	})
	mustLoad(t, r, v2Path)

	// Проверяем, что активна v2
	if v := r.GetVersion("test-playbook"); v != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %q", v)
	}

	// Проверяем историю (3 записи: v1 active, v1 inactive, v2 active)
	history := r.GetHistory("test-playbook")
	if len(history) < 3 {
		t.Fatalf("expected >= 3 history entries, got %d", len(history))
	}

	// Первая запись — v1, активна (первоначальная загрузка)
	if history[0].Version != "1.0.0" {
		t.Errorf("expected history[0] version 1.0.0, got %q", history[0].Version)
	}
	if !history[0].Active {
		t.Error("expected history[0] to be active (first load)")
	}

	// Вторая запись — v1, неактивна (сохранена при загрузке v2)
	if history[1].Version != "1.0.0" {
		t.Errorf("expected history[1] version 1.0.0, got %q", history[1].Version)
	}
	if history[1].Active {
		t.Error("expected history[1] to be inactive (previous version)")
	}

	// Третья запись — v2, активна
	if history[2].Version != "2.0.0" {
		t.Errorf("expected history[2] version 2.0.0, got %q", history[2].Version)
	}
	if !history[2].Active {
		t.Error("expected history[2] to be active")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Semver Validation
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_SemverValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		wantOk  bool
	}{
		{"valid basic", "1.2.3", true},
		{"valid with v prefix", "v1.2.3", true},
		{"valid major zero", "0.1.0", true},
		{"valid pre-release", "1.2.3-alpha", true},
		{"valid pre-release with dot", "1.2.3-alpha.1", true},
		{"valid build metadata", "1.2.3+build123", true},
		{"valid pre-release+build", "1.2.3-rc.1+build.42", true},
		{"valid large numbers", "10.200.3000", true},
		{"invalid empty", "", false},
		{"invalid no patch", "1.2", false},
		{"invalid only major", "1", false},
		{"invalid text", "abc", false},
		{"invalid with spaces", "1.2.3 beta", false},
		{"invalid leading zero", "01.2.3", false},
		{"invalid double dots", "1..2", false},
		{"invalid trailing dot", "1.2.3.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSemver(tt.version)
			if got != tt.wantOk {
				t.Errorf("isValidSemver(%q) = %v, want %v", tt.version, got, tt.wantOk)
			}
		})
	}
}

func TestPlaybookRegistry_LoadFile_RejectsInvalidSemver(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "bad-version.yml", &PlaybookSchema{
		Name:    "bad-playbook",
		Version: "not-semver",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})

	err := r.LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid semver, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rollback
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_Rollback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем v1
	v1Path := writePlaybook(t, dir, "rb-v1.yml", &PlaybookSchema{
		Name:    "rb-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "step1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, v1Path)

	// Загружаем v2
	v2Path := writePlaybook(t, dir, "rb-v2.yml", &PlaybookSchema{
		Name:    "rb-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "step2", Action: "restart", Target: "dev"}},
	})
	mustLoad(t, r, v2Path)

	// Rollback к v1
	ok, err := r.Rollback("rb-playbook", "1.0.0")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if !ok {
		t.Fatal("expected Rollback to return true")
	}

	// Проверяем, что активна v1
	if v := r.GetVersion("rb-playbook"); v != "1.0.0" {
		t.Errorf("after rollback expected version 1.0.0, got %q", v)
	}

	// Проверяем, что схема полностью восстановлена (не partial)
	schema, ok := r.Get("rb-playbook")
	if !ok {
		t.Fatal("playbook not found after rollback")
	}
	if len(schema.Steps) != 1 {
		t.Errorf("expected 1 step after rollback, got %d", len(schema.Steps))
	}
	if schema.Steps[0].Action != "check" {
		t.Errorf("expected step action 'check', got %q", schema.Steps[0].Action)
	}
}

func TestPlaybookRegistry_RollbackNotFound(t *testing.T) {
	t.Parallel()

	r := NewPlaybookRegistry(DefaultRegistryConfig)
	ok, err := r.Rollback("nonexistent", "1.0.0")
	if err != nil {
		t.Fatalf("Rollback nonexistent: %v", err)
	}
	if ok {
		t.Fatal("expected false for nonexistent playbook")
	}
}

func TestPlaybookRegistry_RollbackLatest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем v1
	v1Path := writePlaybook(t, dir, "rbl-v1.yml", &PlaybookSchema{
		Name:    "rbl-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, v1Path)

	// Загружаем v2
	v2Path := writePlaybook(t, dir, "rbl-v2.yml", &PlaybookSchema{
		Name:    "rbl-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "s2", Action: "restart", Target: "dev"}},
	})
	mustLoad(t, r, v2Path)

	// Загружаем v3
	v3Path := writePlaybook(t, dir, "rbl-v3.yml", &PlaybookSchema{
		Name:    "rbl-playbook",
		Version: "3.0.0",
		Steps:   []PlaybookStep{{Name: "s3", Action: "reboot", Target: "dev"}},
	})
	mustLoad(t, r, v3Path)

	// Проверяем активную версию
	if v := r.GetVersion("rbl-playbook"); v != "3.0.0" {
		t.Fatalf("expected active version 3.0.0, got %q", v)
	}

	// RollbackLatest — должен откатить к v2 (последняя активная до v3)
	ok, err := r.RollbackLatest("rbl-playbook")
	if err != nil {
		t.Fatalf("RollbackLatest: %v", err)
	}
	if !ok {
		t.Fatal("expected RollbackLatest to return true")
	}

	if v := r.GetVersion("rbl-playbook"); v != "2.0.0" {
		t.Errorf("after RollbackLatest expected version 2.0.0, got %q", v)
	}
}

func TestPlaybookRegistry_RollbackLatest_NoHistory(t *testing.T) {
	t.Parallel()

	r := NewPlaybookRegistry(DefaultRegistryConfig)
	ok, err := r.RollbackLatest("nonexistent")
	if err != nil {
		t.Fatalf("RollbackLatest nonexistent: %v", err)
	}
	if ok {
		t.Fatal("expected false for nonexistent playbook")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// DiffVersions
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_DiffVersions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Создаём v1 с двумя шагами
	v1Path := writePlaybook(t, dir, "diff-v1.yml", &PlaybookSchema{
		Name:        "diff-playbook",
		Version:     "1.0.0",
		Description: "initial version",
		Steps: []PlaybookStep{
			{Name: "check-alive", Action: "ping", Target: "device", Timeout: "5s"},
			{Name: "collect-logs", Action: "exec", Target: "device", Timeout: "30s"},
		},
	})
	mustLoad(t, r, v1Path)

	// Создаём v2 с изменениями: удалён check-alive, изменён collect-logs, добавлен reboot
	v2Path := writePlaybook(t, dir, "diff-v2.yml", &PlaybookSchema{
		Name:        "diff-playbook",
		Version:     "2.0.0",
		Description: "updated version",
		Steps: []PlaybookStep{
			{Name: "collect-logs", Action: "exec", Target: "device", Timeout: "60s"}, // timeout changed
			{Name: "reboot-device", Action: "reboot", Target: "device", Timeout: "120s"},
		},
	})
	mustLoad(t, r, v2Path)

	// Сравниваем версии
	diff, err := r.DiffVersions("diff-playbook", "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("DiffVersions: %v", err)
	}

	if diff == nil {
		t.Fatal("DiffVersions returned nil")
	}

	// Проверяем From/To
	if diff.From != "1.0.0" {
		t.Errorf("expected From=1.0.0, got %q", diff.From)
	}
	if diff.To != "2.0.0" {
		t.Errorf("expected To=2.0.0, got %q", diff.To)
	}

	// Проверяем добавленные шаги (reboot-device)
	if len(diff.StepsAdded) != 1 {
		t.Errorf("expected 1 step added, got %d", len(diff.StepsAdded))
	} else if diff.StepsAdded[0].Name != "reboot-device" {
		t.Errorf("expected added step 'reboot-device', got %q", diff.StepsAdded[0].Name)
	}

	// Проверяем удалённые шаги (check-alive)
	if len(diff.StepsRemoved) != 1 {
		t.Errorf("expected 1 step removed, got %d", len(diff.StepsRemoved))
	} else if diff.StepsRemoved[0].Name != "check-alive" {
		t.Errorf("expected removed step 'check-alive', got %q", diff.StepsRemoved[0].Name)
	}

	// Проверяем изменённые шаги (collect-logs timeout: 30s -> 60s)
	if len(diff.StepsChanged) == 0 {
		t.Fatal("expected steps changed, got 0")
	}

	foundTimeoutChange := false
	for _, c := range diff.StepsChanged {
		if c.Name == "collect-logs" && c.Field == "timeout" {
			foundTimeoutChange = true
			if c.OldVal != "30s" || c.NewVal != "60s" {
				t.Errorf("expected timeout change 30s->60s, got %q->%q", c.OldVal, c.NewVal)
			}
		}
	}
	if !foundTimeoutChange {
		t.Error("expected timeout change for collect-logs step")
	}

	// Проверяем изменения метаданных
	if diff.MetadataChanged == nil {
		t.Fatal("expected metadata changes")
	}
	descChange, ok := diff.MetadataChanged["description"]
	if !ok {
		t.Error("expected description change in metadata")
	} else {
		if descChange.Old != "initial version" || descChange.New != "updated version" {
			t.Errorf("unexpected description change: %v -> %v", descChange.Old, descChange.New)
		}
	}
}

func TestPlaybookRegistry_DiffVersions_SameVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "same-v1.yml", &PlaybookSchema{
		Name:    "same-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	diff, err := r.DiffVersions("same-playbook", "1.0.0", "1.0.0")
	if err != nil {
		t.Fatalf("DiffVersions: %v", err)
	}
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if len(diff.StepsAdded) != 0 {
		t.Errorf("expected 0 steps added, got %d", len(diff.StepsAdded))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TagVersion / GetVersionByTag / ListTags
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_TagVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "tag-v1.yml", &PlaybookSchema{
		Name:    "tag-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	// Добавляем тег
	if err := r.TagVersion("tag-playbook", "1.0.0", "stable"); err != nil {
		t.Fatalf("TagVersion: %v", err)
	}

	// Получаем версию по тегу
	version, ok := r.GetVersionByTag("tag-playbook", "stable")
	if !ok {
		t.Fatal("expected to find tag 'stable'")
	}
	if version != "1.0.0" {
		t.Errorf("expected version 1.0.0 for tag 'stable', got %q", version)
	}
}

func TestPlaybookRegistry_TagVersion_NotFound(t *testing.T) {
	t.Parallel()

	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Тег для несуществующего плейбука
	err := r.TagVersion("nonexistent", "1.0.0", "stable")
	if err == nil {
		t.Fatal("expected error for nonexistent playbook")
	}
}

func TestPlaybookRegistry_GetVersionByTag_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "tag2-v1.yml", &PlaybookSchema{
		Name:    "tag2-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	// Несуществующий тег
	_, ok := r.GetVersionByTag("tag2-playbook", "nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent tag")
	}

	// Несуществующий плейбук
	_, ok = r.GetVersionByTag("nonexistent", "stable")
	if ok {
		t.Fatal("expected false for nonexistent playbook")
	}
}

func TestPlaybookRegistry_ListTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "lt-v1.yml", &PlaybookSchema{
		Name:    "lt-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	// Добавляем теги
	if err := r.TagVersion("lt-playbook", "1.0.0", "stable"); err != nil {
		t.Fatalf("TagVersion stable: %v", err)
	}
	if err := r.TagVersion("lt-playbook", "1.0.0", "production"); err != nil {
		t.Fatalf("TagVersion production: %v", err)
	}

	// Списываем теги
	tags := r.ListTags("lt-playbook")
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}

	// Сортируем для детерминированного сравнения
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	if tags[0].Name != "production" {
		t.Errorf("expected tag[0]='production', got %q", tags[0].Name)
	}
	if tags[0].Version != "1.0.0" {
		t.Errorf("expected tag[0].Version='1.0.0', got %q", tags[0].Version)
	}
	if tags[1].Name != "stable" {
		t.Errorf("expected tag[1]='stable', got %q", tags[1].Name)
	}
	if tags[1].Version != "1.0.0" {
		t.Errorf("expected tag[1].Version='1.0.0', got %q", tags[1].Version)
	}

	// Перезаписываем тег
	if err := r.TagVersion("lt-playbook", "1.0.0", "stable"); err != nil {
		t.Fatalf("TagVersion stable overwrite: %v", err)
	}
	tags2 := r.ListTags("lt-playbook")
	if len(tags2) != 2 {
		t.Errorf("expected 2 tags after overwrite, got %d", len(tags2))
	}
}

func TestPlaybookRegistry_ListTags_NoTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "nt-v1.yml", &PlaybookSchema{
		Name:    "nt-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	tags := r.ListTags("nt-playbook")
	if tags != nil {
		t.Errorf("expected nil for no tags, got %v", tags)
	}

	// Несуществующий плейбук
	tags = r.ListTags("nonexistent")
	if tags != nil {
		t.Errorf("expected nil for nonexistent playbook, got %v", tags)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// WatchDir (hot reload)
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_WatchDir(t *testing.T) {
	// WatchDir запускает горутину с ticker'ом — не используем t.Parallel()
	// для избежания race с горутиной.

	dir := t.TempDir()
	cfg := DefaultRegistryConfig
	cfg.WatchInterval = 50 * time.Millisecond // быстрый poll

	r := NewPlaybookRegistry(cfg)

	// Загружаем начальный файл
	v1Path := writePlaybook(t, dir, "watch-v1.yml", &PlaybookSchema{
		Name:    "watch-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, v1Path)

	// Загружаем директорию
	if err := r.LoadDir(dir); err != nil {
		t.Fatalf("LoadDir: %v", err)
	}

	// Запускаем WatchDir
	errCh := r.WatchDir()
	defer r.StopWatch()

	// Ждём, чтобы модификация файла точно попала в следующий poll
	time.Sleep(100 * time.Millisecond)

	// Меняем файл с заведомо разным modtime
	writePlaybook(t, dir, "watch-v1.yml", &PlaybookSchema{
		Name:    "watch-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "s2", Action: "restart", Target: "dev"}},
	})

	// Канал ошибок получает только ошибки, успешные reload туда не попадают.
	// Поэтому опрашиваем версию с таймаутом.
	deadline := time.After(3 * time.Second)
	for {
		if v := r.GetVersion("watch-playbook"); v == "2.0.0" {
			break // успех
		}
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("watch error: %v", err)
			}
		case <-deadline:
			t.Fatal("timeout waiting for hot reload")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestPlaybookRegistry_WatchDir_Disabled(t *testing.T) {
	t.Parallel()

	// Создаём реестр напрямую, чтобы избежать validate(),
	// который устанавливает WatchInterval по умолчанию.
	r := &PlaybookRegistry{
		schemas:  make(map[string]*PlaybookSchema),
		versions: make(map[string][]VersionEntry),
		tags:     make(map[string][]VersionTag),
		files:    make(map[string]string),
		cfg:      RegistryConfig{WatchInterval: 0},
		logger:   slog.Default().With("component", "playbook-registry"),
		stopCh:   make(chan struct{}),
	}

	errCh := r.WatchDir()

	// Канал должен быть сразу закрыт (WatchInterval = 0)
	_, ok := <-errCh
	if ok {
		t.Fatal("expected closed channel for disabled watch")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Edge Cases
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_SemverWithVPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "vp-v1.yml", &PlaybookSchema{
		Name:    "vprefix-playbook",
		Version: "v1.2.3",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	if v := r.GetVersion("vprefix-playbook"); v != "v1.2.3" {
		t.Errorf("expected version 'v1.2.3', got %q", v)
	}
}

func TestPlaybookRegistry_LoadDir_StrictMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := DefaultRegistryConfig
	cfg.StrictMode = true

	r := NewPlaybookRegistry(cfg)

	// Создаём валидный и невалидный файлы
	writePlaybook(t, dir, "good.yml", &PlaybookSchema{
		Name:    "good",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	writePlaybook(t, dir, "bad.yml", &PlaybookSchema{
		Name:    "bad",
		Version: "not-semver",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})

	err := r.LoadDir(dir)
	if err == nil {
		t.Fatal("expected error in strict mode with invalid playbook")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GetVersionHistory
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_GetVersionHistory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем v1
	v1Path := writePlaybook(t, dir, "gvh-v1.yml", &PlaybookSchema{
		Name:    "gvh-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, v1Path)

	// Загружаем v2
	v2Path := writePlaybook(t, dir, "gvh-v2.yml", &PlaybookSchema{
		Name:    "gvh-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "s2", Action: "restart", Target: "dev"}},
	})
	mustLoad(t, r, v2Path)

	// GetVersionHistory через новый метод
	history := r.GetVersionHistory("gvh-playbook")
	if history == nil {
		t.Fatal("GetVersionHistory returned nil")
	}
	if len(history) < 3 {
		t.Fatalf("expected >= 3 history entries, got %d", len(history))
	}

	// Проверяем, что GetHistory возвращает то же самое
	h2 := r.GetHistory("gvh-playbook")
	if len(history) != len(h2) {
		t.Errorf("GetHistory and GetVersionHistory differ in length: %d vs %d", len(history), len(h2))
	}

	// Последняя запись — активная версия 2.0.0
	last := history[len(history)-1]
	if last.Version != "2.0.0" {
		t.Errorf("expected last history entry version 2.0.0, got %q", last.Version)
	}
	if !last.Active {
		t.Error("expected last history entry to be active")
	}
	if last.SHA256 == "" {
		t.Error("expected SHA256 hash in version entry")
	}
	if last.FilePath == "" {
		t.Error("expected FilePath in version entry")
	}
	if last.LoadedAt.IsZero() {
		t.Error("expected non-zero LoadedAt timestamp")
	}
}

func TestPlaybookRegistry_GetVersionHistory_Nonexistent(t *testing.T) {
	t.Parallel()

	r := NewPlaybookRegistry(DefaultRegistryConfig)

	history := r.GetVersionHistory("nonexistent")
	if history != nil {
		t.Errorf("expected nil for nonexistent playbook, got %v", history)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rollback Edge Cases
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_Rollback_SameVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	path := writePlaybook(t, dir, "rsv-v1.yml", &PlaybookSchema{
		Name:    "rsv-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, path)

	// Rollback к той же версии, что активна — должен быть no-op
	beforeCount := len(r.GetVersionHistory("rsv-playbook"))
	ok, err := r.Rollback("rsv-playbook", "1.0.0")
	if err != nil {
		t.Fatalf("Rollback to same version: %v", err)
	}
	if !ok {
		t.Fatal("expected Rollback to return true (already at target)")
	}

	// Количество записей в истории не должно измениться
	afterCount := len(r.GetVersionHistory("rsv-playbook"))
	if afterCount != beforeCount {
		t.Errorf("history length changed after no-op rollback: %d → %d", beforeCount, afterCount)
	}

	if v := r.GetVersion("rsv-playbook"); v != "1.0.0" {
		t.Errorf("expected version to remain 1.0.0, got %q", v)
	}
}

func TestPlaybookRegistry_Rollback_Multiple(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем 4 версии
	versions := []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}
	for _, v := range versions {
		path := writePlaybook(t, dir, "rm-"+v+".yml", &PlaybookSchema{
			Name:    "rm-playbook",
			Version: v,
			Steps:   []PlaybookStep{{Name: "step-" + v, Action: "action-" + v, Target: "dev"}},
		})
		mustLoad(t, r, path)
	}

	// Активна v4.0.0
	if v := r.GetVersion("rm-playbook"); v != "4.0.0" {
		t.Fatalf("expected version 4.0.0, got %q", v)
	}

	// Rollback к v2.0.0 (через одну)
	ok, err := r.Rollback("rm-playbook", "2.0.0")
	if err != nil {
		t.Fatalf("Rollback to 2.0.0: %v", err)
	}
	if !ok {
		t.Fatal("expected Rollback to 2.0.0 to return true")
	}

	// Проверяем активную версию
	if v := r.GetVersion("rm-playbook"); v != "2.0.0" {
		t.Errorf("after rollback expected version 2.0.0, got %q", v)
	}

	// Проверяем, что схема восстановлена полностью
	schema, ok := r.Get("rm-playbook")
	if !ok {
		t.Fatal("playbook not found after rollback")
	}
	if len(schema.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(schema.Steps))
	}
	if schema.Steps[0].Action != "action-2.0.0" {
		t.Errorf("expected step action 'action-2.0.0', got %q", schema.Steps[0].Action)
	}

	// Rollback снова к v4 — проверяем, что ре-роллбэк работает
	ok, err = r.Rollback("rm-playbook", "4.0.0")
	if err != nil {
		t.Fatalf("Rollback back to 4.0.0: %v", err)
	}
	if !ok {
		t.Fatal("expected Rollback back to 4.0.0 to return true")
	}
	if v := r.GetVersion("rm-playbook"); v != "4.0.0" {
		t.Errorf("after second rollback expected version 4.0.0, got %q", v)
	}
}

func TestPlaybookRegistry_RollbackLatest_MultipleVersions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем 3 версии
	for _, v := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		path := writePlaybook(t, dir, "rlm-"+v+".yml", &PlaybookSchema{
			Name:    "rlm-playbook",
			Version: v,
			Steps:   []PlaybookStep{{Name: "step-" + v, Action: "action-" + v, Target: "dev"}},
		})
		mustLoad(t, r, path)
	}

	// RollbackLatest с v3 → v2
	ok, err := r.RollbackLatest("rlm-playbook")
	if err != nil {
		t.Fatalf("RollbackLatest: %v", err)
	}
	if !ok {
		t.Fatal("expected RollbackLatest to return true")
	}
	if v := r.GetVersion("rlm-playbook"); v != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %q", v)
	}

	// RollbackLatest с v2 → v1
	ok, err = r.RollbackLatest("rlm-playbook")
	if err != nil {
		t.Fatalf("RollbackLatest #2: %v", err)
	}
	if !ok {
		t.Fatal("expected RollbackLatest #2 to return true")
	}
	if v := r.GetVersion("rlm-playbook"); v != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", v)
	}

	// Больше некуда откатывать
	ok, err = r.RollbackLatest("rlm-playbook")
	if err != nil {
		t.Fatalf("RollbackLatest #3: %v", err)
	}
	if ok {
		t.Fatal("expected RollbackLatest to return false when no more history")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Version History Truncation
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_VersionHistory_Truncation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := DefaultRegistryConfig
	cfg.MaxVersionHistory = 5 // лимит в 5 записей

	r := NewPlaybookRegistry(cfg)

	// Загружаем 10 версий — история должна быть обрезана до 5
	for i := 1; i <= 10; i++ {
		ver := fmt.Sprintf("%d.0.0", i)
		path := writePlaybook(t, dir, "tr-"+ver+".yml", &PlaybookSchema{
			Name:    "tr-playbook",
			Version: ver,
			Steps:   []PlaybookStep{{Name: "step-" + ver, Action: "action", Target: "dev"}},
		})
		mustLoad(t, r, path)
	}

	history := r.GetVersionHistory("tr-playbook")
	if len(history) > cfg.MaxVersionHistory {
		t.Errorf("expected history <= %d entries, got %d", cfg.MaxVersionHistory, len(history))
	}

	// Последняя запись должна быть 10.0.0 (активная)
	last := history[len(history)-1]
	if last.Version != "10.0.0" {
		t.Errorf("expected last entry version 10.0.0, got %q", last.Version)
	}
	if !last.Active {
		t.Error("expected last entry to be active")
	}
}

func TestPlaybookRegistry_VersionHistory_Unlimited(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := DefaultRegistryConfig
	cfg.MaxVersionHistory = 0 // без ограничения

	r := NewPlaybookRegistry(cfg)

	// Загружаем много версий
	const count = 30
	for i := 1; i <= count; i++ {
		ver := fmt.Sprintf("%d.0.0", i)
		path := writePlaybook(t, dir, "ul-"+ver+".yml", &PlaybookSchema{
			Name:    "ul-playbook",
			Version: ver,
			Steps:   []PlaybookStep{{Name: "step-" + ver, Action: "action", Target: "dev"}},
		})
		mustLoad(t, r, path)
	}

	history := r.GetVersionHistory("ul-playbook")
	// При безлимитной истории: первая загрузка = 1 запись,
	// каждая последующая = 2 записи (prev inactive + new active)
	// Итого: 1 + 2*(count-1) = 2*count - 1
	expectedLen := 2*count - 1
	if len(history) != expectedLen {
		t.Errorf("unlimited history: expected %d entries, got %d", expectedLen, len(history))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rollback — No Duplicate History Entries
// ═══════════════════════════════════════════════════════════════════════

func TestPlaybookRegistry_Rollback_NoDuplicateHistory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewPlaybookRegistry(DefaultRegistryConfig)

	// Загружаем v1
	v1Path := writePlaybook(t, dir, "nd-v1.yml", &PlaybookSchema{
		Name:    "nd-playbook",
		Version: "1.0.0",
		Steps:   []PlaybookStep{{Name: "s1", Action: "check", Target: "dev"}},
	})
	mustLoad(t, r, v1Path)

	// Загружаем v2
	v2Path := writePlaybook(t, dir, "nd-v2.yml", &PlaybookSchema{
		Name:    "nd-playbook",
		Version: "2.0.0",
		Steps:   []PlaybookStep{{Name: "s2", Action: "restart", Target: "dev"}},
	})
	mustLoad(t, r, v2Path)

	// Загружаем v3
	v3Path := writePlaybook(t, dir, "nd-v3.yml", &PlaybookSchema{
		Name:    "nd-playbook",
		Version: "3.0.0",
		Steps:   []PlaybookStep{{Name: "s3", Action: "reboot", Target: "dev"}},
	})
	mustLoad(t, r, v3Path)

	// Получаем историю ДО rollback
	beforeHistory := r.GetVersionHistory("nd-playbook")

	// Rollback к v1
	ok, err := r.Rollback("nd-playbook", "1.0.0")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if !ok {
		t.Fatal("expected Rollback to return true")
	}

	// Получаем историю ПОСЛЕ rollback
	afterHistory := r.GetVersionHistory("nd-playbook")

	// Проверяем, что дубликатов нет: каждая версия должна встречаться
	// не более 3 раз (активная + неактивная на момент перезаписи +
	// возможная перезагрузка при rollback). При нормальной работе:
	//   - Первая загрузка: 1 запись
	//   - Каждая перезапись новой версией: +1 inactive предыдущей
	//   - Rollback к старой версии: +1 запись (перезагрузка файла)
	versionCounts := make(map[string]int)
	for _, entry := range afterHistory {
		versionCounts[entry.Version]++
	}
	for ver, count := range versionCounts {
		if count > 3 {
			t.Errorf("version %q appears %d times (expected <= 3)", ver, count)
		}
	}

	// История должна увеличиться (была версия добавлена от LoadFile при rollback)
	if len(afterHistory) <= len(beforeHistory) {
		t.Errorf("expected history to grow after rollback, was %d, now %d",
			len(beforeHistory), len(afterHistory))
	}

	// v1 должна быть активной
	if v := r.GetVersion("nd-playbook"); v != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", v)
	}
}
