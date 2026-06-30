package descriptor

import (
	"encoding/json"
	"testing"
)

func TestProtocolDescriptor_Validate(t *testing.T) {
	t.Run("valid descriptor", func(t *testing.T) {
		d := &ProtocolDescriptor{
			Vendor:  "Hikvision",
			Version: "1.0.0",
			Protocols: map[string]Protocol{
				"isapi": {
					Transport: "http",
					BaseURL:   "http://{{.IP}}:{{.Port}}",
					Endpoints: map[string]Endpoint{
						"get_info": {
							Method: "GET",
							Path:   "/ISAPI/System/deviceInfo",
						},
					},
				},
			},
		}
		if err := d.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing vendor", func(t *testing.T) {
		d := &ProtocolDescriptor{
			Version: "1.0.0",
		}
		if err := d.Validate(); err == nil {
			t.Error("expected error for missing vendor")
		}
	})

	t.Run("missing version", func(t *testing.T) {
		d := &ProtocolDescriptor{Vendor: "Test"}
		if err := d.Validate(); err == nil {
			t.Error("expected error for missing version")
		}
	})

	t.Run("no protocols", func(t *testing.T) {
		d := &ProtocolDescriptor{Vendor: "Test", Version: "1.0.0"}
		if err := d.Validate(); err == nil {
			t.Error("expected error for no protocols")
		}
	})

	t.Run("missing endpoint method", func(t *testing.T) {
		d := &ProtocolDescriptor{
			Vendor:  "Test",
			Version: "1.0.0",
			Protocols: map[string]Protocol{
				"test": {
					Transport: "http",
					Endpoints: map[string]Endpoint{
						"ep": {Path: "/test"},
					},
				},
			},
		}
		if err := d.Validate(); err == nil {
			t.Error("expected error for missing method")
		}
	})
}

func TestProtocolDescriptor_JSON(t *testing.T) {
	jsonStr := `{
		"vendor": "Hikvision",
		"version": "1.0.0",
		"protocols": {
			"isapi": {
				"transport": "http",
				"base_url": "http://{{.IP}}:80",
				"endpoints": {
					"get_device_info": {
						"method": "GET",
						"path": "/ISAPI/System/deviceInfo"
					}
				}
			}
		}
	}`

	var d ProtocolDescriptor
	if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if d.Vendor != "Hikvision" {
		t.Errorf("expected vendor Hikvision, got %s", d.Vendor)
	}
	if d.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", d.Version)
	}
	if len(d.Protocols) != 1 {
		t.Errorf("expected 1 protocol, got %d", len(d.Protocols))
	}

	// Round-trip
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var d2 ProtocolDescriptor
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if d2.Vendor != d.Vendor {
		t.Errorf("round-trip vendor mismatch")
	}
}

func TestProtocolDescriptor_Clone(t *testing.T) {
	original := &ProtocolDescriptor{
		Vendor:  "Hikvision",
		Version: "1.0.0",
		Protocols: map[string]Protocol{
			"isapi": {
				Transport: "http",
				Endpoints: map[string]Endpoint{
					"get_info": {Method: "GET", Path: "/info"},
				},
			},
		},
	}

	clone := original.Clone()
	if clone.Vendor != original.Vendor {
		t.Error("clone vendor mismatch")
	}

	// Изменяем клон — оригинал не должен измениться
	clone.Vendor = "Modified"
	if original.Vendor != "Hikvision" {
		t.Error("clone should be a deep copy")
	}
}

func TestParseResponse(t *testing.T) {
	t.Run("json parsing", func(t *testing.T) {
		body := []byte(`{"device": {"model": "DS-2CD2T47G2", "serial": "ABC123"}}`)
		parser := &ResponseParser{
			Format: "json",
			Mappings: map[string]string{
				"model":  "device.model",
				"serial": "device.serial",
			},
		}
		result, err := parseResponse(body, parser)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if result["model"] != "DS-2CD2T47G2" {
			t.Errorf("expected model DS-2CD2T47G2, got %v", result["model"])
		}
		if result["serial"] != "ABC123" {
			t.Errorf("expected serial ABC123, got %v", result["serial"])
		}
	})

	t.Run("key-value parsing", func(t *testing.T) {
		body := []byte("deviceType=IPC-HFW1230\nserialNo=XYZ789\nfirmwareVersion=2.800.0000000.0")
		parser := &ResponseParser{
			Format:    "key_value",
			Separator: "=",
			Mappings: map[string]string{
				"model":    "deviceType",
				"serial":   "serialNo",
				"firmware": "firmwareVersion",
			},
		}
		result, err := parseResponse(body, parser)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if result["model"] != "IPC-HFW1230" {
			t.Errorf("expected model IPC-HFW1230, got %v", result["model"])
		}
	})
}

func TestRenderTemplate(t *testing.T) {
	t.Run("no template", func(t *testing.T) {
		result, err := renderTemplate("/ISAPI/System/deviceInfo", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "/ISAPI/System/deviceInfo" {
			t.Errorf("expected unchanged string, got %s", result)
		}
	})

	t.Run("with template", func(t *testing.T) {
		params := map[string]interface{}{
			"IP":   "192.168.1.100",
			"Port": "80",
		}
		result, err := renderTemplate("http://{{.IP}}:{{.Port}}/ISAPI/System/deviceInfo", params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "http://192.168.1.100:80/ISAPI/System/deviceInfo"
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})
}
