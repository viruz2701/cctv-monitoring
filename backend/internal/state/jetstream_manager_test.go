// Package state — tests for JetStream KV State Manager.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — distributed state)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - OWASP ASVS V1.8 (Architecture — stateless design)
package state

import (
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"

	"github.com/nats-io/nats.go"
)

// TestJetStreamNewManager_NilJetStream проверяет обработку nil JetStream контекста.
func TestJetStreamNewManager_NilJetStream(t *testing.T) {
	_, err := NewJetStreamStateManager(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil JetStream context")
	}
}

// TestJetStreamSetAndGet проверяет базовую операцию Set/Get через NATS KV.
//
// Требует: работающий NATS сервер на nats://localhost:4222
// Если NATS недоступен — тест пропускается (dev mode).
func TestJetStreamSetAndGet(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available, skipping JetStream KV test")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{
		DeviceID:       "js-cam-001",
		Name:           "JetStream Camera 1",
		Status:         models.StatusOnline,
		VendorType:     "Hikvision",
		DeviceType:     models.DeviceTypeCamera,
		AssetClass:     models.AssetInternal,
		Health:         models.HealthHealthy,
		ConnectionType: models.ConnIP,
		LastSeen:       time.Now(),
		RegisteredAt:   time.Now(),
	}

	mgr.Set(dev)

	got, ok := mgr.Get("js-cam-001")
	if !ok {
		t.Fatal("expected device to be found")
	}
	if got.DeviceID != "js-cam-001" {
		t.Errorf("expected DeviceID=js-cam-001, got %s", got.DeviceID)
	}
	if got.Name != "JetStream Camera 1" {
		t.Errorf("expected Name='JetStream Camera 1', got %s", got.Name)
	}
	if got.Status != models.StatusOnline {
		t.Errorf("expected Status=ONLINE, got %s", got.Status)
	}
}

// TestJetStreamGetNonExistent проверяет Get несуществующего устройства.
func TestJetStreamGetNonExistent(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	_, ok := mgr.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent device")
	}
}

// TestJetStreamDelete проверяет удаление устройства.
func TestJetStreamDelete(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{DeviceID: "js-cam-to-delete"}
	mgr.Set(dev)

	mgr.Delete("js-cam-to-delete")

	_, ok := mgr.Get("js-cam-to-delete")
	if ok {
		t.Error("expected device to be deleted")
	}
}

// TestJetStreamGetAll проверяет GetAll после нескольких Set.
func TestJetStreamGetAll(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	devices := []*models.Device{
		{DeviceID: "js-cam-001", Name: "Camera 1"},
		{DeviceID: "js-cam-002", Name: "Camera 2"},
		{DeviceID: "js-cam-003", Name: "Camera 3"},
	}

	for _, d := range devices {
		mgr.Set(d)
	}

	all := mgr.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 devices, got %d", len(all))
	}

	for _, d := range devices {
		if _, ok := all[d.DeviceID]; !ok {
			t.Errorf("expected device %s to be in GetAll", d.DeviceID)
		}
	}
}

// TestJetStreamUpdateLastSeen проверяет обновление LastSeen.
func TestJetStreamUpdateLastSeen(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{
		DeviceID: "js-cam-001",
		LastSeen: time.Now().Add(-1 * time.Hour),
	}
	mgr.Set(dev)

	oldLastSeen := dev.LastSeen
	mgr.UpdateLastSeen("js-cam-001")

	updated, ok := mgr.Get("js-cam-001")
	if !ok {
		t.Fatal("expected device to exist after UpdateLastSeen")
	}
	if updated.LastSeen.Equal(oldLastSeen) {
		t.Error("expected LastSeen to be updated")
	}
}

// TestJetStreamSetOnline проверяет установку статуса ONLINE.
func TestJetStreamSetOnline(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{
		DeviceID: "js-cam-001",
		Status:   models.StatusOffline,
	}
	mgr.Set(dev)

	mgr.SetOnline("js-cam-001")

	updated, _ := mgr.Get("js-cam-001")
	if updated.Status != models.StatusOnline {
		t.Errorf("expected Status=ONLINE, got %s", updated.Status)
	}
}

