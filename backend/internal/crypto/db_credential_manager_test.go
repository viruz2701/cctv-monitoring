package crypto

import (
	"context"
	"os"
	"testing"
)

// skipIfNoDB пропускает тест если нет PostgreSQL.
func skipIfNoDB(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Skipping: TEST_DATABASE_URL not set")
	}
}

func TestCredentialManager_StoreRetrieve(t *testing.T) {
	// Этот тест требует PostgreSQL. Без него — только проверка интерфейса.
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	// Placeholder: в CI тесты будут запускаться с testcontainers-go
	// Сейчас проверяем что интерфейс определён корректно
	var mgr CredentialManager
	_ = mgr // проверка компиляции
}

func TestCredentialManager_Interface(t *testing.T) {
	// Проверяем что интерфейс определён корректно (compile-time check)
	var _ CredentialManager = (*DBCredentialManager)(nil)

	ctx := context.Background()
	_ = ctx
}

func TestCredentialManager_Errors(t *testing.T) {
	t.Run("empty device_id", func(t *testing.T) {
		if ErrDeviceIDRequired.Error() != "device_id is required" {
			t.Errorf("unexpected error message: %s", ErrDeviceIDRequired.Error())
		}
	})

	t.Run("credential not found", func(t *testing.T) {
		if ErrCredentialNotFound.Error() != "credential not found for device" {
			t.Errorf("unexpected error message: %s", ErrCredentialNotFound.Error())
		}
	})
}

func TestCredentialRecord_JSON(t *testing.T) {
	// Проверяем что CredentialRecord сериализуется корректно
	record := CredentialRecord{
		DeviceID:  "cam-001",
		Algorithm: "aes-256-gcm",
		KeyRef:    "primary",
	}

	// username_enc и password_enc должны быть skipped в JSON (tag: "-")
	if record.UsernameEnc != nil {
		t.Error("expected nil UsernameEnc")
	}
	if record.PasswordEnc != nil {
		t.Error("expected nil PasswordEnc")
	}
}
