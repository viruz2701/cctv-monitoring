package dlp

import (
	"regexp"
	"testing"
)

func TestDetect_Email(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Contact user@example.com for support")
	if len(matches) == 0 {
		t.Fatal("expected to find email")
	}
	if matches[0].Type != PIIEmail {
		t.Fatalf("expected email type, got %s", matches[0].Type)
	}
	if matches[0].Level != SensitivityLow {
		t.Fatalf("expected low sensitivity, got %v", matches[0].Level)
	}
}

func TestDetect_Phone(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Call +375291234567 for help")
	if len(matches) == 0 {
		t.Fatal("expected to find phone")
	}
	if matches[0].Type != PIIPhone {
		t.Fatalf("expected phone type, got %s", matches[0].Type)
	}
}

func TestDetect_SSN(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("SSN: 123-45-6789")
	if len(matches) == 0 {
		t.Fatal("expected to find SSN")
	}
	if matches[0].Level != SensitivityHigh {
		t.Fatalf("expected high sensitivity for SSN, got %v", matches[0].Level)
	}
}

func TestDetect_Passport(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Passport MP1234567")
	if len(matches) == 0 {
		t.Fatal("expected to find passport")
	}
	if matches[0].Level != SensitivityHigh {
		t.Fatalf("expected high sensitivity for passport")
	}
}

func TestDetect_BankCard(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Card: 4111-1111-1111-1111")
	if len(matches) == 0 {
		t.Fatal("expected to find bank card")
	}
}

func TestDetect_NoPII(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("This is a normal text without any sensitive data")
	if len(matches) != 0 {
		t.Fatalf("expected no PII, got %d matches", len(matches))
	}
}

func TestDetect_MultiplePII(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Email: user@test.com, Phone: +375291234567, SSN: 123-45-6789")
	if len(matches) < 3 {
		t.Fatalf("expected at least 3 PII matches, got %d", len(matches))
	}
}

func TestDetectSensitivity_None(t *testing.T) {
	d := NewDLPDetector()
	level := d.DetectSensitivity("normal text")
	if level >= SensitivityLow {
		t.Fatalf("expected no sensitivity, got %v", level)
	}
}

func TestDetectSensitivity_Low(t *testing.T) {
	d := NewDLPDetector()
	level := d.DetectSensitivity("email: test@example.com")
	if level != SensitivityLow {
		t.Fatalf("expected low sensitivity, got %v", level)
	}
}

func TestDetectSensitivity_High(t *testing.T) {
	d := NewDLPDetector()
	level := d.DetectSensitivity("SSN: 123-45-6789")
	if level != SensitivityHigh {
		t.Fatalf("expected high sensitivity, got %v", level)
	}
}

func TestHasHighSensitivity_True(t *testing.T) {
	d := NewDLPDetector()
	if !d.HasHighSensitivity("SSN: 123-45-6789") {
		t.Fatal("expected high sensitivity")
	}
}

func TestHasHighSensitivity_False(t *testing.T) {
	d := NewDLPDetector()
	if d.HasHighSensitivity("email: test@example.com") {
		t.Fatal("expected no high sensitivity")
	}
}

func TestRedact_Email(t *testing.T) {
	d := NewDLPDetector()
	input := "Contact user@example.com"
	matches := d.Detect(input)
	redacted := Redact(input, matches)
	if redacted != "Contact u***@example.com" {
		t.Fatalf("expected 'Contact u***@example.com', got %q", redacted)
	}
}

func TestRedact_Phone(t *testing.T) {
	d := NewDLPDetector()
	input := "Call +375291234567"
	matches := d.Detect(input)
	redacted := Redact(input, matches)
	if redacted == input {
		t.Fatal("expected phone to be redacted")
	}
}

func TestRedact_SSN(t *testing.T) {
	d := NewDLPDetector()
	input := "SSN: 123-45-6789"
	matches := d.Detect(input)
	redacted := Redact(input, matches)
	if redacted != "SSN: ***REDACTED***" {
		t.Fatalf("expected SSN to be redacted, got %q", redacted)
	}
}

func TestRedact_Multiple(t *testing.T) {
	d := NewDLPDetector()
	input := "User: test@example.com, Phone: +375291234567"
	matches := d.Detect(input)
	redacted := Redact(input, matches)
	if redacted == input {
		t.Fatal("expected PII to be redacted")
	}
}

func TestAddPattern(t *testing.T) {
	d := NewDLPDetector()
	d.AddPattern(PIIPattern{
		Type: "custom", Level: SensitivityHigh,
		Regex: regexp.MustCompile(`CUSTOM-\d{4}`),
	})
	matches := d.Detect("Code: CUSTOM-1234")
	if len(matches) == 0 {
		t.Fatal("expected custom pattern to match")
	}
}

func TestDetect_NoDuplicates(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("email: test@example.com and also test@example.com again")
	emailCount := 0
	for _, m := range matches {
		if m.Type == PIIEmail {
			emailCount++
		}
	}
	if emailCount > 1 {
		t.Fatalf("expected no duplicate email matches, got %d", emailCount)
	}
}

func TestDetect_INN(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("УНП: 123456789")
	if len(matches) == 0 {
		t.Fatal("expected to find INN/UNP")
	}
}

func TestDetect_Address(t *testing.T) {
	d := NewDLPDetector()
	matches := d.Detect("Address: 123 Main str.")
	if len(matches) == 0 {
		t.Fatal("expected to find address")
	}
}
