// Package analytics — tests for query builder.
package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNew_ValidTemplates(t *testing.T) {
	templates := []QueryTemplate{
		{ID: "test1", Name: "Test 1", SQL: "SELECT 1", Dimensions: []Field{{Key: "x", Label: "X", Type: "string"}}},
		{ID: "test2", Name: "Test 2", SQL: "SELECT 2", Measures: []Field{{Key: "y", Label: "Y", Type: "number", AggFunction: "SUM"}}},
	}

	qb, err := New(nil, templates)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	got := qb.GetTemplates()
	if len(got) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(got))
	}
}

func TestNew_DuplicateID(t *testing.T) {
	templates := []QueryTemplate{
		{ID: "dup", Name: "A", SQL: "SELECT 1"},
		{ID: "dup", Name: "B", SQL: "SELECT 2"},
	}

	_, err := New(nil, templates)
	if err == nil {
		t.Fatal("expected error for duplicate template ID")
	}
}

func TestNew_EmptySQL(t *testing.T) {
	templates := []QueryTemplate{
		{ID: "empty", Name: "Empty", SQL: ""},
	}

	_, err := New(nil, templates)
	if err == nil {
		t.Fatal("expected error for empty SQL")
	}
}

func TestGetTemplate(t *testing.T) {
	templates := []QueryTemplate{
		{ID: "findme", Name: "Find Me", SQL: "SELECT 1"},
	}

	qb, err := New(nil, templates)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl, ok := qb.GetTemplate("findme")
	if !ok {
		t.Fatal("expected to find template 'findme'")
	}
	if tmpl.Name != "Find Me" {
		t.Fatalf("expected name 'Find Me', got %q", tmpl.Name)
	}

	_, ok = qb.GetTemplate("nonexistent")
	if ok {
		t.Fatal("expected not to find nonexistent template")
	}
}

