package api

import (
	"testing"
	"time"
)

func TestDateValidation_ValidRange(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"valid 2020", "2020-01-01", false},
		{"valid 2025", "2025-06-23", false},
		{"valid 2035", "2035-12-31", false},
		{"valid RFC3339 2025", "2025-06-23T10:00:00Z", false},
		{"valid RFC3339 2035", "2035-12-31T23:59:59Z", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseValidatedDate(tt.date, "test_date")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValidatedDate(%q) error = %v, wantErr = %v", tt.date, err, tt.wantErr)
			}
		})
	}
}

func TestDateValidation_OutOfRange(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"before 2020", "2019-12-31", true},
		{"after 2035", "2036-01-01", true},
		{"far past", "1999-01-01", true},
		{"far future", "2050-01-01", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseValidatedDate(tt.date, "test_date")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseValidatedDate(%q) error = %v, wantErr = %v", tt.date, err, tt.wantErr)
			}
		})
	}
}

func TestDateValidation_ErrorMessages(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		field   string
		contains string
	}{
		{"empty string", "", "next_due", "required"},
		{"too long", string(make([]byte, 65)), "test_date", "too long"},
		{"out of range", "2019-12-31", "next_due", "out of range"},
		{"invalid format", "not-a-date", "test_date", "format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseValidatedDate(tt.date, tt.field)
			if err == nil {
				t.Errorf("expected error for %q", tt.name)
				return
			}
			if !contains(err.Error(), tt.contains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.contains)
			}
		})
	}
}

func TestDateValidation_YearBoundary(t *testing.T) {
	// Тест граничных значений года
	_, err2020 := parseValidatedDate("2020-01-01", "test_date")
	if err2020 != nil {
		t.Errorf("2020-01-01 should be valid: %v", err2020)
	}

	_, err2035 := parseValidatedDate("2035-12-31", "test_date")
	if err2035 != nil {
		t.Errorf("2035-12-31 should be valid: %v", err2035)
	}

	_, err2019 := parseValidatedDate("2019-12-31", "test_date")
	if err2019 == nil {
		t.Error("2019-12-31 should be invalid (year < 2020)")
	}

	_, err2036 := parseValidatedDate("2036-01-01", "test_date")
	if err2036 == nil {
		t.Error("2036-01-01 should be invalid (year > 2035)")
	}
}

func TestIsFutureDate(t *testing.T) {
	now := time.Now().UTC()

	if !isFutureDate(now.Add(24 * time.Hour)) {
		t.Error("tomorrow should be in the future")
	}

	if isFutureDate(now.Add(-24 * time.Hour)) {
		t.Error("yesterday should not be in the future")
	}
}

func TestParseFutureDate(t *testing.T) {
	tomorrow := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	parsed, err := parseFutureDate(tomorrow, "test_date")
	if err != nil {
		t.Errorf("tomorrow should be valid: %v", err)
	}
	if parsed.IsZero() {
		t.Error("parsed date should not be zero")
	}
}

func TestNormalizeDateUpdate(t *testing.T) {
	updates := map[string]interface{}{
		"next_due": "2025-06-23",
	}

	err := normalizeDateUpdate(updates, "next_due", false)
	if err != nil {
		t.Errorf("valid date should pass: %v", err)
	}

	parsed, ok := updates["next_due"].(time.Time)
	if !ok {
		t.Error("next_due should be time.Time after normalize")
	}
	if parsed.Year() != 2025 {
		t.Errorf("expected year 2025, got %d", parsed.Year())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
