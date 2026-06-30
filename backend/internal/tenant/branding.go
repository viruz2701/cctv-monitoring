// Package tenant — Tenant Branding Store (P3-WL: White-Label Theming).
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-WL: White-Label Theming
//
// Обеспечивает per-tenant брендирование:
//   - Логотип, фавиконка, цвета
//   - Кастомный домен (CNAME)
//   - Email и PDF шаблоны
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - ISO 27001 A.8.1 (Asset management — tenant assets)
//   - OWASP ASVS V5 (Input validation)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
//
// ═══════════════════════════════════════════════════════════════════════════
package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// BrandingConfig — полная конфигурация бренда tenant'а.
type BrandingConfig struct {
	TenantID    string `json:"tenant_id"`
	CompanyName string `json:"company_name"`
	LogoURL     string `json:"logo_url"`
	FaviconURL  string `json:"favicon_url"`

	// Color scheme
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`

	// Font & CSS
	FontFamily string `json:"font_family"`
	CustomCSS  string `json:"custom_css"`

	// Custom domain (CNAME)
	CustomDomain           string     `json:"custom_domain"`
	CNAMEVerified          bool       `json:"cname_verified"`
	CNAMEVerifiedAt        *time.Time `json:"cname_verified_at,omitempty"`
	CNAMEVerificationToken string     `json:"cname_verification_token,omitempty"`

	// Email branding
	EmailHeaderLogoURL string `json:"email_header_logo_url"`
	EmailFooterText    string `json:"email_footer_text"`
	EmailPrimaryColor  string `json:"email_primary_color"`

	// PDF branding
	PDFLogoURL        string `json:"pdf_logo_url"`
	PDFPrimaryColor   string `json:"pdf_primary_color"`
	PDFSecondaryColor string `json:"pdf_secondary_color"`
	PDFFooterText     string `json:"pdf_footer_text"`

	// State
	IsActive  bool      `json:"is_active"`
	IsDefault bool      `json:"is_default"`
	IsLocked  bool      `json:"is_locked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// BrandingUpdateRequest — запрос на обновление бренда.
type BrandingUpdateRequest struct {
	CompanyName *string `json:"company_name,omitempty"`
	LogoURL     *string `json:"logo_url,omitempty"`
	FaviconURL  *string `json:"favicon_url,omitempty"`

	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`

	FontFamily *string `json:"font_family,omitempty"`
	CustomCSS  *string `json:"custom_css,omitempty"`

	CustomDomain *string `json:"custom_domain,omitempty"`

	EmailHeaderLogoURL *string `json:"email_header_logo_url,omitempty"`
	EmailFooterText    *string `json:"email_footer_text,omitempty"`
	EmailPrimaryColor  *string `json:"email_primary_color,omitempty"`

	PDFLogoURL        *string `json:"pdf_logo_url,omitempty"`
	PDFPrimaryColor   *string `json:"pdf_primary_color,omitempty"`
	PDFSecondaryColor *string `json:"pdf_secondary_color,omitempty"`
	PDFFooterText     *string `json:"pdf_footer_text,omitempty"`

	IsActive *bool `json:"is_active,omitempty"`
}

// DomainVerificationResult — результат верификации домена.
type DomainVerificationResult struct {
	Domain        string `json:"domain"`
	Verified      bool   `json:"verified"`
	VerifiedAt    string `json:"verified_at,omitempty"`
	Token         string `json:"token,omitempty"`
	ExpectedCNAME string `json:"expected_cname,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// BrandingStore
// ────────────────────────────────────────────────────────────────────────────

// BrandingStore управляет конфигурацией бренда tenant'ов.
type BrandingStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewBrandingStore создаёт новый BrandingStore.
func NewBrandingStore(pool *pgxpool.Pool) *BrandingStore {
	return &BrandingStore{
		pool:   pool,
		logger: slog.Default().With("component", "tenant.branding"),
	}
}

// ── CRUD ──────────────────────────────────────────────────────────────────

// Get возвращает конфигурацию бренда для tenant'а.
// Если конфигурация не найдена, возвращает BrandingConfig с tenant_id и is_default=true.
func (s *BrandingStore) Get(ctx context.Context, tenantID string) (*BrandingConfig, error) {
	cfg := &BrandingConfig{}
	err := s.pool.QueryRow(ctx, `
		SELECT
			tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color,
			font_family, custom_css,
			custom_domain, cname_verified, cname_verified_at, cname_verification_token,
			email_header_logo_url, email_footer_text, email_primary_color,
			pdf_logo_url, pdf_primary_color, pdf_secondary_color, pdf_footer_text,
			is_active, is_default, is_locked, created_at, updated_at, updated_by
		FROM tenant_branding
		WHERE tenant_id = $1
	`, tenantID).Scan(
		&cfg.TenantID, &cfg.CompanyName, &cfg.LogoURL, &cfg.FaviconURL,
		&cfg.PrimaryColor, &cfg.SecondaryColor, &cfg.AccentColor,
		&cfg.FontFamily, &cfg.CustomCSS,
		&cfg.CustomDomain, &cfg.CNAMEVerified, &cfg.CNAMEVerifiedAt, &cfg.CNAMEVerificationToken,
		&cfg.EmailHeaderLogoURL, &cfg.EmailFooterText, &cfg.EmailPrimaryColor,
		&cfg.PDFLogoURL, &cfg.PDFPrimaryColor, &cfg.PDFSecondaryColor, &cfg.PDFFooterText,
		&cfg.IsActive, &cfg.IsDefault, &cfg.IsLocked, &cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Tenant not found — return default config
			return &BrandingConfig{
				TenantID:       tenantID,
				PrimaryColor:   "#2563eb",
				SecondaryColor: "#6366f1",
				AccentColor:    "#06b6d4",
				FontFamily:     "Inter, system-ui, sans-serif",
				IsDefault:      true,
			}, nil
		}
		return nil, fmt.Errorf("get tenant branding: %w", err)
	}

	return cfg, nil
}

// Upsert создаёт или обновляет конфигурацию бренда tenant'а.
func (s *BrandingStore) Upsert(ctx context.Context, tenantID, updatedBy string, req *BrandingUpdateRequest) (*BrandingConfig, error) {
	// Build dynamic SET clause with coalesce
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_branding (
			tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color,
			font_family, custom_css, custom_domain,
			email_header_logo_url, email_footer_text, email_primary_color,
			pdf_logo_url, pdf_primary_color, pdf_secondary_color, pdf_footer_text,
			is_active, updated_by
		) VALUES (
			$1,
			COALESCE($2, ''),
			COALESCE($3, ''),
			COALESCE($4, ''),
			COALESCE($5, '#2563eb'),
			COALESCE($6, '#6366f1'),
			COALESCE($7, '#06b6d4'),
			COALESCE($8, 'Inter, system-ui, sans-serif'),
			COALESCE($9, ''),
			COALESCE($10, ''),
			COALESCE($11, ''),
			COALESCE($12, ''),
			COALESCE($13, '#2563eb'),
			COALESCE($14, ''),
			COALESCE($15, '#2563eb'),
			COALESCE($16, '#6366f1'),
			COALESCE($17, ''),
			COALESCE($18, false),
			$19
		)
		ON CONFLICT (tenant_id) DO UPDATE SET
			company_name = COALESCE(NULLIF($2, ''), tenant_branding.company_name),
			logo_url = COALESCE(NULLIF($3, ''), tenant_branding.logo_url),
			favicon_url = COALESCE(NULLIF($4, ''), tenant_branding.favicon_url),
			primary_color = COALESCE(NULLIF($5, ''), tenant_branding.primary_color),
			secondary_color = COALESCE(NULLIF($6, ''), tenant_branding.secondary_color),
			accent_color = COALESCE(NULLIF($7, ''), tenant_branding.accent_color),
			font_family = COALESCE(NULLIF($8, ''), tenant_branding.font_family),
			custom_css = COALESCE(NULLIF($9, ''), tenant_branding.custom_css),
			custom_domain = COALESCE(NULLIF($10, ''), tenant_branding.custom_domain),
			email_header_logo_url = COALESCE(NULLIF($11, ''), tenant_branding.email_header_logo_url),
			email_footer_text = COALESCE(NULLIF($12, ''), tenant_branding.email_footer_text),
			email_primary_color = COALESCE(NULLIF($13, ''), tenant_branding.email_primary_color),
			pdf_logo_url = COALESCE(NULLIF($14, ''), tenant_branding.pdf_logo_url),
			pdf_primary_color = COALESCE(NULLIF($15, ''), tenant_branding.pdf_primary_color),
			pdf_secondary_color = COALESCE(NULLIF($16, ''), tenant_branding.pdf_secondary_color),
			pdf_footer_text = COALESCE(NULLIF($17, ''), tenant_branding.pdf_footer_text),
			is_active = COALESCE($18, tenant_branding.is_active),
			updated_by = $19,
			updated_at = NOW()
	`, tenantID,
		nullableStr(req.CompanyName),
		nullableStr(req.LogoURL),
		nullableStr(req.FaviconURL),
		nullableStr(req.PrimaryColor),
		nullableStr(req.SecondaryColor),
		nullableStr(req.AccentColor),
		nullableStr(req.FontFamily),
		nullableStr(req.CustomCSS),
		nullableStr(req.CustomDomain),
		nullableStr(req.EmailHeaderLogoURL),
		nullableStr(req.EmailFooterText),
		nullableStr(req.EmailPrimaryColor),
		nullableStr(req.PDFLogoURL),
		nullableStr(req.PDFPrimaryColor),
		nullableStr(req.PDFSecondaryColor),
		nullableStr(req.PDFFooterText),
		nullableBool(req.IsActive),
		updatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert tenant branding: %w", err)
	}

	// Audit log
	s.logAudit(ctx, tenantID, "updated", "", "", "", updatedBy)

	return s.Get(ctx, tenantID)
}

// Delete сбрасывает конфигурацию бренда tenant'а в значения по умолчанию.
func (s *BrandingStore) Delete(ctx context.Context, tenantID, updatedBy string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenant_branding SET
			company_name = '',
			logo_url = '',
			favicon_url = '',
			primary_color = '#2563eb',
			secondary_color = '#6366f1',
			accent_color = '#06b6d4',
			font_family = 'Inter, system-ui, sans-serif',
			custom_css = '',
			custom_domain = '',
			cname_verified = false,
			cname_verified_at = NULL,
			cname_verification_token = '',
			email_header_logo_url = '',
			email_footer_text = '',
			email_primary_color = '#2563eb',
			pdf_logo_url = '',
			pdf_primary_color = '#2563eb',
			pdf_secondary_color = '#6366f1',
			pdf_footer_text = '',
			is_active = false,
			updated_by = $2,
			updated_at = NOW()
		WHERE tenant_id = $1
	`, tenantID, updatedBy)
	if err != nil {
		return fmt.Errorf("delete tenant branding: %w", err)
	}

	s.logAudit(ctx, tenantID, "reset", "", "", "", updatedBy)
	return nil
}

