// Package lua tests — unit tests for LuaPluginLoader and Plugin API.
//
// Compliance tests:
//   - IEC 62443-3-3 SL-3: Sandbox verification (no os/io/debug)
//   - OWASP ASVS V5: Input validation (vendor name whitelist)
//   - Приказ ОАЦ №66 п. 7.18: Plugin loading from filesystem
package lua

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// testLogger is a discard logger for tests.
var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// setupTestPlugins creates temporary plugin files for testing.
func setupTestPlugins(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "lua-plugins-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Valid plugin
	validPlugin := `local plugin = {}
function plugin.greet(agent, name)
    return "Hello, " .. name
end
return plugin`
	if err := os.WriteFile(filepath.Join(dir, "test_vendor.lua"), []byte(validPlugin), 0644); err != nil {
		t.Fatalf("failed to write plugin: %v", err)
	}

	// Plugin that doesn't return a table
	badPlugin := `return "not a table"`
	if err := os.WriteFile(filepath.Join(dir, "bad_plugin.lua"), []byte(badPlugin), 0644); err != nil {
		t.Fatalf("failed to write bad plugin: %v", err)
	}

	// Non-Lua file
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a plugin"), 0644); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}
	return dir, cleanup
}

// ============================================================================
// LuaPluginLoader tests
// ============================================================================

func TestNewLuaPluginLoader(t *testing.T) {
	loader := NewLuaPluginLoader("/var/lib/edge/plugins", nil)
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
	if loader.pluginsPath != "/var/lib/edge/plugins" {
		t.Errorf("expected pluginsPath=/var/lib/edge/plugins, got %s", loader.pluginsPath)
	}
	if loader.logger == nil {
		t.Error("expected non-nil logger")
	}
	if loader.loaded == nil {
		t.Error("expected non-nil loaded map")
	}
}

func TestLoadPlugin_Success(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}
	if plugin.Vendor != "test_vendor" {
		t.Errorf("expected vendor=test_vendor, got %s", plugin.Vendor)
	}
	if plugin.State == nil {
		t.Fatal("expected non-nil Lua state")
	}
}

