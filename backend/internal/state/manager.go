package state

import (
	"gb-telemetry-collector/internal/models"
	"sync"
	"time"
)

type DeviceStateManager interface {
    Get(deviceID string) (*models.Device, bool)
    Set(device *models.Device)
    Delete(deviceID string)
    GetAll() map[string]*models.Device
    UpdateLastSeen(deviceID string)
    SetOnline(deviceID string)
    SetOffline(deviceID string)
    AddAlarm(deviceID string, alarm *models.Alarm)
}

type InMemoryStateManager struct {
	devices sync.Map // map[string]*models.Device
}

func NewInMemoryStateManager() *InMemoryStateManager {
	return &InMemoryStateManager{}
}

func (m *InMemoryStateManager) Get(deviceID string) (*models.Device, bool) {
	val, ok := m.devices.Load(deviceID)
	if !ok {
		return nil, false
	}
	return val.(*models.Device), true
}

func (m *InMemoryStateManager) Set(device *models.Device) {
	m.devices.Store(device.DeviceID, device)
}

func (m *InMemoryStateManager) Delete(deviceID string) {
    m.devices.Delete(deviceID)
}

func (m *InMemoryStateManager) GetAll() map[string]*models.Device {
	result := make(map[string]*models.Device)
	m.devices.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*models.Device)
		return true
	})
	return result
}

func (m *InMemoryStateManager) UpdateLastSeen(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		dev.LastSeen = time.Now()
		m.Set(dev)
	}
}

func (m *InMemoryStateManager) SetOnline(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		if dev.Status != models.StatusOnline {
			dev.Status = models.StatusOnline
			m.Set(dev)
			// TODO: послать событие в шину (EventBus)
		}
	}
}

func (m *InMemoryStateManager) SetOffline(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		if dev.Status != models.StatusOffline {
			dev.Status = models.StatusOffline
			m.Set(dev)
			// TODO: событие DEVICE_OFFLINE
		}
	}
}

func (m *InMemoryStateManager) AddAlarm(deviceID string, alarm *models.Alarm) {
	if dev, ok := m.Get(deviceID); ok {
		dev.LastAlarm = alarm
		m.Set(dev)
		// TODO: сохранить в БД, отправить в шину
	}
}
