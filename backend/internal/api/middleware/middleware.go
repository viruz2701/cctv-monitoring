// Package middleware — API middleware с OWASP ASVS L3 compliance.
//
// Содержит standalone middleware функции, которые могут быть импортированы
// маршрутизатором api. Server-bound middleware (требующие *Server) остаются
// в пакете api.
//
// Соответствует:
//   - OWASP ASVS V5.3.3 (CSP nonce)
//   - OWASP ASVS V9.1 (CORS whitelist)
//   - ISO 27001 A.13.2 (Access control — CORS)
//   - ISO 27001 A.13.2.3 (Content Security Policy)
package middleware
