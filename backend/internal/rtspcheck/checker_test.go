package rtspcheck

import (
	"testing"
	"time"
)

func TestParseRTSPURL(t *testing.T) {
	tests := []struct {
		raw          string
		expectedHost string
		expectedPort int
	}{
		{"rtsp://192.168.1.100:554/stream1", "192.168.1.100", 554},
		{"rtsp://admin:pass@192.168.1.100:554/stream1", "192.168.1.100", 554},
		{"rtsp://192.168.1.100/stream1", "192.168.1.100", 554},
		{"rtsp://camera.local:8554/test", "camera.local", 8554},
		{"rtsps://10.0.0.1:554/", "10.0.0.1", 554},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			host, port, err := parseRTSPURL(tt.raw)
			if err != nil {
				t.Fatalf("parseRTSPURL failed: %v", err)
			}
			if host != tt.expectedHost {
				t.Errorf("expected host %s, got %s", tt.expectedHost, host)
			}
			if port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestParseRTSPURL_Invalid(t *testing.T) {
	_, _, err := parseRTSPURL("")
	if err == nil {
		t.Error("expected error for empty URL")
	}

	_, _, err = parseRTSPURL("rtsp:///path")
	if err == nil {
		t.Error("expected error for URL with empty host")
	}
}

func TestCountStreams(t *testing.T) {
	sdp := `v=0
o=- 123456 789 IN IP4 192.168.1.100
s=RTSP Stream
m=video 0 RTP/AVP 96
a=rtpmap:96 H264/90000
m=audio 0 RTP/AVP 0
a=rtpmap:0 PCMU/8000
m=application 0 RTP/AVP 107
a=rtpmap:107 vnd.onvif.metadata/90000`

	count := countStreams(sdp)
	if count != 3 {
		t.Errorf("expected 3 streams (video+audio+metadata), got %d", count)
	}
}

func TestCountStreams_Empty(t *testing.T) {
	if countStreams("") != 0 {
		t.Error("expected 0 streams for empty SDP")
	}
}

func TestHealthScore(t *testing.T) {
	tests := []struct {
		name     string
		result   *CheckResult
		minScore float64
		maxScore float64
	}{
		{"online", &CheckResult{Status: StatusOnline, ResponseTime: 50 * time.Millisecond, StreamHealth: StreamHealthy}, 90, 100},
		{"offline", &CheckResult{Status: StatusOffline}, 0, 0},
		{"timeout", &CheckResult{Status: StatusTimeout}, 40, 60},
		{"degraded", &CheckResult{Status: StatusDegraded}, 60, 80},
		{"frozen", &CheckResult{Status: StatusOnline, ResponseTime: 50 * time.Millisecond, StreamHealth: StreamFrozen}, 50, 70},
		{"slow", &CheckResult{Status: StatusOnline, ResponseTime: 1500 * time.Millisecond, StreamHealth: StreamHealthy}, 70, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.result.HealthScore()
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("expected score %.0f-%.0f, got %.0f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	r := &CheckResult{
		URL:          "rtsp://192.168.1.100:554/stream1",
		Status:       StatusOnline,
		StatusCode:   200,
		ResponseTime: 50 * time.Millisecond,
		Streams:      2,
		Server:       "RtspServer/1.0",
		StreamHealth: StreamHealthy,
	}

	summary := r.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	t.Logf("Summary: %s", summary)

	// Offline summary
	r2 := &CheckResult{
		Status: StatusOffline,
		Error:  "connection refused",
	}
	summary2 := r2.Summary()
	if summary2 == "" {
		t.Error("expected non-empty offline summary")
	}
	t.Logf("Offline summary: %s", summary2)
}

func TestNewChecker(t *testing.T) {
	checker := NewChecker(nil)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}
	if checker.timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", checker.timeout)
	}

	checker.SetTimeout(10 * time.Second)
	if checker.timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", checker.timeout)
	}
}

func TestDetectFrozenStream(t *testing.T) {
	checker := NewChecker(nil)

	// First check — healthy
	health := checker.detectStreamHealth("rtsp://cam/stream1", "SDP body v1")
	if health != StreamHealthy {
		t.Error("expected healthy on first check")
	}

	// Same body — frozen
	health = checker.detectStreamHealth("rtsp://cam/stream1", "SDP body v1")
	if health != StreamFrozen {
		t.Error("expected frozen on duplicate body")
	}

	// Different body — healthy again (stream changed)
	health = checker.detectStreamHealth("rtsp://cam/stream1", "SDP body v2")
	if health != StreamHealthy {
		t.Error("expected healthy after body change")
	}
}

func TestDetectFrozenStream_DifferentURLs(t *testing.T) {
	checker := NewChecker(nil)

	// Same body for different URLs — should not be frozen
	checker.detectStreamHealth("rtsp://cam1/stream", "SDP body")
	health := checker.detectStreamHealth("rtsp://cam2/stream", "SDP body")
	if health != StreamHealthy {
		t.Error("same body on different URLs should not be frozen")
	}
}

func TestEmptyBodyHealth(t *testing.T) {
	checker := NewChecker(nil)
	health := checker.detectStreamHealth("rtsp://cam/stream", "")
	if health != StreamUnknown {
		t.Error("expected unknown for empty body")
	}
}
