// Package providers — Regional Crypto Provider implementations.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: Regional Crypto Providers
//
// Провайдеры:
//   - BeltCrypto: belt-GCM (СТБ 34.101.31) — для BY региона (stub до bp2012/crypto)
//   - AESCrypto: AES-256-GCM — для EU/US/INTL регионов
//   - GOSTCrypto: GOST 28147-89 — stub для RU (full impl в P2-RU)
//   - SMCrypto: SM4 — stub для CN (full impl в P2-CN)
//
// Automatic provider selection via ComplianceProfile.
//
// Compliance:
//   - СТБ 34.101.30 — Криптографические алгоритмы
//   - СТБ 34.101.31 — belt-gcm
//   - ГОСТ 28147-89, ГОСТ Р 34.10-2012
//   - GM/T 0002-2012 (SM4), GM/T 0003-2012 (SM2), GM/T 0004-2012 (SM3)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"fmt"

	"gb-telemetry-collector/internal/compliance"
	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// ProviderFactory — создаёт CryptoProvider на основе ComplianceProfile
// ────────────────────────────────────────────────────────────────────────────

// NewFromProfile создаёт CryptoProvider на основе compliance профиля.
//
// Автоматический выбор провайдера:
//   - BY (СТБ): belt-gcm — BeltCrypto (stub до bp2012/crypto)
//   - EU (GDPR): AES-256-GCM — AESCrypto
//   - INTL (ISO 27001): AES-256-GCM — AESCrypto
//   - RU (ГОСТ): GOST 28147-89 — GOSTCrypto (stub)
//   - CN (SM4): SM4 — SMCrypto (stub)
//
// Возвращает ошибку для неподдерживаемых регионов или неизвестных провайдеров.
func NewFromProfile(p compliance.ComplianceProfile) (stb.CryptoProvider, error) {
	if p == nil {
		return nil, fmt.Errorf("crypto provider: nil compliance profile")
	}

	switch p.Region() {
	case compliance.RegionBY:
		return NewBeltCrypto(), nil
	case compliance.RegionEU, compliance.RegionINTL:
		return NewAESCrypto(), nil
	case compliance.RegionRU:
		return NewGOSTCrypto(), nil
	case compliance.RegionCN:
		return NewSMCrypto(), nil
	default:
		return nil, fmt.Errorf("crypto provider: unsupported region %s", p.Region())
	}
}

// MustFromProfile создаёт CryptoProvider из профиля и паникует при ошибке.
func MustFromProfile(p compliance.ComplianceProfile) stb.CryptoProvider {
	provider, err := NewFromProfile(p)
	if err != nil {
		panic(fmt.Sprintf("crypto provider: %v", err))
	}
	return provider
}

// ────────────────────────────────────────────────────────────────────────────
// Provider metadata
// ────────────────────────────────────────────────────────────────────────────

// ProviderInfo содержит метаданные о криптопровайдере.
type ProviderInfo struct {
	Name        string `json:"name"`
	Algorithm   string `json:"algorithm"`
	KeySizeBits int    `json:"key_size_bits"`
	Region      string `json:"region"`
	Standard    string `json:"standard"`
	Status      string `json:"status"` // "active" | "stub" | "experimental"
}

// Info возвращает метаданные провайдера.
func Info(provider stb.CryptoProvider) *ProviderInfo {
	switch p := provider.(type) {
	case *BeltCrypto:
		return &ProviderInfo{
			Name:        "belt-gcm",
			Algorithm:   "belt-GCM (СТБ 34.101.31)",
			KeySizeBits: 256,
			Region:      "BY",
			Standard:    "СТБ 34.101.30/31",
			Status:      p.status,
		}
	case *AESCrypto:
		return &ProviderInfo{
			Name:        "aes-256-gcm",
			Algorithm:   "AES-256-GCM",
			KeySizeBits: 256,
			Region:      "INTL",
			Standard:    "NIST SP 800-38D",
			Status:      "active",
		}
	case *GOSTCrypto:
		return &ProviderInfo{
			Name:        "gost-28147-89",
			Algorithm:   "ГОСТ 28147-89 (Магма/Кузнечик)",
			KeySizeBits: 256,
			Region:      "RU",
			Standard:    "ГОСТ Р 34.12-2015",
			Status:      p.status,
		}
	case *SMCrypto:
		return &ProviderInfo{
			Name:        "sm4",
			Algorithm:   "SM4 (国密)",
			KeySizeBits: 128,
			Region:      "CN",
			Standard:    "GM/T 0002-2012",
			Status:      p.status,
		}
	default:
		return &ProviderInfo{
			Name:   "unknown",
			Status: "unknown",
		}
	}
}
