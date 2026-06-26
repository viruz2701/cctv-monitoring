// Package ml — tests for prediction service.
package ml

import (
	"bufio"
	"log/slog"
	"strings"
	"testing"
)

// newTestService создаёт PredictionService с настроенным логгером для тестов.
func newTestService() *PredictionService {
	return &PredictionService{
		logger: slog.Default(),
		cfg:    DefaultMLConfig(),
	}
}

// ── JSONL Parser Tests ──────────────────────────────────────────────

func TestParseOutput_ValidJSONL(t *testing.T) {
	svc := newTestService()
	input := `{"device_id":"CAM-001","failure_probability":0.87,"confidence_score":0.92,"model_version":"xgboost_v1","model_variant":"A","prediction_date":"2026-06-26T14:00:00+00:00","prediction_window_days":30,"is_actionable":true,"is_anomaly":false,"calibration_bin":8,"top_features":[{"feature":"offline_ratio","importance":0.45,"value":0.32}],"features_snapshot":{"offline_ratio":0.32},"trace_id":"abc123"}
{"device_id":"CAM-002","failure_probability":0.32,"confidence_score":0.65,"model_version":"xgboost_v1","model_variant":"A","prediction_date":"2026-06-26T14:00:00+00:00","prediction_window_days":30,"is_actionable":false,"is_anomaly":false,"calibration_bin":3,"top_features":[],"features_snapshot":{"offline_ratio":0.05},"trace_id":"abc123"}
{"_meta":{"total":2,"actionable":1,"avg_probability":0.595,"status":"ok","timestamp":"2026-06-26T14:00:00+00:00"}}`

	results, meta, err := svc.parseOutput(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("parseOutput error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Проверяем первое предсказание
	if results[0].DeviceID != "CAM-001" {
		t.Errorf("expected CAM-001, got %s", results[0].DeviceID)
	}
	if results[0].FailureProbability != 0.87 {
		t.Errorf("expected 0.87, got %f", results[0].FailureProbability)
	}
	if results[0].ConfidenceScore != 0.92 {
		t.Errorf("expected 0.92, got %f", results[0].ConfidenceScore)
	}
	if !results[0].IsActionable {
		t.Errorf("expected actionable=true")
	}
	if results[0].ModelVariant != "A" {
		t.Errorf("expected variant A, got %s", results[0].ModelVariant)
	}
	if len(results[0].TopFeatures) != 1 {
		t.Errorf("expected 1 top feature, got %d", len(results[0].TopFeatures))
	}

	// Проверяем второе предсказание
	if results[1].DeviceID != "CAM-002" {
		t.Errorf("expected CAM-002, got %s", results[1].DeviceID)
	}
	if results[1].IsActionable {
		t.Errorf("expected actionable=false")
	}

	// Проверяем мета-информацию
	if meta.Meta.Total != 2 {
		t.Errorf("expected meta total=2, got %d", meta.Meta.Total)
	}
	if meta.Meta.Actionable != 1 {
		t.Errorf("expected meta actionable=1, got %d", meta.Meta.Actionable)
	}
	if meta.Meta.Status != "ok" {
		t.Errorf("expected meta status=ok, got %s", meta.Meta.Status)
	}
}

func TestParseOutput_InvalidJSONLine(t *testing.T) {
	svc := newTestService()
	input := `not valid json
{"device_id":"CAM-001","failure_probability":0.87,"confidence_score":0.92,"model_version":"xgboost_v1","model_variant":"A","prediction_date":"2026-06-26T14:00:00+00:00","prediction_window_days":30,"is_actionable":true,"is_anomaly":false,"calibration_bin":8,"top_features":[],"features_snapshot":{},"trace_id":"abc123"}
{"_meta":{"total":1,"actionable":1,"avg_probability":0.87,"status":"ok","timestamp":"2026-06-26T14:00:00+00:00"}}`

	results, meta, err := svc.parseOutput(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("parseOutput error: %v", err)
	}

	// Должен пропустить невалидную строку, но распарсить валидную
	if len(results) != 1 {
		t.Fatalf("expected 1 result (skipping invalid), got %d", len(results))
	}

	if results[0].DeviceID != "CAM-001" {
		t.Errorf("expected CAM-001, got %s", results[0].DeviceID)
	}

	if meta.Meta.Total != 1 {
		t.Errorf("expected meta total=1, got %d", meta.Meta.Total)
	}
}

func TestParseOutput_EmptyInput(t *testing.T) {
	svc := newTestService()
	input := `{"_meta":{"total":0,"actionable":0,"avg_probability":0,"status":"ok","timestamp":"2026-06-26T14:00:00+00:00"}}`

	results, meta, err := svc.parseOutput(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("parseOutput error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	if meta.Meta.Status != "ok" {
		t.Errorf("expected status=ok, got %s", meta.Meta.Status)
	}
}

func TestParseOutput_MissingDeviceID(t *testing.T) {
	svc := newTestService()
	input := `{"failure_probability":0.87,"confidence_score":0.92,"model_version":"xgboost_v1","model_variant":"A","prediction_date":"2026-06-26T14:00:00+00:00","prediction_window_days":30,"is_actionable":true,"is_anomaly":false,"calibration_bin":8,"top_features":[],"features_snapshot":{},"trace_id":"abc123"}
{"_meta":{"total":0,"actionable":0,"avg_probability":0,"status":"ok","timestamp":"2026-06-26T14:00:00+00:00"}}`

	results, _, err := svc.parseOutput(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("parseOutput error: %v", err)
	}

	// Должен пропустить строку без device_id
	if len(results) != 0 {
		t.Errorf("expected 0 results (missing device_id), got %d", len(results))
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

// ── Truncate Utility Test ───────────────────────────────────────────

func TestTruncateString(t *testing.T) {
	short := "hello"
	if truncated := truncateString(short, 10); truncated != short {
		t.Errorf("expected no truncation, got %s", truncated)
	}

	long := "this is a very long string that should be truncated"
	truncated := truncateString(long, 20)
	if len(truncated) != 23 { // 20 + "..."
		t.Errorf("expected length 23, got %d: %s", len(truncated), truncated)
	}
	if truncated != "this is a very long ..." {
		t.Errorf("unexpected truncation: %s", truncated)
	}
}

// ── Config Defaults Test ────────────────────────────────────────────

func TestDefaultMLConfig(t *testing.T) {
	cfg := DefaultMLConfig()

	if cfg.PythonPath != "python3" {
		t.Errorf("expected python3, got %s", cfg.PythonPath)
	}
	if cfg.ScriptPath != "analytics/predict.py" {
		t.Errorf("expected analytics/predict.py, got %s", cfg.ScriptPath)
	}
	if cfg.ModelVariant != "A" {
		t.Errorf("expected variant A, got %s", cfg.ModelVariant)
	}
	if !cfg.ABTestingEnabled {
		t.Errorf("expected ABTestingEnabled=true")
	}
	if cfg.ABTestingRatio != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.ABTestingRatio)
	}
	if cfg.ProbabilityThreshold != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.ProbabilityThreshold)
	}
	if cfg.MinConfidenceThreshold != 0.3 {
		t.Errorf("expected 0.3, got %f", cfg.MinConfidenceThreshold)
	}
	if cfg.NATSTopicPrefix != "ml.prediction" {
		t.Errorf("expected ml.prediction, got %s", cfg.NATSTopicPrefix)
	}
}
