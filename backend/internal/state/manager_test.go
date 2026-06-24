package state

import (
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
)

func TestNewInMemoryStateManager(t *testing.T) {
	m := NewInMemoryStateManager()
	if m == nil {
		t.Fatal("NewInMemoryStateManager returned nil")
	}
}

func TestSetAndGet(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID:   "cam-001",
		Name:       "Camera 1",
		Status:     models.StatusOnline,
		VendorType: "Hikvision",
		DeviceType: models.DeviceTypeCamera,
		AssetClass: models.AssetInternal,
		Health:     models.HealthHealthy,
		ConnectionType: models.ConnIP,
	}

	m.Set(dev)

	got, ok := m.Get("cam-001")
	if !ok {
		t.Fatal("expected device to be found")
	}
	if got.DeviceID != "cam-001" {
		t.Errorf("expected DeviceID=cam-001, got %s", got.DeviceID)
	}
	if got.Name != "Camera 1" {
		t.Errorf("expected Name=Camera 1, got %s", got.Name)
	}
	if got.Status != models.StatusOnline {
		t.Errorf("expected Status=ONLINE, got %s", got.Status)
	}
}

func TestGetNonExistent(t *testing.T) {
	m := NewInMemoryStateManager()

	_, ok := m.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent device")
	}
}

func TestDelete(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{DeviceID: "cam-001"}
	m.Set(dev)

	m.Delete("cam-001")

	_, ok := m.Get("cam-001")
	if ok {
		t.Error("expected device to be deleted")
	}
}

func TestGetAll(t *testing.T) {
	m := NewInMemoryStateManager()

	devices := []*models.Device{
		{DeviceID: "cam-001", Name: "Camera 1"},
		{DeviceID: "cam-002", Name: "Camera 2"},
		{DeviceID: "cam-003", Name: "Camera 3"},
	}

	for _, d := range devices {
		m.Set(d)
	}

	all := m.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 devices, got %d", len(all))
	}

	for _, d := range devices {
		if _, ok := all[d.DeviceID]; !ok {
			t.Errorf("expected device %s to be in GetAll", d.DeviceID)
		}
	}
}

func TestUpdateLastSeen(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID: "cam-001",
		LastSeen: time.Now().Add(-1 * time.Hour),
	}
	m.Set(dev)

	oldLastSeen := dev.LastSeen
	m.UpdateLastSeen("cam-001")

	updated, _ := m.Get("cam-001")
	if updated.LastSeen.Equal(oldLastSeen) {
		t.Error("expected LastSeen to be updated")
	}
}

func TestSetOnline(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID: "cam-001",
		Status:   models.StatusOffline,
	}
	m.Set(dev)

	m.SetOnline("cam-001")

	updated, _ := m.Get("cam-001")
	if updated.Status != models.StatusOnline {
		t.Errorf("expected Status=ONLINE, got %s", updated.Status)
	}
}

func TestSetOffline(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID: "cam-001",
		Status:   models.StatusOnline,
	}
	m.Set(dev)

	m.SetOffline("cam-001")

	updated, _ := m.Get("cam-001")
	if updated.Status != models.StatusOffline {
		t.Errorf("expected Status=OFFLINE, got %s", updated.Status)
	}
}

func TestSetOnlineIdempotent(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID: "cam-001",
		Status:   models.StatusOnline,
	}
	m.Set(dev)

	m.SetOnline("cam-001") // should not panic or change

	updated, _ := m.Get("cam-001")
	if updated.Status != models.StatusOnline {
		t.Errorf("expected Status=ONLINE, got %s", updated.Status)
	}
}

func TestAddAlarm(t *testing.T) {
	m := NewInMemoryStateManager()

	dev := &models.Device{
		DeviceID: "cam-001",
	}
	m.Set(dev)

	alarm := &models.Alarm{
		DeviceID:    "cam-001",
		Description: "Video loss detected",
	}

	m.AddAlarm("cam-001", alarm)

	updated, _ := m.Get("cam-001")
	if updated.LastAlarm == nil {
		t.Fatal("expected LastAlarm to be set")
	}
	if updated.LastAlarm.Description != "Video loss detected" {
		t.Errorf("expected description 'Video loss detected', got %q", updated.LastAlarm.Description)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := NewInMemoryStateManager()

	// Запускаем конкурентные операции
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			dev := &models.Device{
				DeviceID: "cam-001",
				Name:     "Camera 1",
			}
			m.Set(dev)
			m.Get("cam-001")
			m.GetAll()
			m.UpdateLastSeen("cam-001")
			m.SetOnline("cam-001")
			m.SetOffline("cam-001")
			m.AddAlarm("cam-001", &models.Alarm{DeviceID: "cam-001"})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestOperationsOnNonExistentDevice(t *testing.T) {
	m := NewInMemoryStateManager()

	// Эти операции не должны паниковать при работе с несуществующим устройством
	m.UpdateLastSeen("nonexistent")
	m.SetOnline("nonexistent")
	m.SetOffline("nonexistent")
	m.AddAlarm("nonexistent", &models.Alarm{DeviceID: "nonexistent"})
}