func TestLoadPlugin_NotFound(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	_, err := loader.LoadPlugin("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestLoadPlugin_InvalidVendorName(t *testing.T) {
	tests := []struct {
		name    string
		vendor  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too long", strings.Repeat("a", 65), true},
		{"with spaces", "test vendor", true},
		{"with slashes", "test/vendor", true},
		{"with dots", "test.vendor", true},
		{"valid lowercase", "hikvision", false},
		{"valid mixed", "HikVision", false},
		{"valid with digits", "h264", false},
		{"valid with hyphen", "hik-vision", false},
		{"valid with underscore", "hik_vision", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVendor(tt.vendor)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
func TestLoadPlugin_BadPlugin(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	_, err := loader.LoadPlugin("bad_plugin")
	if err == nil {
		t.Fatal("expected error for bad plugin (must return table)")
	}
	if !strings.Contains(err.Error(), "must return a table") {
		t.Errorf("expected 'must return a table' in error, got: %v", err)
	}
}

func TestLoadPlugin_Duplicate(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)

	plugin1, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	plugin2, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	// Should return the same instance
	if plugin1 != plugin2 {
		t.Error("expected same plugin instance on duplicate load")
	}
}

func TestListPlugins(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugins, err := loader.ListPlugins()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %v", len(plugins), plugins)
	}

	pluginSet := make(map[string]bool)
	for _, p := range plugins {
		pluginSet[p] = true
	}
	if !pluginSet["test_vendor"] {
		t.Error("expected test_vendor in plugin list")
	}
	if !pluginSet["bad_plugin"] {
		t.Error("expected bad_plugin in plugin list")
	}
}

func TestListPlugins_NonExistentDir(t *testing.T) {
	loader := NewLuaPluginLoader("/var/lib/edge/nonexistent", testLogger)
	_, err := loader.ListPlugins()
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestUnloadPlugin(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)

	// Load and unload
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if plugin.State == nil {
		t.Fatal("expected non-nil Lua state before unload")
	}

	loader.UnloadPlugin("test_vendor")
	if loader.Loaded() != 0 {
		t.Errorf("expected 0 loaded plugins, got %d", loader.Loaded())
	}
}

func TestUnloadPlugin_NotFound(t *testing.T) {
	loader := NewLuaPluginLoader("/var/lib/edge/plugins", testLogger)
	// Should not panic
	loader.UnloadPlugin("nonexistent")
}

func TestGetPlugin(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)

	// Not loaded yet
	if p := loader.GetPlugin("test_vendor"); p != nil {
		t.Error("expected nil for not-loaded plugin")
	}

	// Load and get
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	got := loader.GetPlugin("test_vendor")
	if got != plugin {
		t.Error("GetPlugin returned different instance")
	}
}

func TestLoaded(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	if loader.Loaded() != 0 {
		t.Errorf("expected 0, got %d", loader.Loaded())
	}

	loader.LoadPlugin("test_vendor")
	if loader.Loaded() != 1 {
		t.Errorf("expected 1, got %d", loader.Loaded())
	}

	loader.UnloadPlugin("test_vendor")
	if loader.Loaded() != 0 {
		t.Errorf("expected 0, got %d", loader.Loaded())
	}
}

// ============================================================================
// Sandbox compliance tests (IEC 62443-3-3 SL-3)
// ============================================================================

func TestSandbox_RestrictedLibraries(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify restricted libraries are not available
	restricted := []string{"os", "io", "debug", "package"}
	for _, lib := range restricted {
		val := plugin.State.GetGlobal(lib)
		if val.Type() != lua.LTNil {
			t.Errorf("restricted library %q should be nil, got %s", lib, val.Type())
		}
	}
}

func TestSandbox_RestrictedFunctions(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	restricted := []string{"dofile", "loadfile", "require", "module", "load"}
	for _, fn := range restricted {
		val := plugin.State.GetGlobal(fn)
		if val.Type() != lua.LTNil {
			t.Errorf("restricted function %q should be nil, got %s", fn, val.Type())
		}
	}
}

func TestSandbox_SafeLibrariesPresent(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	safe := []string{"table", "string", "math"}
	for _, lib := range safe {
		val := plugin.State.GetGlobal(lib)
		if val.Type() != lua.LTTable {
			t.Errorf("safe library %q should be a table, got %s", lib, val.Type())
		}
	}
}

func TestSandbox_APIAvailable(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	agent := plugin.State.GetGlobal("agent")
	if agent.Type() != lua.LTTable {
		t.Fatalf("agent should be a table, got %s", agent.Type())
	}

	expectedAPIs := []string{"http_get", "http_post", "xml_parse", "json_parse", "log"}
	for _, name := range expectedAPIs {
		fn := plugin.State.GetField(agent.(*lua.LTable), name)
		if fn.Type() != lua.LTFunction {
			t.Errorf("agent.%s should be a function, got %s", name, fn.Type())
		}
	}
}

// ============================================================================
// Plugin execution tests
// ============================================================================

func TestPluginExecution(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)
	plugin, err := loader.LoadPlugin("test_vendor")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Call plugin.greet via Lua — the plugin returns a table, so we
	// need to get the 'plugin' global and then call its 'greet' method
	pluginTable := plugin.State.GetGlobal("plugin")
	if pluginTable.Type() != lua.LTTable {
		t.Fatalf("expected plugin to be a table, got %s", pluginTable.Type())
	}

	greetFn := plugin.State.GetField(pluginTable.(*lua.LTable), "greet")
	if greetFn.Type() != lua.LTFunction {
		t.Fatalf("expected plugin.greet to be a function, got %s", greetFn.Type())
	}

	// Call plugin.greet(agent, name) — greet expects 2 args
	if err := plugin.State.CallByParam(lua.P{
		Fn:      greetFn,
		NRet:    1,
		Protect: true,
	}, lua.LNil, lua.LString("World")); err != nil {
		t.Fatalf("plugin call failed: %v", err)
	}

	ret := plugin.State.Get(-1)
	plugin.State.Pop(1)
	if ret.String() != "Hello, World" {
		t.Errorf("expected 'Hello, World', got %s", ret.String())
	}
}