// TestJetStreamSetOffline проверяет установку статуса OFFLINE.
func TestJetStreamSetOffline(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{
		DeviceID: "js-cam-001",
		Status:   models.StatusOnline,
	}
	mgr.Set(dev)

	mgr.SetOffline("js-cam-001")

	updated, _ := mgr.Get("js-cam-001")
	if updated.Status != models.StatusOffline {
		t.Errorf("expected Status=OFFLINE, got %s", updated.Status)
	}
}

// TestJetStreamAddAlarm проверяет добавление тревоги к устройству.
func TestJetStreamAddAlarm(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	dev := &models.Device{
		DeviceID: "js-cam-001",
	}
	mgr.Set(dev)

	alarm := &models.Alarm{
		DeviceID:    "js-cam-001",
		Description: "Video loss detected",
	}

	mgr.AddAlarm("js-cam-001", alarm)

	updated, _ := mgr.Get("js-cam-001")
	if updated.LastAlarm == nil {
		t.Fatal("expected LastAlarm to be set")
	}
	if updated.LastAlarm.Description != "Video loss detected" {
		t.Errorf("expected description 'Video loss detected', got %q", updated.LastAlarm.Description)
	}
}

// TestJetStreamConcurrentAccess проверяет конкурентный доступ к JetStream State Manager.
func TestJetStreamConcurrentAccess(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			dev := &models.Device{
				DeviceID: "js-cam-001",
				Name:     "Camera 1",
			}
			mgr.Set(dev)
			mgr.Get("js-cam-001")
			mgr.GetAll()
			mgr.UpdateLastSeen("js-cam-001")
			mgr.SetOnline("js-cam-001")
			mgr.SetOffline("js-cam-001")
			mgr.AddAlarm("js-cam-001", &models.Alarm{DeviceID: "js-cam-001"})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestJetStreamMetrics проверяет работу метрик.
func TestJetStreamMetrics(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	// Выполняем несколько операций
	dev := &models.Device{DeviceID: "js-cam-metrics"}
	mgr.Set(dev)
	mgr.Get("js-cam-metrics")
	mgr.Get("nonexistent")
	mgr.Delete("js-cam-metrics")

	metrics := mgr.Metrics()
	if metrics["total_sets"] < 1 {
		t.Errorf("expected total_sets >= 1, got %d", metrics["total_sets"])
	}
	if metrics["total_gets"] < 2 {
		t.Errorf("expected total_gets >= 2, got %d", metrics["total_gets"])
	}
	if metrics["cache_misses"] < 1 {
		t.Errorf("expected cache_misses >= 1 (at least one miss for nonexistent), got %d", metrics["cache_misses"])
	}
}

// TestJetStreamOperationsOnNonExistent проверяет операции с несуществующим устройством.
func TestJetStreamOperationsOnNonExistent(t *testing.T) {
	js := connectToNATSForTest(t)
	if js == nil {
		t.Skip("NATS not available")
	}

	mgr, err := NewJetStreamStateManager(js, nil)
	if err != nil {
		t.Fatalf("NewJetStreamStateManager failed: %v", err)
	}
	defer mgr.Stop()

	// Эти операции не должны паниковать
	mgr.UpdateLastSeen("nonexistent")
	mgr.SetOnline("nonexistent")
	mgr.SetOffline("nonexistent")
	mgr.AddAlarm("nonexistent", &models.Alarm{DeviceID: "nonexistent"})
}

// connectToNATSForTest подключается к NATS для тестов.
// Возвращает nil если NATS недоступен.
func connectToNATSForTest(t *testing.T) nats.JetStreamContext {
	t.Helper()

	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		return nil
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil
	}

	t.Cleanup(func() {
		// Очищаем тестовый bucket
		if err := js.DeleteKeyValue(KVDeviceBucket); err != nil {
			// Bucket может не существовать — это нормально
		}
		nc.Close()
	})

	return js
}
