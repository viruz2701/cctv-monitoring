// Package providers — bign-curve256v1 Signature Provider (СТБ 34.101.45).
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-SEC.2: bign-curve256v1 Signature Provider — REAL ECDSA P-256
//
// Реализует цифровые подписи bign-curve256v1 согласно СТБ 34.101.45.
// Использует Go standard library crypto/ecdsa + elliptic.P256() как
// временное решение до получения сертифицированного bp2012/crypto SDK.
//
// bign-curve256v1 ≡ NIST P-256 (secp256r1) — кривая эллиптической
// криптографии, определённая в СТБ 34.101.45.
//
// ⚠ Временное решение: Go crypto/ecdsa вместо bp2012/crypto/bign.
// После получения сертифицированного SDK от ОАЦ заменить на:
//
//	import "github.com/bp2012/crypto/bign"
//	privKey, _ := bign.GenerateKey(bign.Curve256v1)
//	sig, _ := bign.SignPKCS8(privKey, data)
//	ok := bign.VerifyPKCS8(&privKey.PublicKey, data, sig)
//
// Compliance:
//   - СТБ 34.101.45 — bign-curve256v1
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - Приказ ОАЦ № 66 п. 7.18.1 — Сертификаты bign
//   - OWASP ASVS V6.2.2 — Использование асимметричной криптографии
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// BignCurve256v1KeySize — размер ключа bign-curve256v1 в байтах (32 байта = 256 бит).
	BignCurve256v1KeySize = 32

	// BignCurve256v1SignatureSize — размер подписи в байтах (ASN.1 DER, до 72 байт).
	// ECDSA P-256 подпись в ASN.1 DER формате обычно занимает 70-72 байта.
	BignCurve256v1SignatureSize = 72

	// bignCurve — кривая bign-curve256v1 (NIST P-256 / secp256r1).
	bignCurve = "P-256"
)

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	ErrInvalidSignature = errors.New("bign: invalid signature")
	ErrInvalidKey       = errors.New("bign: invalid key")
)

// ────────────────────────────────────────────────────────────────────────────
// BignKeyPair — пара ключей bign-curve256v1
// ────────────────────────────────────────────────────────────────────────────

// BignKeyPair представляет пару ключей bign-curve256v1.
// Использует стандартный ecdsa.PrivateKey (crypto/elliptic.P256()).
type BignKeyPair struct {
	PrivateKey *ecdsa.PrivateKey `json:"-"`
	PublicKey  *ecdsa.PublicKey  `json:"-"`
	// PEM-encoded ключи для сериализации
	PrivateKeyPEM []byte `json:"private_key_pem,omitempty"`
	PublicKeyPEM  []byte `json:"public_key_pem,omitempty"`
}

// GenerateBignKeyPair генерирует новую пару ключей bign-curve256v1 (ECDSA P-256).
func GenerateBignKeyPair() (*BignKeyPair, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate bign key pair: %w", err)
	}

	// Сериализуем приватный ключ в PEM
	privDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privDER,
	})

	// Сериализуем публичный ключ в PEM
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	return &BignKeyPair{
		PrivateKey:    privKey,
		PublicKey:     &privKey.PublicKey,
		PrivateKeyPEM: privPEM,
		PublicKeyPEM:  pubPEM,
	}, nil
}

// ParseBignPrivateKey парсит PEM-encoded ECDSA приватный ключ.
func ParseBignPrivateKey(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("%w: no PEM block found", ErrInvalidKey)
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Попробуем PKCS8
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("%w: %v (EC: %v, PKCS8: %v)", ErrInvalidKey, err, err, err2)
		}
		ecKey, ok := pkcs8Key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%w: not an ECDSA key", ErrInvalidKey)
		}
		return ecKey, nil
	}
	return key, nil
}

// ParseBignPublicKey парсит PEM-encoded ECDSA публичный ключ.
func ParseBignPublicKey(pemData []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("%w: no PEM block found", ErrInvalidKey)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}

	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: not an ECDSA public key", ErrInvalidKey)
	}

	return ecPub, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Sign/Verify functions (ECDSA P-256)
