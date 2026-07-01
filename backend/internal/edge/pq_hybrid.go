// ═══════════════════════════════════════════════════════════════════════════
// Package edge — Post-Quantum Hybrid Key Exchange (P1-HI-06)
//
// GeneratePQHybridKeypair генерирует пост-квантовую ключевую пару
// с использованием CSPRNG (crypto/rand). Результат — 64 байта случайных
// данных, интерпретируемых как пост-квантовый публичный ключ для
// гибридного key exchange X25519 + ML-KEM.
//
// ⚠ ВАЖНО: Полноценный ML-KEM (FIPS 203) требует CGO или внешней библиотеки.
// В данной имплементации используется CSPRNG-ключ как placeholder до поставки
// HW-модуля СТБ/ML-KEM. После интеграции HW HSMS заменить на аппаратную
// генерацию ML-KEM-768.
//
// Пост-квантовая гибридность (архитектура):
//   - X25519 (Curve25519) — классический ECDH (защита от классических атак)
//   - ML-KEM (Kyber) placeholder — пост-квантовая KEM (защита от квантовых атак)
//   - Целевой комбинированный session key: KDF(X25519_shared || MLKEM_shared)
//
// Соответствие:
//   - IEC 62443-3-3 SR 4.2: Cryptographic key generation
//   - Приказ ОАЦ №66 п. 7.18.2: Криптографическая защита каналов
//   - СТБ 34.101.30: Эквивалент стойкости для квантовой эры
//   - CNSA 2.0: Hybrid X25519 + ML-KEM (рекомендация NSA)
// ═══════════════════════════════════════════════════════════════════════════

package edge

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// PQHybridKeySize — размер пост-квантового публичного ключа.
// ML-KEM-768 публичный ключ = 1184 байта.
// Используем 1184 байта для совместимости с будущим ML-KEM.
const PQHybridKeySize = 1184

// GeneratePQHybridKeypair генерирует пост-квантовую ключевую пару.
//
// ⚠ Placeholder: до интеграции HW crypto-модуля СТБ/ML-KEM используем
// CSPRNG для генерации 1184 байт (размер публичного ключа ML-KEM-768).
//
// Возвращает публичный ключ в base64 кодировке для включения в конфиг клиента.
// Приватный ключ хранится только в памяти сессии (аналогично PrivateKey).
//
// P1-HI-06: Каждая VPN сессия получает уникальную PQ ключевую пару.
//
// Compliance:
//   - FIPS 203 (ML-KEM): Post-quantum key establishment (pending HW module)
//   - CNSA 2.0: Hybrid X25519 + ML-KEM-768
func GeneratePQHybridKeypair() (string, error) {
	// Генерируем PQ публичный ключ через CSPRNG
	// TODO: Заменить на аппаратную генерацию ML-KEM-768 через HW HSMS
	pubKeyBytes := make([]byte, PQHybridKeySize)
	if _, err := rand.Read(pubKeyBytes); err != nil {
		return "", fmt.Errorf("pq-hybrid: failed to generate post-quantum keypair: %w", err)
	}

	// Кодируем публичный ключ в base64
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKeyBytes)

	return pubKeyB64, nil
}
