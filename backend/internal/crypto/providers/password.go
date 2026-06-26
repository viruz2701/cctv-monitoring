// Package providers — Password Hashing Providers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.3: Password Hashing Providers
//
// Провайдеры:
//   - BCryptHash — bcrypt (международный fallback)
//   - Argon2IDHash — Argon2id (EU/INTL, рекомендованный OWASP)
//   - BeltHash — belt-hash (BY, СТБ 34.101.31) — stub
//
// Runtime selection per profile через PasswordHashFromProfile().
//
// Compliance:
//   - OWASP ASVS V2 (Authentication — password storage)
//   - NIST SP 800-63B (Digital identity guidelines)
//   - СТБ 34.101.31 (belt-hash)
//   - ISO 27001 A.9.4 (Access control)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// ────────────────────────────────────────────────────────────────────────────
// PasswordHashProvider — интерфейс для хеширования паролей
// ────────────────────────────────────────────────────────────────────────────

// PasswordHashProvider defines the interface for password hashing.
type PasswordHashProvider interface {
	// Name возвращает название провайдера.
	Name() string
	// Hash хеширует пароль.
	Hash(password string) (string, error)
	// Verify проверяет пароль против хеша.
	Verify(password, hash string) (bool, error)
	// Cost возвращает стоимость/итерации.
	Cost() int
}

// ────────────────────────────────────────────────────────────────────────────
// BCryptHash — bcrypt password hashing (fallback)
// ────────────────────────────────────────────────────────────────────────────

// BCryptHash implements PasswordHashProvider using bcrypt.
type BCryptHash struct {
	cost int
}

// NewBCryptHash создаёт bcrypt провайдер.
func NewBCryptHash() *BCryptHash {
	return &BCryptHash{cost: bcrypt.DefaultCost}
}

// NewBCryptHashWithCost создаёт bcrypt провайдер с указанным cost.
func NewBCryptHashWithCost(cost int) *BCryptHash {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BCryptHash{cost: cost}
}

func (b *BCryptHash) Name() string { return "bcrypt" }

func (b *BCryptHash) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

func (b *BCryptHash) Verify(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, fmt.Errorf("bcrypt verify: %w", err)
	}
	return true, nil
}

func (b *BCryptHash) Cost() int { return b.cost }

// ────────────────────────────────────────────────────────────────────────────
// Argon2IDHash — Argon2id password hashing (рекомендованный OWASP)
// ────────────────────────────────────────────────────────────────────────────

// Argon2IDParams — параметры Argon2id.
type Argon2IDParams struct {
	Time    uint32 // Итерации
	Memory  uint32 // Память в KB
	Threads uint8  // Потоки
	KeyLen  uint32 // Длина ключа
	SaltLen uint32 // Длина соли
}

// DefaultArgon2IDParams — параметры Argon2id по умолчанию (OWASP recommended).
var DefaultArgon2IDParams = Argon2IDParams{
	Time:    3,
	Memory:  64 * 1024, // 64 MB
	Threads: 4,
	KeyLen:  32,
	SaltLen: 16,
}

// Argon2IDHash implements PasswordHashProvider using Argon2id.
type Argon2IDHash struct {
	params Argon2IDParams
}

// NewArgon2IDHash создаёт Argon2id провайдер с параметрами по умолчанию.
func NewArgon2IDHash() *Argon2IDHash {
	return &Argon2IDHash{params: DefaultArgon2IDParams}
}

// NewArgon2IDHashWithParams создаёт Argon2id провайдер с указанными параметрами.
func NewArgon2IDHashWithParams(params Argon2IDParams) *Argon2IDHash {
	return &Argon2IDHash{params: params}
}

func (a *Argon2IDHash) Name() string { return "argon2id" }

// Hash хеширует пароль с использованием Argon2id.
// Формат: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
func (a *Argon2IDHash) Hash(password string) (string, error) {
	salt := make([]byte, a.params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, a.params.Time, a.params.Memory, a.params.Threads, a.params.KeyLen)

	// Encode in modular crypto format
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%x$%x",
		argon2.Version, a.params.Memory, a.params.Time, a.params.Threads, salt, hash)

	return encoded, nil
}

// Verify проверяет пароль против Argon2id хеша.
func (a *Argon2IDHash) Verify(password, hash string) (bool, error) {
	// Parse manually: формат $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	// Ищем последние два $ для разделения salt и hash
	lastDollar := lastIndexByte(hash, '$')
	if lastDollar < 0 {
		return false, fmt.Errorf("argon2id: invalid hash format (no last $)")
	}
	secondLastDollar := lastIndexByte(hash[:lastDollar], '$')
	if secondLastDollar < 0 {
		return false, fmt.Errorf("argon2id: invalid hash format (no second last $)")
	}

	saltHex := hash[secondLastDollar+1 : lastDollar]
	hashHex := hash[lastDollar+1:]

	// Parse parameters from the prefix
	var version int
	var memory, timeVal uint32
	var threads uint8
	_, err := fmt.Sscanf(hash[:secondLastDollar],
		"$argon2id$v=%d$m=%d,t=%d,p=%d",
		&version, &memory, &timeVal, &threads)
	if err != nil {
		return false, fmt.Errorf("argon2id parse params: %w", err)
	}

	// Decode hex salt
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false, fmt.Errorf("argon2id decode salt: %w", err)
	}

	// Re-hash with same params
	keyLen := uint32(len(hashHex) / 2)
	expected := argon2.IDKey([]byte(password), salt, timeVal, memory, threads, keyLen)

	expectedHex := fmt.Sprintf("%x", expected)
	return expectedHex == hashHex, nil
}

// lastIndexByte возвращает последний индекс байта c в s, или -1 если не найден.
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (a *Argon2IDHash) Cost() int { return int(a.params.Time) }

// ────────────────────────────────────────────────────────────────────────────
// BeltHash — belt-hash password hashing (СТБ) — stub
// ────────────────────────────────────────────────────────────────────────────

// BeltHash implements PasswordHashProvider using belt-hash (stub).
// ⚠ STUB: Использует bcrypt как fallback.
type BeltHash struct {
	fallback *BCryptHash
}

// NewBeltHash создаёт belt-hash провайдер (stub).
func NewBeltHash() *BeltHash {
	return &BeltHash{fallback: NewBCryptHash()}
}

func (b *BeltHash) Name() string { return "belt-hash" }

func (b *BeltHash) Hash(password string) (string, error) {
	// ⚠ Временно: bcrypt. Цель: belt-hash (СТБ 34.101.31).
	return b.fallback.Hash(password)
}

func (b *BeltHash) Verify(password, hash string) (bool, error) {
	return b.fallback.Verify(password, hash)
}

func (b *BeltHash) Cost() int { return b.fallback.Cost() }

// ────────────────────────────────────────────────────────────────────────────
// Factory
// ────────────────────────────────────────────────────────────────────────────

// PasswordHashFromProfile возвращает PasswordHashProvider на основе
// ComplianceProfile.
//
// BY: belt-hash (stub, fallback bcrypt)
// EU: Argon2id
// INTL: Argon2id
func PasswordHashFromProfile(profileHash string) (PasswordHashProvider, error) {
	switch profileHash {
	case "belt-hash":
		return NewBeltHash(), nil
	case "argon2id":
		return NewArgon2IDHash(), nil
	case "bcrypt":
		return NewBCryptHash(), nil
	default:
		// Fallback на Argon2id
		return NewArgon2IDHash(), nil
	}
}