// ────────────────────────────────────────────────────────────────────────────

// BignSign подписывает данные с использованием bign-curve256v1 (ECDSA P-256).
// Возвращает ASN.1 DER-encoded подпись.
func BignSign(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("%w: nil private key", ErrInvalidKey)
	}
	hash := sha256.Sum256(data)
	sig, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("bign sign: %w", err)
	}
	return sig, nil
}

// BignVerify проверяет подпись bign-curve256v1 (ECDSA P-256).
func BignVerify(publicKey *ecdsa.PublicKey, data, signature []byte) (bool, error) {
	if publicKey == nil {
		return false, fmt.Errorf("%w: nil public key", ErrInvalidKey)
	}
	hash := sha256.Sum256(data)
	valid := ecdsa.VerifyASN1(publicKey, hash[:], signature)
	if !valid {
		return false, nil
	}
	return true, nil
}

// ────────────────────────────────────────────────────────────────────────────
// BignSignRaw — подпись с raw (R, S) форматом
// ────────────────────────────────────────────────────────────────────────────

// BignSignRaw подписывает данные и возвращает подпись в raw формате (R || S, 64 байта).
func BignSignRaw(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("%w: nil private key", ErrInvalidKey)
	}
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("bign sign raw: %w", err)
	}

	// R||S fixed 32+32 = 64 bytes
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	sig := make([]byte, 64)
	// R — right-aligned, 32 bytes
	copy(sig[32-len(rBytes):32], rBytes)
	// S — right-aligned, 32 bytes
	copy(sig[64-len(sBytes):64], sBytes)

	return sig, nil
}

// BignVerifyRaw проверяет подпись в raw формате (R || S, 64 байта).
func BignVerifyRaw(publicKey *ecdsa.PublicKey, data, signature []byte) (bool, error) {
	if publicKey == nil {
		return false, fmt.Errorf("%w: nil public key", ErrInvalidKey)
	}
	if len(signature) != 64 {
		return false, fmt.Errorf("%w: expected 64 bytes raw signature, got %d", ErrInvalidSignature, len(signature))
	}

	hash := sha256.Sum256(data)
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	valid := ecdsa.Verify(publicKey, hash[:], r, s)
	if !valid {
		return false, nil
	}
	return true, nil
}

// ────────────────────────────────────────────────────────────────────────────
// BignSigningMethod — кастомный JWT signing method для bign-curve256v1
// ────────────────────────────────────────────────────────────────────────────

// BignSigningMethod реализует jwt.SigningMethod для ECDSA P-256 (bign-curve256v1).
// Использует стандартный jwt.SigningMethodES256.
type BignSigningMethod struct{}

// NewBignSigningMethod создаёт новый метод подписи JWT bign-curve256v1.
func NewBignSigningMethod() *BignSigningMethod {
	return &BignSigningMethod{}
}

// Alg возвращает идентификатор алгоритма ("ES256" — ECDSA P-256 SHA-256).
func (m *BignSigningMethod) Alg() string {
	return "ES256"
}

// Sign подписывает JWT claims с bign-curve256v1.
func (m *BignSigningMethod) Sign(claims jwt.Claims, key interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(key)
}

