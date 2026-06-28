// Package api — Well-Known URI handlers (RFC 8615, RFC 9116).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-N2: Vulnerability Disclosure Program (VDP)
//
// Соответствует:
//   - EU CRA (Dec 2027) — Coordinated Vulnerability Disclosure
//   - RFC 9116 — security.txt
//   - RFC 8615 — Well-Known URIs
//   - ISO 27001 A.6.1 — Information Security Roles & Responsibilities
//
// Эндпоинты:
//   - GET /.well-known/security.txt — Vulnerability Disclosure Policy
//   - GET /.well-known/security-policy — HTML version of SECURITY.md
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// defaultSecurityTxtContent — содержимое security.txt по умолчанию.
const defaultSecurityTxtContent = `# security.txt — RFC 9116 Vulnerability Disclosure
# CCTV Health Monitor — P0-N2: Vulnerability Disclosure Program (VDP)

Contact: mailto:security@gb-telemetry.com
Contact: https://github.com/gb-telemetry-collector/security/advisories/new
Encryption: https://gb-telemetry.com/.well-known/pgp-key.txt
Policy: https://github.com/gb-telemetry-collector/blob/main/SECURITY.md
Acknowledgments: https://gb-telemetry.com/security-advisories
Expires: 2027-06-28T20:00:00.000Z
Preferred-Languages: en, ru, zh
Canonical: https://gb-telemetry.com/.well-known/security.txt
`

// WellKnownHandler — handler для well-known URI.
type WellKnownHandler struct {
	securityTxtPath string
}

// NewWellKnownHandler создаёт новый WellKnownHandler.
//
// securityTxtPath — путь к файлу security.txt (может быть пустым,
// в этом случае используется встроенная константа).
func NewWellKnownHandler(securityTxtPath string) *WellKnownHandler {
	return &WellKnownHandler{
		securityTxtPath: securityTxtPath,
	}
}

// HandleSecurityTxt обслуживает GET /.well-known/security.txt.
//
// Возвращает security.txt (RFC 9116) с Content-Type text/plain.
func (h *WellKnownHandler) HandleSecurityTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Пробуем загрузить из файла (можно переопределить при деплое)
	if h.securityTxtPath != "" {
		absPath, err := filepath.Abs(h.securityTxtPath)
		if err == nil {
			data, err := os.ReadFile(absPath)
			if err == nil {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}
	}

	// Fallback на встроенную константу
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(defaultSecurityTxtContent))
}

// HandleSecurityPolicy обслуживает GET /.well-known/security-policy.
//
// Возвращает HTML-версию SECURITY.md с информацией о программе VDP.
func (h *WellKnownHandler) HandleSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Security Policy — CCTV Health Monitor</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; line-height: 1.6; color: #1e293b; }
  h1 { color: #0f172a; border-bottom: 2px solid #e2e8f0; padding-bottom: 8px; }
  h2 { color: #334155; margin-top: 32px; }
  code { background: #f1f5f9; padding: 2px 6px; border-radius: 4px; font-size: 0.9em; }
  table { border-collapse: collapse; width: 100%; margin: 16px 0; }
  th, td { border: 1px solid #e2e8f0; padding: 8px 12px; text-align: left; }
  th { background: #f8fafc; }
  .note { background: #eff6ff; border-left: 4px solid #3b82f6; padding: 12px 16px; margin: 16px 0; border-radius: 4px; }
</style>
</head>
<body>
<h1>Security Policy</h1>
<p>This page describes the Vulnerability Disclosure Program (VDP) for CCTV Health Monitor.</p>

<div class="note">
  <strong>Reporting a Vulnerability:</strong> Send details to 
  <a href="mailto:security@gb-telemetry.com">security@gb-telemetry.com</a>
  or use <a href="https://github.com/gb-telemetry-collector/security/advisories/new">GitHub Security Advisories</a>.
</div>

<h2>Supported Versions</h2>
<table>
  <tr><th>Version</th><th>Supported</th></tr>
  <tr><td>1.x</td><td>Active support</td></tr>
  <tr><td>< 1.0</td><td>Not released</td></tr>
</table>

<h2>Disclosure Timeline</h2>
<table>
  <tr><th>Phase</th><th>Duration</th></tr>
  <tr><td>Initial response</td><td>24 hours</td></tr>
  <tr><td>Confirmation / fix</td><td>7 days</td></tr>
  <tr><td>Patch release</td><td>30 days</td></tr>
  <tr><td>Public disclosure</td><td>90 days</td></tr>
</table>

<h2>Scope</h2>
<p><strong>In scope:</strong> Backend API, Frontend, Mobile, P2P Gateway, Auth, Encryption.</p>
<p><strong>Out of scope:</strong> Physical security, hardware vulns, social engineering, DoS, Self-XSS.</p>

<h2>Compliance</h2>
<ul>
  <li>EU Cyber Resilience Act (CRA) — Coordinated Vulnerability Disclosure</li>
  <li>RFC 9116 — security.txt</li>
  <li>ISO 27001 A.6.1 — Information Security Roles & Responsibilities</li>
  <li>OWASP ASVS V1.4 — Security Documentation</li>
  <li>IEC 62443-4-1 — Secure Development Lifecycle</li>
</ul>

<p><em>Last updated: 2026-06-28</em></p>
</body>
</html>`

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// mountWellKnownRoutes монтирует well-known URI маршруты на роутер.
//
// Эндпоинты публичные (без JWT), только GET.
// Соответствует: RFC 8615, RFC 9116, EU CRA.
func (s *Server) mountWellKnownRoutes(r chi.Router) {
	if s.wellKnownHandler == nil {
		return
	}

	r.Get("/.well-known/security.txt", s.wellKnownHandler.HandleSecurityTxt)
	r.Get("/.well-known/security-policy", s.wellKnownHandler.HandleSecurityPolicy)
}
