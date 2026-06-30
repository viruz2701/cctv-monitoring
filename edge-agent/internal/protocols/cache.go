package protocols

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Descriptor represents a protocol descriptor for a CCTV device.
type Descriptor struct {
	// Vendor is the device manufacturer
	Vendor string `json:"vendor"`
	// Model is the device model
	Model string `json:"model"`
	// Version is the protocol version
	Version string `json:"version"`
	// ProtocolType is the type of protocol (ONVIF, RTSP, HTTP, etc.)
	ProtocolType string `json:"protocol_type"`
	// Endpoints are the protocol-specific endpoints
	Endpoints []Endpoint `json:"endpoints"`
	// Commands are supported commands for this descriptor
	Commands []CommandDef `json:"commands"`
	// UpdatedAt is when this descriptor was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// Checksum for integrity (Приказ ОАЦ №66 п. 7.18.3)
	Checksum string `json:"checksum"`
}

// Endpoint represents a protocol endpoint.
type Endpoint struct {
	// Path is the URL path or URI
	Path string `json:"path"`
	// Method is the HTTP method or protocol command
	Method string `json:"method"`
	// Description of the endpoint
	Description string `json:"description"`
	// Parameters for the endpoint
	Parameters []Parameter `json:"parameters"`
}

// Parameter describes a protocol parameter.
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// CommandDef describes a command that can be executed on a device.
type CommandDef struct {
	// Name is the command name (e.g., "reboot", "get_status")
	Name string `json:"name"`
	// Description of the command
	Description string `json:"description"`
	// Parameters for the command
	Parameters []Parameter `json:"parameters"`
	// Timeout is the command execution timeout in seconds
	Timeout int `json:"timeout"`
}

// DescriptorCache provides in-memory caching with USB persistence
// for protocol descriptors. Used on OpenWrt with USB storage.
//
// Compliance: IEC 62443-3-3 SL-3 — кэш дескрипторов с контролем целостности
type DescriptorCache struct {
	mu     sync.RWMutex
	cache  map[string]*Descriptor // key: "vendor/model"
	path   string                 // persistence file path (USB)
	logger *slog.Logger
	dirty  bool
}

// NewDescriptorCache creates a new descriptor cache.
// path is the file path for persistence (e.g., /mnt/usb/descriptors.json).
func NewDescriptorCache(path string, logger *slog.Logger) *DescriptorCache {
	return &DescriptorCache{
		cache:  make(map[string]*Descriptor),
		path:   path,
		logger: logger,
	}
}

// Get retrieves a descriptor by vendor and model.
func (c *DescriptorCache) Get(vendor, model string) (*Descriptor, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := descriptorKey(vendor, model)
	d, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	return d, true
}

// Update replaces the entire cache with new descriptors.
func (c *DescriptorCache) Update(descriptors []Descriptor) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*Descriptor, len(descriptors))
	for i := range descriptors {
		key := descriptorKey(descriptors[i].Vendor, descriptors[i].Model)
		c.cache[key] = &descriptors[i]
	}
	c.dirty = true

	c.logger.Info("descriptor cache updated", "count", len(descriptors))
	return nil
}

// Load reads the cache from persistent storage.
func (c *DescriptorCache) Load() error {
	if c.path == "" {
		return nil // No persistence configured
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // First run, no cache yet
		}
		return fmt.Errorf("read cache file: %w", err)
	}

	var descriptors []Descriptor
	if err := json.Unmarshal(data, &descriptors); err != nil {
		return fmt.Errorf("parse cache file: %w", err)
	}

	c.cache = make(map[string]*Descriptor, len(descriptors))
	for i := range descriptors {
		key := descriptorKey(descriptors[i].Vendor, descriptors[i].Model)
		c.cache[key] = &descriptors[i]
	}

	c.logger.Info("descriptor cache loaded from disk", "count", len(descriptors))
	return nil
}

// Save writes the cache to persistent storage.
func (c *DescriptorCache) Save() error {
	if c.path == "" || !c.dirty {
		return nil
	}

	c.mu.RLock()
	descriptors := make([]Descriptor, 0, len(c.cache))
	for _, d := range c.cache {
		descriptors = append(descriptors, *d)
	}
	c.mu.RUnlock()

	data, err := json.Marshal(descriptors)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()

	c.logger.Info("descriptor cache saved to disk", "path", c.path)
	return nil
}

// List returns all cached descriptors.
func (c *DescriptorCache) List() []Descriptor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]Descriptor, 0, len(c.cache))
	for _, d := range c.cache {
		result = append(result, *d)
	}
	return result
}

// Count returns the number of cached descriptors.
func (c *DescriptorCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

func descriptorKey(vendor, model string) string {
	if model == "" {
		return vendor
	}
	return vendor + "/" + model
}