func TestValidateFields(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl := simpleTemplate()

	tests := []struct {
		name    string
		params  QueryParams
		wantErr bool
	}{
		{
			name:    "valid dimensions",
			params:  QueryParams{Dimensions: []string{"device_id", "vendor_type"}, Measures: []string{"device_count"}},
			wantErr: false,
		},
		{
			name:    "invalid dimension",
			params:  QueryParams{Dimensions: []string{"nonexistent"}},
			wantErr: true,
		},
		{
			name:    "invalid measure",
			params:  QueryParams{Measures: []string{"invalid_measure"}},
			wantErr: true,
		},
		{
			name:    "empty dimensions and measures",
			params:  QueryParams{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := qb.validateFields(tmpl, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFields() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl := simpleTemplate()

	tests := []struct {
		name        string
		params      QueryParams
		wantContain string
		wantErr     bool
	}{
		{
			name:        "select dimensions only",
			params:      QueryParams{Dimensions: []string{"device_id"}},
			wantContain: `"device_id"`,
		},
		{
			name:        "select measures only",
			params:      QueryParams{Measures: []string{"device_count"}},
			wantContain: `COUNT(1) AS "device_count"`,
		},
		{
			name: "dimensions + measures + filter",
			params: QueryParams{
				Dimensions: []string{"device_id", "vendor_type"},
				Measures:   []string{"device_count", "avg_cost"},
				Filters:    []FilterCondition{{Field: "vendor_type", Operator: "eq", Value: "Hikvision"}},
			},
			wantContain: `vendor_type = $1`,
		},
		{
			name: "with limit and offset",
			params: QueryParams{
				Dimensions: []string{"device_id"},
				Limit:      10,
				Offset:     5,
			},
			wantContain: `LIMIT 10`,
		},
		{
			name: "with order by",
			params: QueryParams{
				Dimensions: []string{"device_id"},
				OrderBy:    "device_count",
				OrderDir:   "desc",
			},
			wantContain: `ORDER BY "device_count" DESC`,
		},
		{
			name: "with time range",
			params: QueryParams{
				Dimensions: []string{"device_id"},
				TimeFrom:   timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				TimeTo:     timePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
			wantContain: `"d.created_at" >= $1`,
		},
		{
			name: "no dimensions or measures — error",
			params: QueryParams{
				Dimensions: []string{},
				Measures:   []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.params.TemplateID = "simple"
			sql, _, err := qb.buildQuery(tmpl, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildQuery() returned error: %v", err)
			}

			if tt.wantContain != "" && !contains(sql, tt.wantContain) {
				t.Errorf("buildQuery() SQL = %q, want contain %q", sql, tt.wantContain)
			}
		})
	}
}

func TestBuildQuery_FilterOperators(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl := simpleTemplate()

	tests := []struct {
		name    string
		filter  FilterCondition
		wantSQL string
	}{
		{name: "eq", filter: FilterCondition{Field: "vendor_type", Operator: "eq", Value: "Hik"}, wantSQL: `vendor_type = $1`},
		{name: "neq", filter: FilterCondition{Field: "vendor_type", Operator: "neq", Value: "Hik"}, wantSQL: `vendor_type != $1`},
		{name: "gt", filter: FilterCondition{Field: "avg_cost", Operator: "gt", Value: 5}, wantSQL: `total_cost > $1`},
		{name: "gte", filter: FilterCondition{Field: "avg_cost", Operator: "gte", Value: 5}, wantSQL: `total_cost >= $1`},
		{name: "lt", filter: FilterCondition{Field: "avg_cost", Operator: "lt", Value: 10}, wantSQL: `total_cost < $1`},
		{name: "lte", filter: FilterCondition{Field: "avg_cost", Operator: "lte", Value: 10}, wantSQL: `total_cost <= $1`},
		{name: "like", filter: FilterCondition{Field: "vendor_type", Operator: "like", Value: "Hik%"}, wantSQL: `vendor_type LIKE $1`},
		{name: "contains", filter: FilterCondition{Field: "vendor_type", Operator: "contains", Value: "Hik"}, wantSQL: `vendor_type ILIKE $1`},
		{name: "in", filter: FilterCondition{Field: "vendor_type", Operator: "in", Value: []interface{}{"A", "B"}}, wantSQL: `vendor_type IN ($1, $2)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := qb.buildQuery(tmpl, QueryParams{
				TemplateID: "simple",
				Dimensions: []string{"device_id"},
				Filters:    []FilterCondition{tt.filter},
			})
			if err != nil {
				t.Fatalf("buildQuery() returned error: %v", err)
			}
			if !contains(sql, tt.wantSQL) {
				t.Errorf("buildQuery() SQL = %q, want contain %q", sql, tt.wantSQL)
			}
		})
	}
}

func TestBuildQuery_InvalidFilterOperator(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl := simpleTemplate()

	_, _, err = qb.buildQuery(tmpl, QueryParams{
		TemplateID: "simple",
		Dimensions: []string{"device_id"},
		Filters:    []FilterCondition{{Field: "vendor_type", Operator: "invalid_op", Value: "x"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid operator")
	}
}

func TestBuildQuery_FilterFieldNotFound(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	tmpl := simpleTemplate()

	_, _, err = qb.buildQuery(tmpl, QueryParams{
		TemplateID: "simple",
		Dimensions: []string{"device_id"},
		Filters:    []FilterCondition{{Field: "nonexistent", Operator: "eq", Value: "x"}},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent filter field")
	}
}

func TestExecute_TemplateNotFound(t *testing.T) {
	qb, err := New(nil, []QueryTemplate{simpleTemplate()})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	_, err = qb.Execute(context.Background(), QueryParams{
		TemplateID: "nonexistent",
		Dimensions: []string{"device_id"},
	})

	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}

	var valErr *ValidationError
	if !asType(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDefaultTemplates(t *testing.T) {
	templates := DefaultTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one default template")
	}

	// All templates must be valid for New()
	_, err := New(nil, templates)
	if err != nil {
		t.Fatalf("DefaultTemplates() validation failed: %v", err)
	}

	// Check required fields
	for _, tmpl := range templates {
		if tmpl.ID == "" {
			t.Errorf("template %q has empty ID", tmpl.Name)
		}
		if tmpl.Name == "" {
			t.Errorf("template ID %q has empty Name", tmpl.ID)
		}
		if tmpl.SQL == "" {
			t.Errorf("template %q has empty SQL", tmpl.ID)
		}
		if len(tmpl.Dimensions) == 0 && len(tmpl.Measures) == 0 {
			t.Errorf("template %q has no dimensions or measures", tmpl.ID)
		}
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"device_id", `"device_id"`},
		{"device name", `"device name"`},
		{`quo"te`, `"quo""te"`},
	}

	for _, tt := range tests {
		got := quoteIdent(tt.input)
		if got != tt.expected {
			t.Errorf("quoteIdent(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func simpleTemplate() QueryTemplate {
	return QueryTemplate{
		ID:          "simple",
		Name:        "Simple Test",
		Description: "Template for unit tests",
		SQL:         "SELECT * FROM devices",
		Dimensions: []Field{
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
		},
		Measures: []Field{
			{Key: "device_count", Label: "Device Count", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
			{Key: "avg_cost", Label: "Avg Cost", Type: "number", AggFunction: "AVG", SQLExpr: "total_cost"},
		},
		DateField: "d.created_at",
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func asType[T error](err error, target *T) bool {
	//nolint:errorlint // intentional type assertion
	e, ok := err.(T)
	if ok {
		*target = e
	}
	return ok
}

// ── Compile-time check ────────────────────────────────────────────────────────

var _ = pgxpool.Pool{} // ensure import compiles