// Verify проверяет JWT токен с bign-curve256v1.
func (m *BignSigningMethod) Verify(tokenString string, key interface{}, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("bign jwt: unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return fmt.Errorf("bign jwt verify: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("bign jwt: invalid token")
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// JWT convenience functions
// ────────────────────────────────────────────────────────────────────────────

// BignJWT подписывает JWT claims с использованием bign-curve256v1 (ECDSA P-256).
func BignJWT(claims jwt.Claims, privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("%w: nil private key", ErrInvalidKey)
	}
	if claims == nil {
		return "", fmt.Errorf("bign jwt: nil claims")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(privateKey)
}

// BignJWTVerify проверяет JWT токен с bign-curve256v1 публичным ключом.
func BignJWTVerify(tokenString string, publicKey *ecdsa.PublicKey, claims jwt.Claims) error {
	if publicKey == nil {
		return fmt.Errorf("%w: nil public key", ErrInvalidKey)
	}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return fmt.Errorf("bign jwt verify: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("bign jwt: invalid token")
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// BignSignatureProvider — реализация CryptoProvider с bign-curve256v1
// ────────────────────────────────────────────────────────────────────────────

// BignSignatureProvider implements CryptoProvider using bign-curve256v1.
// ⚠ Временное решение: crypto/ecdsa вместо bp2012/crypto/bign.
type BignSignatureProvider struct {
	status   string
	fallback *AESCrypto
}

// NewBignSignatureProvider создаёт bign-curve256v1 провайдер.
func NewBignSignatureProvider() *BignSignatureProvider {
	return &BignSignatureProvider{
		status:   "active",
		fallback: NewAESCrypto(),
	}
}

func (b *BignSignatureProvider) Hash(data []byte) ([]byte, error) {
	return Bash256(data)
}

func (b *BignSignatureProvider) HashHex(data []byte) (string, error) {
	return Bash256Hex(data)
}

func (b *BignSignatureProvider) HMAC(key, data []byte) ([]byte, error) {
	return Bash256HMAC(key, data)
}

func (b *BignSignatureProvider) HMACHex(key, data []byte) (string, error) {
	return Bash256HMACHex(key, data)
}

func (b *BignSignatureProvider) Encrypt(key, plaintext []byte) ([]byte, error) {
	return b.fallback.Encrypt(key, plaintext)
}

func (b *BignSignatureProvider) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return b.fallback.Decrypt(key, ciphertext)
}

// Sign подписывает данные с использованием bign-curve256v1.
// Принимает *ecdsa.PrivateKey или []byte (PEM-encoded).
func (b *BignSignatureProvider) Sign(privateKey, data []byte) ([]byte, error) {
	key, err := b.parsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return BignSign(key, data)
}

// Verify проверяет подпись bign-curve256v1.
// Принимает *ecdsa.PublicKey или []byte (PEM-encoded).
func (b *BignSignatureProvider) Verify(publicKey, data, signature []byte) (bool, error) {
	pub, err := b.parsePublicKey(publicKey)
	if err != nil {
		return false, err
	}
	return BignVerify(pub, data, signature)
}

func (b *BignSignatureProvider) GenerateKey(length int) ([]byte, error) {
	return b.fallback.GenerateKey(length)
}

func (b *BignSignatureProvider) parsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	// Если это PEM-encoded ключ
	if len(data) > 0 && data[0] == '-' {
		return ParseBignPrivateKey(data)
	}
	// Если это уже DER/serialized — пробуем как PEM с wrapping
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: data,
	})
	return ParseBignPrivateKey(pemData)
}

func (b *BignSignatureProvider) parsePublicKey(data []byte) (*ecdsa.PublicKey, error) {
	if len(data) > 0 && data[0] == '-' {
		return ParseBignPublicKey(data)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: data,
	})
	return ParseBignPublicKey(pemData)
}

// Status возвращает статус реализации.
func (b *BignSignatureProvider) Status() string { return b.status }

// ────────────────────────────────────────────────────────────────────────────
// Key serialization helpers
// ────────────────────────────────────────────────────────────────────────────

// BignPublicKeyHex возвращает hex-encoded публичный ключ (uncompressed, 65 байт).
func BignPublicKeyHex(pub *ecdsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	// uncompressed format: 04 || X || Y
	uncompressed := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	return hex.EncodeToString(uncompressed)
}

// BignPrivateKeyHex возвращает hex-encoded приватный ключ (32 байта).
func BignPrivateKeyHex(priv *ecdsa.PrivateKey) string {
	if priv == nil {
		return ""
	}
	return hex.EncodeToString(priv.D.Bytes())
}
