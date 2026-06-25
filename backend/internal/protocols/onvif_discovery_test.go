// CCTV-2.2.1: ONVIF Discovery Tests
//
// Тесты для WS-Discovery: парсинг ProbeMatch XML, фильтрация устройств,
// NAT connector health check mock.
//
// Compliance:
//   - OWASP ASVS L3: тестирование валидации входных данных
//   - IEC 62443-3-3 SL-3: тесты безопасности
//   - Table-driven тесты (golangci-lint style)

package protocols

import (
	"gb-telemetry-collector/internal/config"
	"log/slog"
	"os"
	"testing"
)

// ─── ProbeMatch XML Parsing ─────────────────────────────────────────────────

const sampleProbeMatchXML = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope
	xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
	xmlns:wsa="http://schemas.xmlsoap.org/ws/2004/08/addressing"
	xmlns:wsd="urn:schemas-xmlsoap-org:ws:2005:04:discovery"
	xmlns:wsdp="http://schemas.xmlsoap.org/ws/2006/02/devprof">
	<soap:Header>
		<wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/ProbeMatches</wsa:Action>
		<wsa:MessageID>urn:uuid:test-message-id</wsa:MessageID>
		<wsa:RelatesTo>urn:uuid:test-relates-to</wsa:RelatesTo>
		<wsa:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</wsa:To>
	</soap:Header>
	<soap:Body>
		<wsd:ProbeMatches>
			<wsd:ProbeMatch>
				<wsa:EndpointReference>
					<wsa:Address>urn:uuid:device-uuid-1234</wsa:Address>
				</wsa:EndpointReference>
				<wsd:Types>wsdp:Device wsdp:NetworkVideoTransmitter</wsd:Types>
				<wsd:Scopes>onvif://www.onvif.org/type/video_encoder onvif://www.onvif.org/hardware/DS-2CD2347G1-LU onvif://www.onvif.org/Profile/Streaming</wsd:Scopes>
				<wsd:XAddrs>http://192.168.1.100/onvif/device_service</wsd:XAddrs>
				<wsd:MetadataVersion>1.0</wsd:MetadataVersion>
			</wsd:ProbeMatch>
		</wsd:ProbeMatches>
	</soap:Body>
</soap:Envelope>`

const sampleProbeMatchXMLNoDevice = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope
	xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
	xmlns:wsd="urn:schemas-xmlsoap-org:ws:2005:04:discovery">
	<soap:Body>
		<wsd:ProbeMatches>
		</wsd:ProbeMatches>
	</soap:Body>
</soap:Envelope>`

const sampleProbeMatchXMLEmptyXAddr = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope
	xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
	xmlns:wsd="urn:schemas-xmlsoap-org:ws:2005:04:discovery">
	<soap:Body>
		<wsd:ProbeMatches>
			<wsd:ProbeMatch>
				<wsd:Types>wsdp:Device</wsd:Types>
				<wsd:Scopes></wsd:Scopes>
				<wsd:XAddrs></wsd:XAddrs>
			</wsd:ProbeMatch>
		</wsd:ProbeMatches>
	</soap:Body>
