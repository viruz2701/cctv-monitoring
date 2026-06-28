// Package auth — аутентификация и управление доступом.
//
// Содержит общие вспомогательные функции для работы с JWT_SECRET и BIGN_PRIVATE_KEY.
// Соответствует: ISO 27001 A.9.4 (Authentication), OWASP ASVS V2 (Authentication)
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-SEC.2: bign JWT — ECDSA P-256 (bign-curve256v1)
//
// Переход с HMAC-SHA256 на ECDSA P-256 для подписи JWT:
//   - JWT_SECRET — legacy symmetric secret (только для обратной совместимости)
//   - BIGN_PRIVATE_KEY — PEM-encoded ECDSA P-256 приватный ключ
//
// Если BIGN_PRIVATE_KEY не задан, система генерирует новый ключ при старте
// (для development режима). Для production требуется явно указать ключ.
//
// Compliance:
//   - СТБ 34.101.45 — bign-curve256v1
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - OWASP ASVS V6.2.2 — Асимметричная криптография
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// ────────────────────────────────────────────────────────────────────────────
// JWT_SECRET (legacy symmetric secret)
// ────────────────────────────────────────────────────────────────────────────

// ErrJWTSecretMissing возвращается когда JWT_SECRET не установлен.
// Используется для graceful degradation — сервер продолжает работу,
// но /health возвращает 503.
var ErrJWTSecretMissing = errors.New("JWT_SECRET environment variable is required")

// GetJWTSecret возвращает JWT_SECRET из переменных окружения.
// Возвращает error если секрет не задан — никогда не паникует.
//
// ⚠ Legacy: Используется только для обратной совместимости (refresh tokens).
// Для JWT подписи используйте GetBignPrivateKey().
//
// Compliance:
//   - ISO 27001 A.9.4.2 (Secure authentication — key management)
//   - OWASP ASVS V2.1 (Secret verification)
//   - Приказ ОАЦ №66 п. 7.18.1 (Unique identification — key material)
func GetJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrJWTSecretMissing
	}
	return []byte(secret), nil
}

// IsJWTSecretSet проверяет установлен ли JWT_SECRET.
// Используется для health check — если не установлен, /health возвращает 503.
func IsJWTSecretSet() bool {
	return os.Getenv("JWT_SECRET") != ""
}

// IsBignKeySet проверяет доступен ли bign ключ для подписи JWT.
// В development режиме ключ генерируется автоматически, так что
// функция всегда возвращает true (если нет системной ошибки).
//
// Для production: true только если BIGN_PRIVATE_KEY или BIGN_PRIVATE_KEY_FILE задан.
func IsBignKeySet() bool {
	if os.Getenv("BIGN_PRIVATE_KEY") != "" || os.Getenv("BIGN_PRIVATE_KEY_FILE") != "" {
		return true
	}
	// В dev режиме ключ генерируется автоматически
	return true
}

// ────────────────────────────────────────────────────────────────────────────
// BIGN_PRIVATE_KEY (ECDSA P-256 / bign-curve256v1)
// ────────────────────────────────────────────────────────────────────────────

// ErrBignKeyMissing возвращается когда BIGN_PRIVATE_KEY не установлен.
var ErrBignKeyMissing = errors.New("BIGN_PRIVATE_KEY environment variable is required")

// bignPrivateKey — кэшированный ECDSA приватный ключ.
var (
	bignPrivateKeyOnce sync.Once
	bignPrivateKey     *ecdsa.PrivateKey
	bignPrivateKeyErr  error
)

// GetBignPrivateKey возвращает ECDSA P-256 приватный ключ для подписи JWT.
//
// Источники ключа (в порядке приоритета):
//  1. BIGN_PRIVATE_KEY — PEM-encoded ECDSA P-256 приватный ключ (production)
//  2. BIGN_PRIVATE_KEY_FILE — путь к PEM-файлу с ключом
//  3. Автогенерация (development mode)
//
// В production режиме ключ ДОЛЖЕН быть явно указан через BIGN_PRIVATE_KEY.
// Ключ кэшируется после первой загрузки.
func GetBignPrivateKey() (*ecdsa.PrivateKey, error) {
	bignPrivateKeyOnce.Do(func() {
		bignPrivateKey, bignPrivateKeyErr = loadBignPrivateKey()
	})
	return bignPrivateKey, bignPrivateKeyErr
}

// ResetBignPrivateKey сбрасывает кэш ключа (для тестов).
func ResetBignPrivateKey() {
	bignPrivateKeyOnce = sync.Once{}
	bignPrivateKey = nil
	bignPrivateKeyErr = nil
}

// loadBignPrivateKey загружает ECDSA ключ из env или генерирует новый.
func loadBignPrivateKey() (*ecdsa.PrivateKey, error) {
	// 1. Прямой PEM из env
	pemData := os.Getenv("BIGN_PRIVATE_KEY")
	if pemData != "" {
		return parseBignPrivateKeyPEM([]byte(pemData))
	}

	// 2. Файл с ключом
	keyFile := os.Getenv("BIGN_PRIVATE_KEY_FILE")
	if keyFile != "" {
		pemBytes, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("read bign key file: %w", err)
		}
		return parseBignPrivateKeyPEM(pemBytes)
	}

	// 3. Автогенерация для development
	log.Println("[WARN] BIGN_PRIVATE_KEY not set, generating ephemeral ECDSA P-256 key for development")
	log.Println("[WARN] Set BIGN_PRIVATE_KEY in production for persistent key")

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral bign key: %w", err)
	}

	// Логируем публичный ключ для отладки
	pubDER, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	log.Printf("[INFO] Ephemeral bign public key:\n%s", pubPEM)

	return privKey, nil
}

// parseBignPrivateKeyPEM парсит PEM-encoded ECDSA P-256 приватный ключ.
func parseBignPrivateKeyPEM(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("bign key: no PEM block found")
	}

	// Пробуем EC PRIVATE KEY
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err == nil {
		if key.Curve != elliptic.P256() {
			return nil, fmt.Errorf("bign key: expected P-256 curve, got %s", key.Curve.Params().Name)
		}
		return key, nil
	}

	// Пробуем PKCS8
	pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, fmt.Errorf("bign key: parse failed (EC: %v, PKCS8: %v)", err, err2)
	}

	ecKey, ok := pkcs8Key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("bign key: not an ECDSA key (got %T)", pkcs8Key)
	}
	if ecKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("bign key: expected P-256 curve, got %s", ecKey.Curve.Params().Name)
	}

	return ecKey, nil
}

// GetBignPublicKey возвращает публичный ключ из приватного.
func GetBignPublicKey() (*ecdsa.PublicKey, error) {
	privKey, err := GetBignPrivateKey()
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}
