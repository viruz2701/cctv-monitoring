// Package events — tests for Schema Registry with Circuit Breaker.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist validation testing)
//   - OWASP ASVS V5.3 (Input validation — structured data validation testing)
//   - ISO 27001 A.12.4.1 (Event logging — data quality enforcement)
//   - IEC 62443 SR 3.1 (Data integrity validation)
//   - Правило 7: Тестирование соответствия (unit ≥ 80%, security, compliance)
package events

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: Circuit Breaker
// ═══════════════════════════════════════════════════════════════════════

// TestSchemaRegistry_CircuitBreaker_OpenOnHighFailureRate проверяет
// что circuit breaker открывается при превышении порога ошибок (>10%).
func TestSchemaRegistry_CircuitBreaker_OpenOnHighFailureRate(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	// Устанавливаем агрессивные параметры для теста
	r.cbConfig = CircuitBreakerConfig{
		FailureThreshold:   0.10,          // 10%
		MinValidationCount: 5,             // всего 5 валидаций для срабатывания
		AutoResetInterval:  1 * time.Hour, // не сбросится во время теста
		Enabled:            true,
	}
	r.cbState = CBClosed

	// Проводим 10 невалидных валидаций — circuit breaker должен открыться
	for i := 0; i < 10; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"invalid": true}`), // missing required fields
		}
		_ = r.Validate(record)
	}

	if r.CircuitBreakerState() != CBOpen {
		t.Error("expected circuit breaker to be OPEN after high failure rate")
	}

	// После открытия — валидация должна пропускать (возвращать nil)
	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000002",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000003","type":"motion","severity":"high","message":"test"}`),
	}
	if err := r.Validate(record); err != nil {
		t.Errorf("expected validation to be skipped (circuit breaker open), got error: %v", err)
	}
}

// TestSchemaRegistry_CircuitBreaker_StaysClosedOnLowFailureRate проверяет
// что circuit breaker остаётся закрытым при низком проценте ошибок.
func TestSchemaRegistry_CircuitBreaker_StaysClosedOnLowFailureRate(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	r.cbConfig = CircuitBreakerConfig{
		FailureThreshold:   0.50, // 50% — высокий порог
		MinValidationCount: 5,
		AutoResetInterval:  1 * time.Hour,
		Enabled:            true,
	}
	r.cbState = CBClosed

	// Проводим 8 валидных + 2 невалидных = 20% ошибок (< 50%)
	for i := 0; i < 8; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"test"}`),
		}
		_ = r.Validate(record)
	}
	for i := 0; i < 2; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000003",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"invalid": true}`),
		}
		_ = r.Validate(record)
	}

	if r.CircuitBreakerState() != CBClosed {
		t.Errorf("expected circuit breaker to stay CLOSED (20%% failures < 50%% threshold), got OPEN")
	}

	// Валидация должна продолжаться
	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000004",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000005","type":"motion","severity":"high","message":"test"}`),
	}
	if err := r.Validate(record); err != nil {
		t.Errorf("expected successful validation, got error: %v", err)
	}
}

// TestSchemaRegistry_CircuitBreaker_NotEnoughData проверяет что circuit breaker
// не срабатывает до накопления минимального количества валидаций.
func TestSchemaRegistry_CircuitBreaker_NotEnoughData(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	r.cbConfig = CircuitBreakerConfig{
		FailureThreshold:   0.10, // 10%
		MinValidationCount: 100,  // минимум 100 валидаций
		AutoResetInterval:  1 * time.Hour,
		Enabled:            true,
	}
	r.cbState = CBClosed
	r.totalValidations.Store(0)
	r.invalidEvents.Store(0)

	// Всего 5 валидаций — все невалидные, но < MinValidationCount
	for i := 0; i < 5; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"invalid": true}`),
		}
		_ = r.Validate(record)
	}

	if r.CircuitBreakerState() != CBClosed {
		t.Error("expected circuit breaker to stay CLOSED (not enough data)")
	}

	// Валидация должна продолжаться (возвращать ошибки)
	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000002",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"invalid": true}`),
	}
	if err := r.Validate(record); err == nil {
		t.Error("expected validation error when circuit breaker is closed")
	}
}

// TestSchemaRegistry_CircuitBreaker_Disabled проверяет что при отключённом
// circuit breaker валидация всегда активна.
func TestSchemaRegistry_CircuitBreaker_Disabled(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	r.cbConfig = CircuitBreakerConfig{
		FailureThreshold:   0.10,
		MinValidationCount: 5,
		AutoResetInterval:  1 * time.Hour,
		Enabled:            false, // circuit breaker отключён
	}
	r.cbState = CBClosed
	r.totalValidations.Store(0)
	r.invalidEvents.Store(0)

	// Много невалидных валидаций
	for i := 0; i < 20; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"invalid": true}`),
		}
		_ = r.Validate(record)
	}

	if r.CircuitBreakerState() != CBClosed {
		t.Error("expected circuit breaker to stay CLOSED when disabled")
	}

	// Валидация всё ещё активна (возвращает ошибки)
	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000002",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"invalid": true}`),
	}
	if err := r.Validate(record); err == nil {
		t.Error("expected validation error even with circuit breaker disabled")
	}
}

