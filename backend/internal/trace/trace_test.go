package trace

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── Helpers ────────────────────────────────────────────────────────────

// validTraceIDLen is the expected length of a generated trace ID (32 hex chars = 16 bytes).
const validTraceIDLen = 32

func isValidHex(s string) bool {
	if len(s) != validTraceIDLen {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ── Middleware Tests ───────────────────────────────────────────────────

func TestMiddleware_GeneratesNewID(t *testing.T) {
	t.Parallel()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := FromContext(r.Context())
		if traceID == "" {
			t.Error("expected non-empty trace ID")
		}
		if !isValidHex(traceID) {
			t.Errorf("expected valid hex trace ID, got: %s", traceID)
		}
		if len(traceID) != validTraceIDLen {
			t.Errorf("expected trace ID length %d, got: %d", validTraceIDLen, len(traceID))
		}

		respID := w.Header().Get("X-Request-ID")
		if respID != traceID {
			t.Errorf("response header X-Request-ID = %s, want %s", respID, traceID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddleware_UsesXRequestID(t *testing.T) {
	t.Parallel()

	expectedID := "my-custom-request-id-12345"

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := FromContext(r.Context())
		if traceID != expectedID {
			t.Errorf("FromContext = %s, want %s", traceID, expectedID)
		}

		respID := w.Header().Get("X-Request-ID")
		if respID != expectedID {
			t.Errorf("response header X-Request-ID = %s, want %s", respID, expectedID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expectedID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddleware_UsesTraceparent(t *testing.T) {
	t.Parallel()

	// W3C traceparent format: version-traceid-parentid-flags
	traceID := "0af7651916cd43dd8448eb211c80319c"
	traceparent := "00-" + traceID + "-b7ad6b7169203331-01"

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := FromContext(r.Context())
		if got != traceID {
			t.Errorf("FromContext = %s, want %s (extracted from traceparent)", got, traceID)
		}

		respID := w.Header().Get("X-Request-ID")
		if respID != traceID {
			t.Errorf("response header X-Request-ID = %s, want %s", respID, traceID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", traceparent)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddleware_TraceparentPriority(t *testing.T) {
	t.Parallel()

	// X-Request-ID should take priority over traceparent
	expectedID := "priority-id"
	traceparent := "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := FromContext(r.Context())
		if got != expectedID {
			t.Errorf("FromContext = %s, want %s (X-Request-ID should have priority)", got, expectedID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expectedID)
	req.Header.Set("traceparent", traceparent)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestMiddleware_SetsResponseHeader(t *testing.T) {
	t.Parallel()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := FromContext(r.Context())
		if traceID == "" {
			t.Error("expected non-empty trace ID")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	respID := rec.Header().Get("X-Request-ID")
	if respID == "" {
		t.Error("response header X-Request-ID is empty")
	}
	if !isValidHex(respID) {
		t.Errorf("response header X-Request-ID is not valid hex: %s", respID)
	}
}

func TestMiddleware_InvalidTraceparent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		traceparent string
	}{
		{"empty", ""},
		{"too short", "00-abc-def"},
		{"invalid hex", "00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-b7ad6b7169203331-01"},
		{"too few parts", "00-onlyone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				traceID := FromContext(r.Context())
				if traceID == "" {
					t.Error("expected non-empty trace ID (should fall back to generated)")
				}
				if !isValidHex(traceID) {
					t.Errorf("expected valid hex trace ID, got: %s", traceID)
				}
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.traceparent != "" {
				req.Header.Set("traceparent", tt.traceparent)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		})
	}
}

// ── Context Helpers Tests ──────────────────────────────────────────────

func TestFromContextOrDefault_Existing(t *testing.T) {
	t.Parallel()

	expected := "test-trace-id"
	ctx := WithContext(context.Background(), expected)

	got := FromContextOrDefault(ctx)
	if got != expected {
		t.Errorf("FromContextOrDefault = %s, want %s", got, expected)
	}
}

func TestFromContextOrDefault_Missing(t *testing.T) {
	t.Parallel()

	got := FromContextOrDefault(context.Background())
	if got != "unknown" {
		t.Errorf("FromContextOrDefault = %s, want unknown", got)
	}
}

func TestFromContextOrDefault_EmptyString(t *testing.T) {
	t.Parallel()

	// Edge case: trace ID explicitly set to empty string
	ctx := WithContext(context.Background(), "")
	got := FromContextOrDefault(ctx)
	if got != "unknown" {
		t.Errorf("FromContextOrDefault = %s, want unknown", got)
	}
}

func TestWithNewID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create new context with fresh ID
	ctx1 := WithNewID(ctx)
	trace1 := FromContext(ctx1)
	if trace1 == "" {
		t.Error("WithNewID: expected non-empty trace ID")
	}
	if !isValidHex(trace1) {
		t.Errorf("WithNewID: expected valid hex, got: %s", trace1)
	}

	// Second call should produce a different ID
	ctx2 := WithNewID(ctx)
	trace2 := FromContext(ctx2)
	if trace2 == "" {
		t.Error("WithNewID: expected non-empty trace ID")
	}
	if trace1 == trace2 {
		t.Error("WithNewID: expected different trace IDs for consecutive calls")
	}

	// Verify original context is not modified
	if got := FromContext(ctx); got != "" {
		t.Errorf("original context should not have trace ID, got: %s", got)
	}
}

func TestWithNewID_PreservesExistingValues(t *testing.T) {
	t.Parallel()

	// Ensure WithNewID doesn't wipe existing context values
	type ctxKey string
	key := ctxKey("user")
	expectedUser := "test-user"

	ctx := context.WithValue(context.Background(), key, expectedUser)
	ctxWithTrace := WithNewID(ctx)

	if user := ctxWithTrace.Value(key).(string); user != expectedUser {
		t.Errorf("preserved value = %s, want %s", user, expectedUser)
	}

	traceID := FromContext(ctxWithTrace)
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
}

func TestFromContext_Empty(t *testing.T) {
	t.Parallel()

	if got := FromContext(context.Background()); got != "" {
		t.Errorf("FromContext on empty context = %s, want empty string", got)
	}
}

func TestFromContext_Existing(t *testing.T) {
	t.Parallel()

	expected := "test-id"
	ctx := WithContext(context.Background(), expected)

	if got := FromContext(ctx); got != expected {
		t.Errorf("FromContext = %s, want %s", got, expected)
	}
}

// ── Log Attributes Tests ───────────────────────────────────────────────

func TestLogAttrs_WithTraceID(t *testing.T) {
	t.Parallel()

	expected := "log-attrs-test-id"
	ctx := WithContext(context.Background(), expected)

	attrs := LogAttrs(ctx)
	if len(attrs) != 1 {
		t.Fatalf("LogAttrs returned %d attrs, want 1", len(attrs))
	}

	got := attrs[0].Value.String()
	if !strings.Contains(got, expected) {
		t.Errorf("LogAttrs[0] = %s, should contain %s", got, expected)
	}
}

func TestLogAttrs_WithoutTraceID(t *testing.T) {
	t.Parallel()

	attrs := LogAttrs(context.Background())
	if len(attrs) != 1 {
		t.Fatalf("LogAttrs returned %d attrs, want 1", len(attrs))
	}

	// Should have empty trace_id
	if attrs[0].Value.String() != "" {
		t.Errorf("LogAttrs[0] = %s, want empty", attrs[0].Value.String())
	}
}

func TestSlogAttr(t *testing.T) {
	t.Parallel()

	expected := "slog-attr-test"
	ctx := WithContext(context.Background(), expected)

	attr := SlogAttr(ctx)
	if attr.Key != "trace_id" {
		t.Errorf("SlogAttr key = %s, want trace_id", attr.Key)
	}
	if attr.Value.String() != expected {
		t.Errorf("SlogAttr value = %s, want %s", attr.Value.String(), expected)
	}
}

func TestLogAttrs_UsageWithSlog(t *testing.T) {
	t.Parallel()

	// Verify LogAttrs values can be passed to slog manually
	ctx := WithContext(context.Background(), "test-slog")
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	attrs := LogAttrs(ctx)
	args := make([]any, 0, len(attrs)*2)
	for _, a := range attrs {
		args = append(args, a.Key, a.Value.String())
	}
	logger.Info("test message", args...)

	output := buf.String()
	if !strings.Contains(output, "test-slog") {
		t.Errorf("log output should contain trace ID, got: %s", output)
	}
}

// ── NewID Tests ────────────────────────────────────────────────────────

func TestNewID_Length(t *testing.T) {
	t.Parallel()

	id := NewID()
	if len(id) != validTraceIDLen {
		t.Errorf("NewID() length = %d, want %d", len(id), validTraceIDLen)
	}
}

func TestNewID_Uniqueness(t *testing.T) {
	t.Parallel()

	ids := make(map[string]bool)
	const count = 100
	for i := 0; i < count; i++ {
		id := NewID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestNewID_HexFormat(t *testing.T) {
	t.Parallel()

	id := NewID()
	if !isValidHex(id) {
		t.Errorf("NewID() = %s, expected valid hex string", id)
	}
}

// ── ExtractFromHTTP Tests ─────────────────────────────────────────────

func TestExtractFromHTTP_WithXRequestID(t *testing.T) {
	t.Parallel()

	expected := "extract-http-id"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expected)

	ctx := ExtractFromHTTP(req)
	got := FromContext(ctx)
	if got != expected {
		t.Errorf("ExtractFromHTTP = %s, want %s", got, expected)
	}
}

func TestExtractFromHTTP_WithTraceparent(t *testing.T) {
	t.Parallel()

	traceID := "9b8a7c6d5e4f3a2b1c0d9e8f7a6b5c4d"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-"+traceID+"-b7ad6b7169203331-01")

	ctx := ExtractFromHTTP(req)
	got := FromContext(ctx)
	if got != traceID {
		t.Errorf("ExtractFromHTTP = %s, want %s", got, traceID)
	}
}

func TestExtractFromHTTP_GeneratedFallback(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := ExtractFromHTTP(req)
	got := FromContext(ctx)
	if got == "" {
		t.Error("ExtractFromHTTP: expected non-empty trace ID (fallback)")
	}
	if !isValidHex(got) {
		t.Errorf("ExtractFromHTTP: expected valid hex, got: %s", got)
	}
}

func TestExtractFromHTTP_PreservesOriginalContext(t *testing.T) {
	t.Parallel()

	type ctxKey string
	key := ctxKey("original")
	expected := "preserved-value"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), key, expected))

	ctx := ExtractFromHTTP(req)
	if v := ctx.Value(key).(string); v != expected {
		t.Errorf("preserved value = %s, want %s", v, expected)
	}
	traceID := FromContext(ctx)
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
}

// ── Edge Cases ─────────────────────────────────────────────────────────

func TestParseTraceparent_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard format",
			input:    "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			expected: "0af7651916cd43dd8448eb211c80319c",
		},
		{
			name:     "different flags",
			input:    "01-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-00",
			expected: "0af7651916cd43dd8448eb211c80319c",
		},
		{
			name:     "uppercase hex",
			input:    "00-0AF7651916CD43DD8448EB211C80319C-b7ad6b7169203331-01",
			expected: "0AF7651916CD43DD8448EB211C80319C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTraceparent(tt.input)
			if got != tt.expected {
				t.Errorf("parseTraceparent = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestParseTraceparent_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "00-abc-def"},
		{"invalid hex", "00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-b7ad6b7169203331-01"},
		{"trace id too short (30 chars)", "00-0af7651916cd43dd8448eb211c8031-b7ad6b7169203331-01"},
		{"trace id too long", "00-0af7651916cd43dd8448eb211c80319cff-b7ad6b7169203331-01"},
		{"no hyphen", "justonefield"},
		{"only traceparent prefix", "00-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTraceparent(tt.input)
			if got != "" {
				t.Errorf("parseTraceparent(%q) = %s, want empty", tt.input, got)
			}
		})
	}
}

// ── Benchmark ──────────────────────────────────────────────────────────

func BenchmarkNewID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewID()
	}
}

func BenchmarkFromContext(b *testing.B) {
	ctx := WithContext(context.Background(), "benchmark-trace-id")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromContext(ctx)
	}
}

func BenchmarkMiddleware(b *testing.B) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkLogAttrs(b *testing.B) {
	ctx := WithContext(context.Background(), "benchmark-trace-id")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogAttrs(ctx)
	}
}

// ── Data Race Tests ────────────────────────────────────────────────────

func TestConcurrentFromContext(t *testing.T) {
	ctx := WithContext(context.Background(), "concurrent-test")
	done := make(chan struct{})
	const goroutines = 20

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = FromContext(ctx)
				_ = FromContextOrDefault(ctx)
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

func TestConcurrentNewID(t *testing.T) {
	done := make(chan struct{})
	const goroutines = 10

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				id := NewID()
				if len(id) != validTraceIDLen {
					t.Errorf("concurrent NewID: expected length %d, got %d", validTraceIDLen, len(id))
				}
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}
