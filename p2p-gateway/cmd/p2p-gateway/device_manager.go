package main

import (
	"fmt"
	"sync"

	"p2p-gateway/internal/models"
	"p2p-gateway/pkg/adapters"
)

type DeviceManager struct {
	mu       sync.RWMutex
	adapters map[string]adapters.DeviceAdapter
	devices  map[string]*models.Device
	nextPort int
}

func NewDeviceManager(startPort int) *DeviceManager {
	return &DeviceManager{
		adapters: make(map[string]adapters.DeviceAdapter),
		devices:  make(map[string]*models.Device),
		nextPort: startPort,
	}
}

func (dm *DeviceManager) RegisterAdapter(brand string, adapter adapters.DeviceAdapter) {
	dm.adapters[brand] = adapter
}

func (dm *DeviceManager) AddDevice(brand, serial, username, password, securityCode, ipAddress string) (*models.Device, error) {
	adapter, ok := dm.adapters[brand]
	if !ok {
		return nil, fmt.Errorf("unsupported brand: %s", brand)
	}
	dm.mu.Lock()
	port := dm.nextPort
	dm.nextPort += 2
	dm.mu.Unlock()

	dev := &models.Device{
		ID:           fmt.Sprintf("%s_%s", brand, serial),
		Brand:        brand,
		Serial:       serial,
		Username:     username,
		Password:     password,
		SecurityCode: securityCode,
		IPAddress:    ipAddress,
		ProxyPort:    port,
		Status:       models.StatusUnknown,
	}

	if err := adapter.Start(dev); err != nil {
		return nil, err
	}

	dm.mu.Lock()
	dm.devices[dev.ID] = dev
	dm.mu.Unlock()

	return dev, nil
}

func (dm *DeviceManager) StopDevice(deviceID string) error {
	dm.mu.RLock()
	dev, ok := dm.devices[deviceID]
	dm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("device not found")
	}
	adapter, ok := dm.adapters[dev.Brand]
	if !ok {
		return fmt.Errorf("adapter for brand %s not found", dev.Brand)
	}
	if err := adapter.Stop(dev); err != nil {
		return err
	}
	dm.mu.Lock()
	delete(dm.devices, deviceID)
	dm.mu.Unlock()
	return nil
}

// ShutdownAll останавливает все устройства в порядке, обратном регистрации.
// Возвращает слайс ошибок (nil при успешном shutdown).
func (dm *DeviceManager) ShutdownAll() []error {
	dm.mu.RLock()
	ids := make([]string, 0, len(dm.devices))
	for id := range dm.devices {
		ids = append(ids, id)
	}
	dm.mu.RUnlock()

	var errs []error
	for _, id := range ids {
		if err := dm.StopDevice(id); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", id, err))
		}
	}
	return errs
}

func (dm *DeviceManager) GetDevice(deviceID string) (*models.Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	dev, ok := dm.devices[deviceID]
	return dev, ok
}

func (dm *DeviceManager) GetDevices() []*models.Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	devices := make([]*models.Device, 0, len(dm.devices))
	for _, dev := range dm.devices {
		devices = append(devices, dev)
	}
	return devices
}

func (dm *DeviceManager) GetAdapter(brand string) (adapters.DeviceAdapter, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	adapter, ok := dm.adapters[brand]
	return adapter, ok
}

func (dm *DeviceManager) GetAdapterCounts() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	total := len(dm.adapters)
	active := 0
	for _, a := range dm.adapters {
		if a != nil {
			active++
		}
	}
	return fmt.Sprintf("%d registered, %d devices", total, len(dm.devices))
}