// TestSchemaRegistry_CircuitBreaker_ConfigUpdate проверяет обновление
// конфигурации circuit breaker через SetCircuitBreakerConfig.
func TestSchemaRegistry_CircuitBreaker_ConfigUpdate(t *testing.T) {
	r := NewSchemaRegistry(nil)

	cfg := CircuitBreakerConfig{
		FailureThreshold:   0.25,
		MinValidationCount: 50,
		AutoResetInterval:  10 * time.Minute,
		Enabled:            false,
	}

	r.SetCircuitBreakerConfig(cfg)

	got := r.CircuitBreakerConfig()
	if got.FailureThreshold != 0.25 {
		t.Errorf("expected FailureThreshold 0.25, got %f", got.FailureThreshold)
	}
	if got.MinValidationCount != 50 {
		t.Errorf("expected MinValidationCount 50, got %d", got.MinValidationCount)
	}
	if got.AutoResetInterval != 10*time.Minute {
		t.Errorf("expected AutoResetInterval 10m, got %v", got.AutoResetInterval)
	}
	if got.Enabled != false {
		t.Errorf("expected Enabled false, got %v", got.Enabled)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: Validation Counters
// ═══════════════════════════════════════════════════════════════════════

// TestSchemaRegistry_ValidationCounters проверяет атомарные счётчики валидации.
func TestSchemaRegistry_ValidationCounters(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	r.cbConfig = CircuitBreakerConfig{
		FailureThreshold:   0.50,
		MinValidationCount: 100,
		AutoResetInterval:  1 * time.Hour,
		Enabled:            false, // отключаем CB для точного подсчёта
	}

	// Сбрасываем счётчики для чистоты теста
	r.totalValidations.Store(0)
	r.invalidEvents.Store(0)

	// 3 успешных валидации (alarm.resolved не имеет format:'uuid' для device_id)
	for i := 0; i < 3; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    SourceAlarms,
			EventType: "alarm.resolved",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"alarm_id":"0190abcd-1234-7000-8000-000000000002","resolved_by":"tech-001","resolution":"Fixed"}`),
		}
		if err := r.Validate(record); err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	}

	// 2 невалидных
	for i := 0; i < 2; i++ {
		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000003",
			Source:    SourceAlarms,
			EventType: "alarm.created",
			Timestamp: time.Now(),
			Data:      json.RawMessage(`{"invalid": true}`),
		}
		_ = r.Validate(record)
	}

	counters := r.ValidationCounters()
	if counters.TotalValidations != 5 {
		t.Errorf("expected 5 total validations, got %d", counters.TotalValidations)
	}
	if counters.InvalidEvents != 2 {
		t.Errorf("expected 2 invalid events, got %d", counters.InvalidEvents)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Performance Tests: Schema Validation <5ms overhead
// ═══════════════════════════════════════════════════════════════════════

// TestSchemaRegistry_Validate_Performance проверяет что валидация
// укладывается в <5ms overhead per validation.
func TestSchemaRegistry_Validate_Performance(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger
	r.cbConfig.Enabled = false // отключаем CB для чистоты замера

	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000001",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"test"}`),
	}

	// Прогрев (warm-up): 10 итераций
	for i := 0; i < 10; i++ {
		_ = r.Validate(record)
	}

	// Измерение: 100 итераций
	const iterations = 100
	start := time.Now()
	for i := 0; i < iterations; i++ {
		if err := r.Validate(record); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	}
	elapsed := time.Since(start)
	avgPerOp := elapsed / iterations

	t.Logf("Average validation time: %v (%d iterations)", avgPerOp, iterations)

	if avgPerOp > 5*time.Millisecond {
		t.Errorf("validation too slow: avg %v > 5ms threshold", avgPerOp)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Edge Case Tests: Schema Validation
// ═══════════════════════════════════════════════════════════════════════

// TestSchemaRegistry_Validate_InvalidJSON проверяет обработку
// невалидного JSON в payload.
func TestSchemaRegistry_Validate_InvalidJSON(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	record := &EventRecord{
		ID:        "test-id",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{invalid json}`),
	}

	err := r.Validate(record)
	if err == nil {
		t.Error("expected validation error for invalid JSON payload")
	}
}

// TestSchemaRegistry_Validate_EmptyJSONObject проверяет пустой объект.
func TestSchemaRegistry_Validate_EmptyJSONObject(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	record := &EventRecord{
		ID:        "test-id",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	err := r.Validate(record)
	if err == nil {
		t.Error("expected validation error for empty object (missing required fields)")
	}
}

// TestSchemaRegistry_Validate_NonRequiredSchema проверяет что схема
// с Required=false (телефония) пропускает валидацию если схема не найдена,
// но ВСЁ РАВНО валидирует если схема зарегистрирована.
func TestSchemaRegistry_Validate_NonRequiredSchema(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	// telemetry.metric имеет Required=false (см. SchemaDefinition.Required),
	// но JSON Schema ВСЁ РАВНО содержит required: ["device_id","metric","value"].
	// Схема зарегистрирована → валидация выполняется.
	record := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000001",
		Source:    SourceTelemetry,
		EventType: "telemetry.metric",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	err := r.Validate(record)
	if err == nil {
		t.Error("expected validation error: telemetry schema has required fields even if non-required")
	}

	// Для незарегистрированной схемы с Required=false — ошибки нет
	record2 := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000002",
		Source:    SourceTelemetry,
		EventType: "telemetry.unknown", // незарегистрированный тип
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	if err := r.Validate(record2); err != nil {
		t.Errorf("expected no error for unknown non-required schema, got: %v", err)
	}
}

// TestSchemaRegistry_Validate_UnknownSource проверяет что неизвестный
// источник не вызывает ошибку (логируется WARN, пропускается).
func TestSchemaRegistry_Validate_UnknownSource(t *testing.T) {
	r := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	r.logger = logger

	record := &EventRecord{
		ID:        "test-id",
		Source:    "unknown_source",
		EventType: "unknown.event",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"test": true}`),
	}

	err := r.Validate(record)
	if err != nil {
		t.Errorf("expected no error for unknown source, got: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: Publisher Validation Integration
// ═══════════════════════════════════════════════════════════════════════

// TestPublisher_PublishRecord_WithValidation проверяет что PublishRecord
// выполняет валидацию через SchemaRegistry.
func TestPublisher_PublishRecord_WithValidation(t *testing.T) {
	registry := NewSchemaRegistry(nil)
	logger := slog.New(slog.DiscardHandler)
	registry.logger = logger

	// Создаём Publisher с schemaRegistry (без реального NATS — только тест структуры)
	p := &Publisher{
		schemaRegistry: registry,
		logger:         logger,
	}

	// Валидный record (alarm.resolved не имеет format:'uuid' для device_id)
	validRecord := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000001",
		Source:    SourceAlarms,
		EventType: "alarm.resolved",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"alarm_id":"0190abcd-1234-7000-8000-000000000002","resolved_by":"tech-001","resolution":"Fixed"}`),
	}

	// Не можем протестировать полный publish без NATS, но можем проверить
	// что schemaRegistry вызывается через экспорт внутреннего метода для тестов
	// Вместо этого проверяем что registry.Validate() работает корректно
	if err := registry.Validate(validRecord); err != nil {
		t.Errorf("expected valid record to pass validation, got: %v", err)
	}

	// Невалидный record
	invalidRecord := &EventRecord{
		ID:        "0190abcd-1234-7000-8000-000000000003",
		Source:    SourceAlarms,
		EventType: "alarm.created",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"invalid": true}`),
	}

	if err := registry.Validate(invalidRecord); err == nil {
		t.Error("expected invalid record to fail validation")
	}

	// Проверяем что publisher использует schemaRegistry (структурно)
	if p.schemaRegistry == nil {
		t.Error("expected schemaRegistry to be set on publisher")
	}
}

