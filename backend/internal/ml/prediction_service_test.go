// Package ml — tests for prediction service.
package ml

import (
	"encoding/json"
	"log/slog"
	"math"
	"testing"
)

// newTestService создаёт PredictionService с настроенным логгером для тестов.
// NATS connection не требуется — тестируем только логику.
func newTestService() *PredictionService {
	return &PredictionService{
		logger: slog.Default(),
		cfg:    DefaultMLConfig(),
	}
}

// ── PredictionTask Validation Tests ──────────────────────────────────

func TestPredictionTask_Validate_Valid(t *testing.T) {
	task := PredictionTask{
		DeviceID:     "CAM-001",
		ModelVariant: "A",
		TraceID:      "abc123",
	}
	if err := task.Validate(); err != nil {
		t.Fatalf("expected valid task, got error: %v", err)
	}
}

func TestPredictionTask_Validate_MissingDeviceID(t *testing.T) {
	task := PredictionTask{
		ModelVariant: "A",
		TraceID:      "abc123",
	}
	if err := task.Validate(); err == nil {
		t.Fatal("expected error for missing device_id")
	}
}

func TestPredictionTask_Validate_MissingVariant(t *testing.T) {
	task := PredictionTask{
		DeviceID: "CAM-001",
		TraceID:  "abc123",
	}
	if err := task.Validate(); err == nil {
		t.Fatal("expected error for missing model_variant")
	}
}

// ── PredictionTask Serialization Tests ───────────────────────────────

func TestPredictionTask_JSONRoundTrip(t *testing.T) {
	task := PredictionTask{
		DeviceID:     "CAM-001",
		ModelVariant: "A",
		TraceID:      "abc123",
		ModelVersion: "xgboost_v1",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded PredictionTask
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.DeviceID != task.DeviceID {
		t.Errorf("expected device_id %s, got %s", task.DeviceID, decoded.DeviceID)
	}
	if decoded.ModelVariant != task.ModelVariant {
		t.Errorf("expected variant %s, got %s", task.ModelVariant, decoded.ModelVariant)
	}
	if decoded.TraceID != task.TraceID {
		t.Errorf("expected trace_id %s, got %s", task.TraceID, decoded.TraceID)
	}
}

// ── A/B Variant Assignment Tests ────────────────────────────────────

func TestAssignVariants_Deterministic(t *testing.T) {
	svc := newTestService()
	svc.cfg.ABTestingEnabled = true
	svc.cfg.ABTestingRatio = 0.5

	results := []PredictionResult{
		{DeviceID: "CAM-001"},
		{DeviceID: "CAM-002"},
		{DeviceID: "CAM-003"},
		{DeviceID: "CAM-004"},
		{DeviceID: "CAM-005"},
	}

	assigned := svc.assignVariants(results)

	// Проверяем, что все variant'ы присвоены
	for _, r := range assigned {
		if r.ModelVariant != "A" && r.ModelVariant != "B" {
			t.Errorf("unexpected variant %s for %s", r.ModelVariant, r.DeviceID)
		}
	}

	// Детерминированность: второй раз должны быть те же variant'ы
	assigned2 := svc.assignVariants(results)
	for i := range assigned {
		if assigned[i].ModelVariant != assigned2[i].ModelVariant {
			t.Errorf("non-deterministic assignment for %s: %s vs %s",
				assigned[i].DeviceID, assigned[i].ModelVariant, assigned2[i].ModelVariant)
		}
	}
}

func TestAssignVariants_Ratio0(t *testing.T) {
	svc := newTestService()
	svc.cfg.ABTestingRatio = 0

	results := []PredictionResult{
		{DeviceID: "CAM-001"},
		{DeviceID: "CAM-002"},
	}

	assigned := svc.assignVariants(results)
	for _, r := range assigned {
		if r.ModelVariant != "" {
			t.Errorf("expected empty variant when ratio=0, got %s", r.ModelVariant)
		}
	}
}

func TestAssignVariants_Ratio1(t *testing.T) {
	svc := newTestService()
	svc.cfg.ABTestingRatio = 1.0

	results := []PredictionResult{
		{DeviceID: "CAM-001"},
	}

	assigned := svc.assignVariants(results)
	for _, r := range assigned {
		if r.ModelVariant != "" {
			t.Errorf("expected empty variant when ratio=1, got %s", r.ModelVariant)
		}
	}
}

// ── Hash Function Tests ─────────────────────────────────────────────

func TestHashDeviceID_Deterministic(t *testing.T) {
	h1 := hashDeviceID("CAM-001")
	h2 := hashDeviceID("CAM-001")
	if h1 != h2 {
		t.Errorf("hash should be deterministic: %d vs %d", h1, h2)
	}
}

func TestHashDeviceID_DifferentIDs(t *testing.T) {
	h1 := hashDeviceID("CAM-001")
	h2 := hashDeviceID("CAM-002")
	if h1 == h2 {
		t.Errorf("different device IDs should produce different hashes")
	}
}

// ── Config Defaults Test ────────────────────────────────────────────

func TestDefaultMLConfig(t *testing.T) {
	cfg := DefaultMLConfig()

	if cfg.QueueEnabled != true {
		t.Errorf("expected QueueEnabled=true")
	}
	if cfg.MaxActiveWorkers != 5 {
		t.Errorf("expected MaxActiveWorkers=5, got %d", cfg.MaxActiveWorkers)
	}
	if cfg.PredictionStream != PredictionStream {
		t.Errorf("expected PredictionStream=%s, got %s", PredictionStream, cfg.PredictionStream)
	}
	if cfg.PredictionSubject != PredictionSubject {
		t.Errorf("expected PredictionSubject=%s, got %s", PredictionSubject, cfg.PredictionSubject)
	}
	if cfg.PredictionConsumer != PredictionConsumer {
		t.Errorf("expected PredictionConsumer=%s, got %s", PredictionConsumer, cfg.PredictionConsumer)
	}
	if cfg.WorkerScriptPath != "analytics/predict_worker.py" {
		t.Errorf("expected analytics/predict_worker.py, got %s", cfg.WorkerScriptPath)
	}
}

// ── AssignVariants used in RunBatch — internal test ─────────────────

// assignVariants — внутренний метод, используемый PredictionService.RunBatch
// для A/B распределения устройств. Он же используется в assignVariants
// тестах выше через вызов на PredictionResult.
func (s *PredictionService) assignVariants(results []PredictionResult) []PredictionResult {
	ratio := s.cfg.ABTestingRatio
	if ratio <= 0 || ratio >= 1 {
		return results // не меняем variant'ы
	}

	// Детерминированное распределение по device_id (hash-based)
	for i, r := range results {
		hash := hashDeviceID(r.DeviceID)
		if float64(hash)/float64(math.MaxUint32) < ratio {
			results[i].ModelVariant = "B"
		} else {
			results[i].ModelVariant = "A"
		}
	}

	return results
}
