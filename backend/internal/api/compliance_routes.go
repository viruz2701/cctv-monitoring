// Package api — routes for Compliance & Fines Shield (KF-15.1.1).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - OWASP ASVS V3 (Session management — через JWT middleware)
//   - OWASP ASVS V4 (RBAC — admin/manager/owner)
//   - ISO 27001 A.9.2 (Access control — role-based)
//   - IEC 62443-3-3 SR 2.1 (Account management)
package api

import "github.com/go-chi/chi/v5"

// mountComplianceRoutes регистрирует маршруты Compliance & Fines Shield.
//
// Все маршруты доступны только для admin, manager, owner.
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2
func (s *Server) mountComplianceRoutes(r chi.Router) {
	r.Route("/api/v1/compliance", func(r chi.Router) {
		// ── Compliance & Fines Shield (KF-15.1.1) ─────────────────────
		// GET /api/v1/compliance/summary — общая сводка рисков
		r.Get("/summary", s.handleComplianceSummary)

		// GET /api/v1/compliance/risks — детальные риски (device_id, site_id)
		r.Get("/risks", s.handleComplianceRisks)

		// GET /api/v1/compliance/fines — таблица штрафов
		r.Get("/fines", s.handleComplianceFines)

		// POST /api/v1/compliance/refresh — принудительное обновление
		r.Post("/refresh", s.handleComplianceRefresh)

		// POST /api/v1/compliance/calculate — вычисление риска по параметрам
		r.Post("/calculate", s.handleComplianceCalculate)

		// ── P2-RU.2: 152-ФЗ Personal Data Features ───────────────────
		// Consent management
		r.Post("/personal-data/consent", s.handleGrantConsent)
		r.Post("/personal-data/consent/revoke", s.handleRevokeConsent)
		r.Get("/personal-data/consent", s.handleListConsents)

		// DSAR (Data Subject Access Request)
		r.Post("/personal-data/dsar", s.handleSubmitDSAR)
		r.Post("/personal-data/dsar/fulfill", s.handleFulfillDSAR)
		r.Post("/personal-data/dsar/reject", s.handleRejectDSAR)
		r.Get("/personal-data/dsar", s.handleListDSARs)

		// Data Inventory
		r.Post("/personal-data/inventory", s.handleRegisterInventoryItem)
		r.Get("/personal-data/inventory", s.handleGetInventory)
		r.Get("/personal-data/inventory/export", s.handleExportInventory)

		// Роскомнадзор Reporting
		r.Post("/personal-data/report/rkn", s.handleGenerateRoskomnadzorReport)

		// ── P2-EU.1: GDPR-Specific Features ──────────────────────────
		// Right to be Forgotten (Art. 17)
		r.Post("/gdpr/erasure", s.handleRequestErasure)
		r.Post("/gdpr/erasure/complete", s.handleCompleteErasure)
		r.Post("/gdpr/erasure/reject", s.handleRejectErasure)
		r.Get("/gdpr/erasure", s.handleListErasureRequests)

		// Data Portability (Art. 20)
		r.Post("/gdpr/portability", s.handleCreatePortabilityExport)
		r.Get("/gdpr/portability", s.handleListPortabilityExports)

		// Consent Audit Trail (Art. 7)
		r.Get("/gdpr/consent-audit", s.handleGetConsentAuditTrail)

		// DPIA (Art. 35)
		r.Post("/gdpr/dpia", s.handleGenerateDPIA)
		r.Get("/gdpr/dpia", s.handleListDPIAReports)

		// Schrems II / Data Transfers (Art. 44-49)
		r.Post("/gdpr/transfers", s.handleCreateTransferAgreement)
		r.Post("/gdpr/transfers/{id}/tia", s.handleCompleteTIA)
		r.Get("/gdpr/transfers", s.handleListTransferAgreements)
	})
}