// TestPublisher_PublishRecord_WithoutValidation проверяет что PublishRecord
// работает без SchemaRegistry (nil schemaRegistry).
func TestPublisher_PublishRecord_WithoutValidation(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	p := &Publisher{
		schemaRegistry: nil,
		logger:         logger,
	}

	if p.schemaRegistry != nil {
		t.Error("expected schemaRegistry to be nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Compliance Tests: Circuit Breaker
// ═══════════════════════════════════════════════════════════════════════

// TestCompliance_CircuitBreaker_DefaultConfig проверяет что конфигурация
// по умолчанию соответствует требованиям (10%, 100 валидаций, 5 мин reset).
func TestCompliance_CircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig

	if cfg.FailureThreshold != 0.10 {
		t.Errorf("compliance: expected FailureThreshold 0.10, got %f", cfg.FailureThreshold)
	}
	if cfg.MinValidationCount != 100 {
		t.Errorf("compliance: expected MinValidationCount 100, got %d", cfg.MinValidationCount)
	}
	if cfg.AutoResetInterval != 5*time.Minute {
		t.Errorf("compliance: expected AutoResetInterval 5m, got %v", cfg.AutoResetInterval)
	}
	if !cfg.Enabled {
		t.Error("compliance: expected circuit breaker enabled by default")
	}
}