// ── Domain Verification ───────────────────────────────────────────────────

// GetVerificationToken возвращает токен верификации домена.
func (s *BrandingStore) GetVerificationToken(ctx context.Context, tenantID string) (*DomainVerificationResult, error) {
	cfg, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if cfg.CustomDomain == "" {
		return nil, fmt.Errorf("no custom domain configured for tenant %s", tenantID)
	}

	// Если токена нет — генерируем новый
	token := cfg.CNAMEVerificationToken
	if token == "" {
		token = fmt.Sprintf("cctv-verify-%s-%d", tenantID, time.Now().UnixNano())
		_, err := s.pool.Exec(ctx, `
			UPDATE tenant_branding
			SET cname_verification_token = $2
			WHERE tenant_id = $1
		`, tenantID, token)
		if err != nil {
			return nil, fmt.Errorf("generate verification token: %w", err)
		}
	}

	expectedCNAME := fmt.Sprintf("%s.verify.cctv-monitor.io", tenantID)

	return &DomainVerificationResult{
		Domain:        cfg.CustomDomain,
		Token:         token,
		ExpectedCNAME: expectedCNAME,
		Verified:      cfg.CNAMEVerified,
	}, nil
}

// VerifyDomain проверяет CNAME запись домена.
// В production здесь выполняется DNS lookup, сейчас — заглушка.
func (s *BrandingStore) VerifyDomain(ctx context.Context, tenantID string) (*DomainVerificationResult, error) {
	cfg, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if cfg.CustomDomain == "" {
		return nil, fmt.Errorf("no custom domain configured for tenant %s", tenantID)
	}

	token := cfg.CNAMEVerificationToken
	if token == "" {
		token = fmt.Sprintf("cctv-verify-%s-%d", tenantID, time.Now().UnixNano())
	}

	// ── DNS verification (stub) ──────────────────────────────────────
	// В production выполняется:
	//   1. Lookup CNAME for customDomain
	//   2. Compare with expectedCNAME
	//   3. Verify token in TXT record
	expectedCNAME := fmt.Sprintf("%s.verify.cctv-monitor.io", tenantID)

	// Stub: always succeeds for non-empty domain
	now := time.Now()
	_, err = s.pool.Exec(ctx, `
		UPDATE tenant_branding
		SET cname_verified = true,
		    cname_verified_at = $2,
		    updated_at = NOW()
		WHERE tenant_id = $1
	`, tenantID, now)
	if err != nil {
		return nil, fmt.Errorf("verify domain: %w", err)
	}

	// Log verification attempt
	s.logAudit(ctx, tenantID, "domain_verified", "custom_domain", cfg.CustomDomain, cfg.CustomDomain, "system")

	// Log to domain_verifications table
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO tenant_domain_verifications (tenant_id, domain, verification_token, verified, verified_at)
		VALUES ($1, $2, $3, true, $4)
	`, tenantID, cfg.CustomDomain, token, now)

	return &DomainVerificationResult{
		Domain:        cfg.CustomDomain,
		Token:         token,
		ExpectedCNAME: expectedCNAME,
		Verified:      true,
		VerifiedAt:    now.Format(time.RFC3339),
	}, nil
}

// ── Audit ─────────────────────────────────────────────────────────────────

func (s *BrandingStore) logAudit(ctx context.Context, tenantID, action, fieldName, oldValue, newValue, changedBy string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_branding_audit (tenant_id, action, field_name, old_value, new_value, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, action, fieldName, oldValue, newValue, changedBy)
	if err != nil {
		s.logger.Warn("failed to log branding audit", "tenant_id", tenantID, "action", action, "error", err)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────

func nullableStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullableBool(b *bool) interface{} {
	if b == nil {
		return nil
	}
	return *b
}
