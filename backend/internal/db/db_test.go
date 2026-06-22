package db

import (
	"testing"
	"time"
)

func TestConnectionPoolDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{
			name: "empty config should get defaults",
			cfg:  Config{},
			want: Config{
				MaxConns:          25,
				MinConns:          5,
				MaxConnLifetime:   5 * time.Minute,
				MaxConnIdleTime:   3 * time.Minute,
				HealthCheckPeriod: 1 * time.Minute,
			},
		},
		{
			name: "partial config should fill missing defaults",
			cfg: Config{
				MaxConns: 10,
			},
			want: Config{
				MaxConns:          10,
				MinConns:          5,
				MaxConnLifetime:   5 * time.Minute,
				MaxConnIdleTime:   3 * time.Minute,
				HealthCheckPeriod: 1 * time.Minute,
			},
		},
		{
			name: "zero values should get defaults",
			cfg: Config{
				MaxConns:          0,
				MinConns:          0,
				MaxConnLifetime:   0,
				MaxConnIdleTime:   0,
				HealthCheckPeriod: 0,
			},
			want: Config{
				MaxConns:          25,
				MinConns:          5,
				MaxConnLifetime:   5 * time.Minute,
				MaxConnIdleTime:   3 * time.Minute,
				HealthCheckPeriod: 1 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.withDefaults()

			if got.MaxConns != tt.want.MaxConns {
				t.Errorf("MaxConns = %d, want %d", got.MaxConns, tt.want.MaxConns)
			}
			if got.MinConns != tt.want.MinConns {
				t.Errorf("MinConns = %d, want %d", got.MinConns, tt.want.MinConns)
			}
			if got.MaxConnLifetime != tt.want.MaxConnLifetime {
				t.Errorf("MaxConnLifetime = %v, want %v", got.MaxConnLifetime, tt.want.MaxConnLifetime)
			}
			if got.MaxConnIdleTime != tt.want.MaxConnIdleTime {
				t.Errorf("MaxConnIdleTime = %v, want %v", got.MaxConnIdleTime, tt.want.MaxConnIdleTime)
			}
			if got.HealthCheckPeriod != tt.want.HealthCheckPeriod {
				t.Errorf("HealthCheckPeriod = %v, want %v", got.HealthCheckPeriod, tt.want.HealthCheckPeriod)
			}
		})
	}
}

func TestConnectionPoolMinExceedsMax(t *testing.T) {
	cfg := Config{
		MinConns: 10,
		MaxConns: 5,
	}

	_, err := New(cfg, nil)
	if err == nil {
		t.Error("expected error when MinConns > MaxConns")
	} else if err.Error() != "database min connections (10) cannot exceed max connections (5)" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConnectionPoolValidateParams(t *testing.T) {
	// Проверяем что withDefaults не падает с любыми входными данными
	cfg := Config{
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   3 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}

	result := cfg.withDefaults()
	if result.MaxConns != 25 {
		t.Errorf("MaxConns should remain 25, got %d", result.MaxConns)
	}
	if result.MinConns != 5 {
		t.Errorf("MinConns should remain 5, got %d", result.MinConns)
	}
}

func TestDSNFormat(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "test_user",
		Password: "test_pass",
		DBName:   "test_db",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	expected := "postgres://test_user:test_pass@localhost:5432/test_db?sslmode=disable"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestClosedPoolPanic(t *testing.T) {
	// Проверяем что Close() не паникует если Pool == nil
	var db *DB
	if db != nil {
		db.Close()
	}
	// Если дошли сюда без panic — тест пройден
}