// ============================================================================
// API function tests (without HTTP)
// ============================================================================

func TestJSONParse(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	registerAPI(L, testLogger)

	if err := L.DoString(`
		local result = agent.json_parse('{"name":"test","value":42,"active":true,"nested":{"key":"val"}}')
		assert(result.name == "test", "name should be test")
		assert(result.value == 42, "value should be 42")
		assert(result.active == true, "active should be true")
		assert(result.nested.key == "val", "nested.key should be val")
	`); err != nil {
		t.Fatalf("json_parse test failed: %v", err)
	}
}

func TestJSONParse_Array(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	registerAPI(L, testLogger)

	if err := L.DoString(`
		local result = agent.json_parse('[10, 20, 30]')
		assert(result[1] == 10, "result[1] should be 10")
		assert(result[2] == 20, "result[2] should be 20")
		assert(result[3] == 30, "result[3] should be 30")
	`); err != nil {
		t.Fatalf("json_parse array test failed: %v", err)
	}
}

func TestJSONParse_InvalidInput(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	registerAPI(L, testLogger)

	if err := L.DoString(`
		local result, err = agent.json_parse('not valid json')
		assert(result == nil, "result should be nil")
		assert(err ~= nil, "err should not be nil")
	`); err != nil {
		t.Fatalf("json_parse invalid test failed: %v", err)
	}
}

func TestXMLParse(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	registerAPI(L, testLogger)

	xmlStr := "<deviceInfo><modelName>DS-2CD2T47G2-L</modelName><serialNumber>ABCDEF123</serialNumber><firmwareVersion>V5.6.0</firmwareVersion></deviceInfo>"

	if err := L.DoString(`
		local result = agent.xml_parse([['` + xmlStr + `']])
		assert(result.modelName == "DS-2CD2T47G2-L", "modelName should match")
		assert(result.serialNumber == "ABCDEF123", "serialNumber should match")
		assert(result.firmwareVersion == "V5.6.0", "firmwareVersion should match")
	`); err != nil {
		t.Fatalf("xml_parse test failed: %v", err)
	}
}

func TestLog(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logBuf := &strings.Builder{}
	logger := slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registerAPI(L, logger)

	if err := L.DoString(`
		agent.log("info", "test message")
		agent.log("error", "error message")
	`); err != nil {
		t.Fatalf("log test failed: %v", err)
	}

	output := logBuf.String()
	if !strings.Contains(output, "test message") {
		t.Error("expected test message in log output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("expected error message in log output")
	}
}

// ============================================================================
// Concurrency tests
// ============================================================================

func TestConcurrentLoad(t *testing.T) {
	dir, cleanup := setupTestPlugins(t)
	defer cleanup()

	loader := NewLuaPluginLoader(dir, testLogger)

	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := loader.LoadPlugin("test_vendor")
			if err != nil {
				t.Errorf("concurrent load failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 2; i++ {
		<-done
	}

	if loader.Loaded() != 1 {
		t.Errorf("expected 1 loaded plugin after concurrent loads, got %d", loader.Loaded())
	}
}

// ============================================================================
// Benchmark
// ============================================================================

func BenchmarkLoadPlugin(b *testing.B) {
	dir, err := os.MkdirTemp("", "lua-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	validPlugin := `local plugin = {}
function plugin.greet(agent, name)
    return "Hello, " .. name
end
return plugin`
	if err := os.WriteFile(filepath.Join(dir, "bench_vendor.lua"), []byte(validPlugin), 0644); err != nil {
		b.Fatalf("failed to write plugin: %v", err)
	}

	loader := NewLuaPluginLoader(dir, testLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin, err := loader.LoadPlugin("bench_vendor")
		if err != nil {
			b.Fatalf("load failed: %v", err)
		}
		loader.UnloadPlugin(plugin.Vendor)
	}
}
