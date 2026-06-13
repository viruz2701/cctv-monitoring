package main

import (
	"fmt"
	"sync"
	"time"
)

type DeviceStatus string

const (
	StatusUnknown DeviceStatus = "unknown"
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
)

type Device struct {
	ID           string       `json:"id"`
	Brand        string       `json:"brand"`
	Serial       string       `json:"serial"`
	Username     string       `json:"username,omitempty"`
	Password     string       `json:"password,omitempty"`
	SecurityCode string       `json:"security_code,omitempty"`
	ProxyPort    int          `json:"proxy_port"`
	RTSPURL      string       `json:"rtsp_url"`
	Status       DeviceStatus `json:"status"`
	LastSeen     time.Time    `json:"last_seen"`
}

type DeviceManager struct {
	mu       sync.RWMutex
	devices  map[string]*Device
	adapters map[string]DeviceAdapter
	cfg      *Config
}

func NewDeviceManager(cfg *Config) *DeviceManager {
	return &DeviceManager{
		devices:  make(map[string]*Device),
		adapters: make(map[string]DeviceAdapter),
		cfg:      cfg,
	}
}

func (dm *DeviceManager) RegisterAdapter(brand string, adapter DeviceAdapter) {
	dm.adapters[brand] = adapter
}

func (dm *DeviceManager) AddDevice(dev *Device) error {
	adapter, ok := dm.adapters[dev.Brand]
	if !ok {
		return fmt.Errorf("unsupported brand: %s", dev.Brand)
	}
	// Assign unique ports
	dev.ProxyPort = dm.cfg.ProxyBaseRTSPPort + len(dm.devices)*2
	// Start proxy process
	if err := adapter.Start(dev); err != nil {
		return err
	}
	dm.mu.Lock()
	dm.devices[dev.ID] = dev
	dm.mu.Unlock()
	return nil
}

func (dm *DeviceManager) GetDevice(id string) (*Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	dev, ok := dm.devices[id]
	return dev, ok
}

func (dm *DeviceManager) GetAllDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	list := make([]*Device, 0, len(dm.devices))
	for _, dev := range dm.devices {
		list = append(list, dev)
	}
	return list
}

func (dm *DeviceManager) UpdateStatus(devID string, status DeviceStatus) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dev, ok := dm.devices[devID]; ok {
		dev.Status = status
		dev.LastSeen = time.Now()
	}
}
