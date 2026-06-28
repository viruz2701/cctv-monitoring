# Security Policy — CCTV Health Monitor

## ════════════════════════════════════════════════════════════
# P0-N2: Vulnerability Disclosure Program (VDP)
# Соответствует: EU CRA (Coordinated Vulnerability Disclosure)
# RFC 9116, ISO 27001 A.6.1
# ════════════════════════════════════════════════════════════

## Supported Versions

We release security patches for the following versions:

| Version | Supported          |
|---------|--------------------|
| 1.x     | ✅ Active support  |
| < 1.0   | ❌ Not released    |

## Reporting a Vulnerability

We take the security of CCTV Health Monitor seriously. If you believe
you have found a security vulnerability, please report it to us as
described below.

### 📧 Contact

- **Primary**: security@gb-telemetry.com
- **PGP Key**: Available at `https://gb-telemetry.com/.well-known/pgp-key.txt`
- **Alternative**: https://github.com/gb-telemetry-collector/security/advisories/new

### 📋 What to Include

Please include the following details in your report:

1. **Type of vulnerability** (e.g., SQL injection, XSS, privilege escalation)
2. **Affected component** (backend Go, frontend React, mobile, P2P gateway)
3. **Steps to reproduce** — minimal, complete, reproducible
4. **Impact assessment** — what an attacker could achieve
5. **Suggested fix** (optional but appreciated)
6. **Your contact information** for follow-up

### 🕐 Disclosure Timeline

We follow Coordinated Vulnerability Disclosure (CVD):

| Phase                    | Duration     |
|--------------------------|--------------|
| Initial response         | ≤ 24 hours   |
| Confirmation/fix         | ≤ 7 days     |
| Patch release            | ≤ 30 days    |
| Public disclosure        | ≤ 90 days    |

### 🏆 Bug Bounty

This project currently operates a **vulnerability recognition program**
(not a monetary bug bounty). Researchers who report valid vulnerabilities
will be:

- Credited in our Security Advisories page
- Listed in our Hall of Fame (with permission)
- Eligible for swag (CCTV Health Monitor merchandise)

We may introduce a monetary bug bounty program in future releases.

### 🚫 Scope

**In scope:**
- Backend API (Go, Chi, PostgreSQL)
- Frontend (React 19, TypeScript, Vite)
- Mobile (React Native, Expo)
- P2P Gateway
- Authentication & Authorization
- Data encryption & key management

**Out of scope:**
- Physical security of CCTV cameras
- Third-party hardware vulnerabilities
- Social engineering attacks
- DoS/DDoS attacks (report anyway)
- Self-XSS

## Security Advisories

Security advisories are published at:
https://gb-telemetry.com/security-advisories

RSS feed: https://gb-telemetry.com/security-advisories/feed.xml

## Compliance

This Vulnerability Disclosure Program complies with:

- **EU Cyber Resilience Act (CRA)** — Coordinated Vulnerability Disclosure
- **RFC 9116** — security.txt
- **ISO 27001 A.6.1** — Information Security Roles & Responsibilities
- **OWASP ASVS V1.4** — Security Documentation
- **IEC 62443-4-1** — Secure Development Lifecycle (Vulnerability Handling)

## Hall of Fame

We thank the following security researchers for their contributions:

*(No entries yet — be the first!)*

---

*Last updated: 2026-06-28*
*Version: 1.0.0*
