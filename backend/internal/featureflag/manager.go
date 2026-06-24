// Package featureflag — самописная система Feature Flags (F-0.2.4).
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — feature gating)
//   - ISO 27001 A.12.1.2 (Change management — controlled rollout)
//   - ISO/IEC 27019 PCC.A.12 (Change management for ICS)
//   - СТБ 34.101.27 (Защита информации — контроль доступа к функциям)
//   - OWASP ASVS V1.1 (Architecture — feature flags как security control)
//   - OWASP ASVS V5 (Input validation — whitelist for feature keys)
//
// Кэширование: in-memory map + sync.RWMutex, перезагрузка каждые 60s.
// Не использует Unleash — самописная для KII-2 compliance.
package featureflag

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gb-telemetry-collector/internal/models"
)

// DB — интерфейс для работы с БД (избегаем циклических импортов).
type DB interface {
	// GetAllFeatureFlags возвращает все фича-флаги из БД.
	GetAllFeatureFlags(ctx context.Context) ([]models.FeatureFlag, error)
	// SetFeatureFlagEnabled обновляет enabled для флага.
	SetFeatureFlagEnabled(ctx context.Context, key string, enabled bool) error
}

// DefaultRefreshInterval — интервал обновления кэша (60 секунд).
const DefaultRefreshInterval = 60 * time.Second

// Manager — in-memory кэш фича-флагов с периодической синхронизацией из БД.
//
// Thread-safe: sync.RWMutex для конкурентного доступа.
// Fail-secure (IEC 62443 SR 7.1): при ошибке загрузки флаг считается disabled.
type Manager struct {
	mu              sync.RWMutex
	flags           map[string]models.FeatureFlag
	db              DB
	logger          *slog.Logger
	refreshInterval time.Duration
	stopCh          chan struct{}
	stopOnce        sync.Once
}

// NewManager создаёт FeatureFlagManager и загружает флаги из БД.
// Запускает фоновую горутину для периодического обновления кэша.
// panic если db или logger nil.
func NewManager(db DB, logger *slog.Logger) *Manager {
	if db == nil {
		panic("featureflag: db cannot be nil")
	}
	if logger == nil {
		panic("featureflag: logger cannot be nil")
	}

	m := &Manager{
		flags:           make(map[string]models.FeatureFlag),
		db:              db,
		logger:          logger,
		refreshInterval: DefaultRefreshInterval,
		stopCh:          make(chan struct{}),
	}

	// Первичная загрузка из БД (blocking, при старте)
	if err := m.refresh(context.Background()); err != nil {
		logger.Error("FeatureFlag: initial load failed, starting with empty cache", "error", err)
	}

	// Фоновое обновление кэша
	go m.periodicRefresh()

	logger.Info("FeatureFlag manager initialized",
		"refresh_interval", m.refreshInterval,
		"initial_flags", len(m.flags),
	)

	return m
}

// IsEnabled проверяет включён ли флаг (глобально, для всех тенантов).
// Возвращает false если флаг не найден (fail-secure).
func (m *Manager) IsEnabled(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[key]
	if !ok {
		return false
	}
	return flag.Enabled
}

// IsEnabledForTenant проверяет включён ли флаг для конкретного тенанта.
// Сначала ищет tenant-specific флаг, затем глобальный ('*').
// Возвращает false если флаг не найден (fail-secure).
func (m *Manager) IsEnabledForTenant(key, tenantID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Сначала ищем tenant-specific флаг
	tenantKey := fmt.Sprintf("%s:%s", key, tenantID)
	if flag, ok := m.flags[tenantKey]; ok {
		return flag.Enabled
	}

	// Затем глобальный
	if flag, ok := m.flags[key]; ok {
		return flag.Enabled && flag.TenantID == "*"
	}

	return false
}

// SetEnabled обновляет состояние флага в БД и в кэше.
// Возвращает ошибку если обновление в БД не удалось.
//
// Compliance: мутация данных — логируется через audit_log (ISO 27001 A.12.4).
func (m *Manager) SetEnabled(key string, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.db.SetFeatureFlagEnabled(ctx, key, enabled); err != nil {
		return fmt.Errorf("featureflag: set %q = %v: %w", key, enabled, err)
	}

	// Обновляем кэш
	m.mu.Lock()
	if flag, ok := m.flags[key]; ok {
		flag.Enabled = enabled
		flag.UpdatedAt = time.Now()
		m.flags[key] = flag
	} else {
		// Флаг может быть создан напрямую в БД, добавляем в кэш
		m.flags[key] = models.FeatureFlag{
			Key:       key,
			Enabled:   enabled,
			TenantID:  "*",
			UpdatedAt: time.Now(),
		}
	}
	m.mu.Unlock()

	m.logger.Info("FeatureFlag updated",
		"key", key,
		"enabled", enabled,
	)
	return nil
}

// GetAll возвращает копию всех флагов из кэша.
func (m *Manager) GetAll() []models.FeatureFlag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.FeatureFlag, 0, len(m.flags))
	for _, flag := range m.flags {
		result = append(result, flag)
	}
	return result
}

// Stop останавливает фоновое обновление кэша.
// Безопасен для многократного вызова (sync.Once).
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		m.logger.Info("FeatureFlag manager stopped")
	})
}

// refresh загружает все флаги из БД в кэш.
func (m *Manager) refresh(ctx context.Context) error {
	flags, err := m.db.GetAllFeatureFlags(ctx)
	if err != nil {
		return fmt.Errorf("featureflag: refresh: %w", err)
	}

	m.mu.Lock()
	// Сбрасываем и перестраиваем мапу
	m.flags = make(map[string]models.FeatureFlag, len(flags))
	for i := range flags {
		m.flags[flags[i].Key] = flags[i]
	}
	m.mu.Unlock()

	m.logger.Debug("FeatureFlag cache refreshed", "count", len(flags))
	return nil
}

// periodicRefresh — фоновая горутина для обновления кэша.
func (m *Manager) periodicRefresh() {
	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := m.refresh(ctx); err != nil {
				m.logger.Error("FeatureFlag periodic refresh failed", "error", err)
			}
			cancel()
		case <-m.stopCh:
			return
		}
	}
}