</soap:Envelope>`

const sampleProbeMatchXMLInvalid = `this is not xml`

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestParseProbeMatch(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantErr    bool
		wantXAddr  string
		wantTypes  int
		wantScopes int
	}{
		{
			name:       "valid probe match with camera",
			data:       []byte(sampleProbeMatchXML),
			wantErr:    false,
			wantXAddr:  "http://192.168.1.100/onvif/device_service",
			wantTypes:  2,
			wantScopes: 3,
		},
		{
			name:       "empty probe matches",
			data:       []byte(sampleProbeMatchXMLNoDevice),
			wantErr:    false,
			wantXAddr:  "",
			wantTypes:  0,
			wantScopes: 0,
		},
		{
			name:       "empty xaddr returns error",
			data:       []byte(sampleProbeMatchXMLEmptyXAddr),
			wantErr:    true,
			wantXAddr:  "",
			wantTypes:  0,
			wantScopes: 0,
		},
		{
			name:       "invalid XML returns error",
			data:       []byte(sampleProbeMatchXMLInvalid),
			wantErr:    true,
			wantXAddr:  "",
			wantTypes:  0,
			wantScopes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, err := parseProbeMatch(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseProbeMatch() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseProbeMatch() unexpected error: %v", err)
				return
			}

			if device == nil && tt.wantXAddr != "" {
				t.Errorf("parseProbeMatch() returned nil device, expected XAddr=%s", tt.wantXAddr)
				return
			}

			if device == nil {
				return
			}

			if device.XAddr != tt.wantXAddr {
				t.Errorf("parseProbeMatch() XAddr = %q, want %q", device.XAddr, tt.wantXAddr)
			}

			if len(device.Types) != tt.wantTypes {
				t.Errorf("parseProbeMatch() Types count = %d, want %d", len(device.Types), tt.wantTypes)
			}

			if len(device.Scopes) != tt.wantScopes {
				t.Errorf("parseProbeMatch() Scopes count = %d, want %d", len(device.Scopes), tt.wantScopes)
			}
		})
	}
}

// ─── Filter Tests ───────────────────────────────────────────────────────────

func TestFilterDevicesByType(t *testing.T) {
	devices := []DiscoveredDevice{
		{
			XAddr: "http://192.168.1.100/onvif/device_service",
			Types: []string{"wsdp:Device", "wsdp:NetworkVideoTransmitter"},
		},
		{
			XAddr: "http://192.168.1.101/onvif/device_service",
			Types: []string{"wsdp:Device", "wsdp:NetworkVideoDisplay"},
		},
		{
			XAddr: "http://192.168.1.102/onvif/device_service",
			Types: []string{"wsdp:Device"},
		},
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{
			name:     "filter by NetworkVideoTransmitter",
			filter:   []string{"NetworkVideoTransmitter"},
			expected: 1,
		},
		{
			name:     "filter by NetworkVideoDisplay",
			filter:   []string{"NetworkVideoDisplay"},
			expected: 1,
		},
		{
			name:     "filter by multiple types",
			filter:   []string{"NetworkVideoTransmitter", "NetworkVideoDisplay"},
			expected: 2,
		},
		{
			name:     "filter by non-existent type",
			filter:   []string{"NetworkVideoEncoder"},
			expected: 0,
		},
		{
			name:     "empty filter returns all",
			filter:   []string{},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterDevicesByType(devices, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("FilterDevicesByType() count = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

// ─── Scopes Parsing ─────────────────────────────────────────────────────────

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "multiple scopes",
			input:    "onvif://www.onvif.org/type/video_encoder onvif://www.onvif.org/hardware/DS-2CD2347G1-LU",
			expected: 2,
		},
		{
			name:     "single scope",
			input:    "onvif://www.onvif.org/type/video_encoder",
			expected: 1,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "leading/trailing spaces",
			input:    "  scope1  scope2  ",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseScopes(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseScopes() count = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

// ─── UUID Generator ─────────────────────────────────────────────────────────

func TestNewUUID(t *testing.T) {
	uuid1 := newUUID()
	uuid2 := newUUID()

	if uuid1 == "" {
		t.Error("newUUID() returned empty string")
	}

	if uuid1 == uuid2 {
		t.Error("newUUID() returned duplicate UUIDs")
	}

	// Проверяем формат UUID v4
	if len(uuid1) != 36 {
		t.Errorf("newUUID() length = %d, want 36", len(uuid1))
	}
}

// ─── NAT Health Check Mock ──────────────────────────────────────────────────

// TestONVIFNATManagerHealthCheck тестирует health check NAT менеджера
// с пустым p2pURL (direct mode — всегда healthy).
func TestONVIFNATManagerHealthCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mgr := NewONVIFNATManager(
		config.ONVIFConfig{
			Enabled:     true,
			ConnectMode: "direct",
		},
		"", // p2pURL пустой — direct mode
		"",
		logger,
	)

	// В direct mode HealthCheck должен возвращать ошибку если устройство не зарегистрировано
	err := mgr.HealthCheck(nil, "test_device")
	if err == nil {
		t.Error("Expected error for unregistered device, got nil")
	}
}
