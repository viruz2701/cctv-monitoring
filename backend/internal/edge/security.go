// ═══════════════════════════════════════════════════════════════════════
// Package edge — Security controls for Edge Agent (P3-NICE.3)
//
// Соответствие:
//   - IEC 62443 SL-4: Tamper detection, secure boot verification
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности
//   - ISO 27001 A.12.4: Audit logging
//   - СТБ 34.101.30: bash-256 хеширование
// ═══════════════════════════════════════════════════════════════════════

package edge

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// ═══ Security event types ═════════════════════════════════════════════

type SecurityEventType string

const (
	EventConnectionEstablished SecurityEventType = "connection_established"
	EventConnectionClosed      SecurityEventType = "connection_closed"
	EventConnectionFailure     SecurityEventType = "connection_failure"
	EventIntegrityPassed       SecurityEventType = "integrity_passed"
	EventIntegrityFailed       SecurityEventType = "integrity_failed"
	EventIntegritySkipped      SecurityEventType = "integrity_skipped"
	EventTamperDetected        SecurityEventType = "tamper_detected"
	EventCertificateExpiring   SecurityEventType = "certificate_expiring"
	EventConfigChange          SecurityEventType = "config_change"
)

type Severity string

const (
	SevInfo     Severity = "info"
	SevWarning  Severity = "warning"
	SevCritical Severity = "critical"
)

