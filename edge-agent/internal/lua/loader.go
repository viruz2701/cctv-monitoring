// Package lua implements a sandboxed Lua plugin loader for binary protocol
// handlers that cannot be expressed declaratively via Protocol Descriptor.
//
// Compliance:
//   - IEC 62443-3-3 SL-3 (Zone 5 — Edge): Sandboxed execution, no os/io/debug
//   - OWASP ASVS V5: Input validation through Lua API wrapper
//   - Приказ ОАЦ №66 п. 7.18: Plugins loaded from signed /usb/plugins (signature check TODO)
package lua

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// LuaPlugin represents a loaded Lua plugin instance.
type LuaPlugin struct {
	State  *lua.LState
	Vendor string
}

// LuaPluginLoader manages loading, listing and unloading of Lua plugins.
// Plugins are loaded from a filesystem path (default /usb/plugins/{vendor}.lua).
// Each plugin runs in a sandboxed Lua state with restricted libraries.
type LuaPluginLoader struct {
	pluginsPath string
	logger      *slog.Logger
	mu          sync.RWMutex
	loaded      map[string]*LuaPlugin
}

// NewLuaPluginLoader creates a new loader that reads plugins from pluginsPath.
// Logger is optional; if nil, a default no-op logger is used.
func NewLuaPluginLoader(pluginsPath string, logger *slog.Logger) *LuaPluginLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &LuaPluginLoader{
		pluginsPath: pluginsPath,
		logger:      logger,
		loaded:      make(map[string]*LuaPlugin),
	}
}

// LoadPlugin loads a Lua plugin by vendor name.
// It reads pluginsPath/{vendor}.lua, creates a sandboxed Lua state,
// registers the plugin API, executes the script, and verifies it
// returns a table with plugin functions.
func (l *LuaPluginLoader) LoadPlugin(vendor string) (*LuaPlugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if p, ok := l.loaded[vendor]; ok {
		return p, nil
	}

	// Validate vendor name (OWASP ASVS V5 — whitelist input validation)
	if err := validateVendor(vendor); err != nil {
		return nil, fmt.Errorf("lua: invalid vendor name %q: %w", vendor, err)
	}

	pluginPath := filepath.Join(l.pluginsPath, vendor+".lua")

	// Check file exists
	if _, err := os.Stat(pluginPath); err != nil {
		return nil, fmt.Errorf("lua: plugin file not found %q: %w", pluginPath, err)
	}

	// Create sandboxed Lua state
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	// Open only safe libraries (no os, io, debug, package)
	safeLibs := map[string]lua.LGFunction{
		"_G":        lua.OpenBase,
		"table":     lua.OpenTable,
		"string":    lua.OpenString,
		"math":      lua.OpenMath,
		"coroutine": lua.OpenCoroutine,
	}
	for name, fn := range safeLibs {
		if err := L.DoString(fmt.Sprintf("%s = {}", name)); err != nil {
			L.Close()
			return nil, fmt.Errorf("lua: failed to init %s lib: %w", name, err)
		}
		L.Push(L.NewFunction(fn))
		L.Call(0, 0)
	}

	// Remove dangerous globals from base library
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("module", lua.LNil)
	L.SetGlobal("load", lua.LNil)
	L.SetGlobal("rawget", lua.LNil)
	L.SetGlobal("rawset", lua.LNil)
	L.SetGlobal("rawequal", lua.LNil)
	L.SetGlobal("getmetatable", lua.LNil)
	L.SetGlobal("setmetatable", lua.LNil)

	// Register plugin API
	registerAPI(L, l.logger)

	// Load and execute plugin script
	if err := L.DoFile(pluginPath); err != nil {
		L.Close()
		return nil, fmt.Errorf("lua: failed to load plugin %q: %w", vendor, err)
	}

	// Verify plugin returns a table
	pluginTable := L.Get(-1)
	if pluginTable.Type() != lua.LTTable {
		L.Close()
		return nil, fmt.Errorf("lua: plugin %q must return a table, got %s", vendor, pluginTable.Type())
	}

	// Store the plugin table as a global so Lua scripts can reference it
	L.SetGlobal("plugin", pluginTable)
	L.Pop(1)

	plugin := &LuaPlugin{
		State:  L,
		Vendor: vendor,
	}
	l.loaded[vendor] = plugin
	l.logger.Info("lua plugin loaded", "vendor", vendor, "path", pluginPath)

	return plugin, nil
}

// ListPlugins returns a list of available plugin files in the plugins directory.
// Only files matching {name}.lua are returned.
func (l *LuaPluginLoader) ListPlugins() ([]string, error) {
	entries, err := os.ReadDir(l.pluginsPath)
	if err != nil {
		return nil, fmt.Errorf("lua: cannot read plugins dir %q: %w", l.pluginsPath, err)
	}

	var plugins []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".lua") {
			plugins = append(plugins, strings.TrimSuffix(name, ".lua"))
		}
	}
	return plugins, nil
}

// UnloadPlugin removes a loaded plugin and closes its Lua state.
// Safe to call multiple times for the same vendor.
func (l *LuaPluginLoader) UnloadPlugin(vendor string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	plugin, ok := l.loaded[vendor]
	if !ok {
		return
	}
	plugin.State.Close()
	delete(l.loaded, vendor)
	l.logger.Info("lua plugin unloaded", "vendor", vendor)
}

// GetPlugin returns a loaded plugin by vendor name, or nil if not loaded.
func (l *LuaPluginLoader) GetPlugin(vendor string) *LuaPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded[vendor]
}

// Loaded returns the number of currently loaded plugins.
func (l *LuaPluginLoader) Loaded() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.loaded)
}

// validateVendor checks that the vendor name contains only safe characters
// (OWASP ASVS V5 — whitelist validation).
func validateVendor(vendor string) error {
	if vendor == "" {
		return errors.New("vendor name cannot be empty")
	}
	if len(vendor) > 64 {
		return errors.New("vendor name too long (max 64 chars)")
	}
	for _, r := range vendor {
		if !isAllowedVendorChar(r) {
			return fmt.Errorf("invalid character %q in vendor name", r)
		}
	}
	return nil
}

func isAllowedVendorChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' || r == '-'
}