// SecurityEvent represents a security-related event for audit logging.
// Соответствует ISO 27001 A.12.4 (audit logging) и Приказ ОАЦ №66 п. 7.18.
type SecurityEvent struct {
	Type      SecurityEventType `json:"type"`
	Severity  Severity          `json:"severity"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ═══ Integrity Checker ═══════════════════════════════════════════════

// IntegrityChecker реализует контроль целостности бинарников edge agent.
// Использует bash-256 (через SHA-256) для проверки хеша исполняемого файла.
//
// Соответствие:
//   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности через bash-256
//   - IEC 62443 SL-4: Tamper detection
type IntegrityChecker struct {
	expectedHash string        // ожидаемый bash-256 хеш
	execPath     string        // путь к проверяемому файлу
	interval     time.Duration // интервал проверки
	lastCheck    time.Time     // время последней проверки
	valid        bool          // результат последней проверки
	mu           sync.RWMutex
	logger       *slog.Logger
}

// NewIntegrityChecker creates a new integrity checker.
func NewIntegrityChecker(expectedHash string, interval time.Duration, logger *slog.Logger) *IntegrityChecker {
	execPath, _ := os.Executable()

	return &IntegrityChecker{
		expectedHash: expectedHash,
		execPath:     execPath,
		interval:     interval,
		logger:       logger.With("component", "integrity-checker"),
	}
}

// Start begins periodic integrity checks in a goroutine.
func (ic *IntegrityChecker) Start(stopCh <-chan struct{}) {
	ic.logger.Info("starting integrity checker",
		"interval", ic.interval,
		"exec_path", ic.execPath,
	)

	ticker := time.NewTicker(ic.interval)
	defer ticker.Stop()

	// Run initial check
	ic.check()

	for {
		select {
		case <-stopCh:
			ic.logger.Info("integrity checker stopped")
			return
		case <-ticker.C:
			ic.check()
		}
	}
}

// check performs a single integrity verification.
func (ic *IntegrityChecker) check() {
	start := time.Now()

	hash, err := ic.computeHash()
	if err != nil {
		ic.mu.Lock()
		ic.valid = false
		ic.lastCheck = time.Now()
		ic.mu.Unlock()

		ic.logger.Error("integrity check failed to compute hash",
			"error", err,
			"duration", time.Since(start),
		)
		return
	}

	valid := hmac.Equal([]byte(hash), []byte(ic.expectedHash))

	ic.mu.Lock()
	ic.valid = valid
	ic.lastCheck = time.Now()
	ic.mu.Unlock()

	if valid {
		ic.logger.Info("integrity check passed",
			"hash", hash[:16]+"...",
			"duration", time.Since(start),
		)
	} else {
		ic.logger.Error("INTEGRITY CHECK FAILED — TAMPER DETECTED",
			"expected", ic.expectedHash[:16]+"...",
			"got", hash[:16]+"...",
			"duration", time.Since(start),
		)
	}
}

// computeHash вычисляет bash-256 хеш исполняемого файла.
// Использует SHA-256 как fallback до имплементации СТБ bash.
func (ic *IntegrityChecker) computeHash() (string, error) {
	data, err := os.ReadFile(ic.execPath)
	if err != nil {
		return "", fmt.Errorf("failed to read executable: %w", err)
	}

	// TODO: Заменить на СТБ bash-256 когда будет доступен
	// Сейчас используем SHA-256 как временное решение
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// LastCheck returns the timestamp of the last integrity check.
func (ic *IntegrityChecker) LastCheck() time.Time {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.lastCheck
}

// IsValid returns whether the last integrity check passed.
func (ic *IntegrityChecker) IsValid() bool {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.valid
}

// ═══ Certificate monitoring ═══════════════════════════════════════════

// CertExpiryMonitor проверяет срок действия mTLS сертификатов
// и генерирует предупреждения за 30 дней до истечения.
//
// Соответствие:
//   - Приказ ОАЦ №66 п. 7.18: Управление сертификатами
//   - ISO 27001 A.12.4: Логирование событий
type CertExpiryMonitor struct {
	certPath string
	keyPath  string
	warnDays int
	logger   *slog.Logger
}

// NewCertExpiryMonitor creates a new certificate expiry monitor.
func NewCertExpiryMonitor(certPath, keyPath string, warnDays int, logger *slog.Logger) *CertExpiryMonitor {
	if warnDays <= 0 {
		warnDays = 30
	}
	return &CertExpiryMonitor{
		certPath: certPath,
		keyPath:  keyPath,
		warnDays: warnDays,
		logger:   logger.With("component", "cert-monitor"),
	}
}

// CheckExpiry проверяет дату истечения сертификата.
// Возвращает количество дней до истечения и ошибку, если сертификат истёк.
func (m *CertExpiryMonitor) CheckExpiry() (daysLeft int, err error) {
	certData, err := os.ReadFile(m.certPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read cert: %w", err)
	}

	cert, err := parseCertificate(certData)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cert: %w", err)
	}

	now := time.Now()
	daysLeft = int(cert.NotAfter.Sub(now).Hours() / 24)

	if daysLeft <= 0 {
		m.logger.Error("CERTIFICATE EXPIRED",
			"cert", m.certPath,
			"expired", cert.NotAfter.Format(time.RFC3339),
			"days_ago", -daysLeft,
		)
		return 0, fmt.Errorf("certificate expired %d days ago", -daysLeft)
	}

	if daysLeft <= m.warnDays {
		m.logger.Warn("certificate expiring soon",
			"cert", m.certPath,
			"days_left", daysLeft,
			"expires", cert.NotAfter.Format(time.RFC3339),
		)
	} else {
		m.logger.Info("certificate valid",
			"cert", m.certPath,
			"days_left", daysLeft,
			"expires", cert.NotAfter.Format(time.RFC3339),
		)
	}

	return daysLeft, nil
}

// parseCertificate parses a PEM-encoded X.509 certificate.
func parseCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// ═══ Secure channel helpers ═══════════════════════════════════════════

// GenerateSessionID creates a unique session identifier for audit trail.
func GenerateSessionID(deviceID string) string {
	now := time.Now().UnixNano()
	data := fmt.Sprintf("%s:%d", deviceID, now)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// CreateHMAC creates an HMAC-SHA256 signature for audit trail integrity.
// TODO: Заменить на СТБ bash-hmac когда будет доступен.
func CreateHMAC(key, data []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHMAC verifies an HMAC-SHA256 signature.
func VerifyHMAC(key, data []byte, expectedMAC string) bool {
	computed := CreateHMAC(key, data)
	return hmac.Equal([]byte(computed), []byte(expectedMAC))
}
