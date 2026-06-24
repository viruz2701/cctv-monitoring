// Package api — CMMS domain routes: maintenance, work orders, spare parts, SLA, reports, technicians, mobile.
package api

import (
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// mountCMMSRoutes регистрирует все CMMS-маршруты на защищённом роутере.
func (s *Server) mountCMMSRoutes(r chi.Router) {
	// Maintenance Schedules
	r.Get("/api/v1/maintenance/schedules", s.listMaintenanceSchedules)
	r.Post("/api/v1/maintenance/schedules", s.createMaintenanceSchedule)
	r.Get("/api/v1/maintenance/schedules/due", s.getDueSchedules)
	r.Get("/api/v1/maintenance/schedules/{id}", s.getMaintenanceSchedule)
	r.Put("/api/v1/maintenance/schedules/{id}", s.updateMaintenanceSchedule)
	r.Delete("/api/v1/maintenance/schedules/{id}", s.deleteMaintenanceSchedule)
	r.Post("/api/v1/maintenance/schedules/{id}/complete", s.completeMaintenanceSchedule)

	// Work Orders
	r.Get("/api/v1/work-orders", s.listWorkOrders)
	r.Post("/api/v1/work-orders", s.createWorkOrder)
	r.Get("/api/v1/work-orders/{id}", s.getWorkOrder)
	r.Put("/api/v1/work-orders/{id}", s.updateWorkOrder)
	r.Delete("/api/v1/work-orders/{id}", s.deleteWorkOrder)
	r.Post("/api/v1/work-orders/{id}/assign", s.assignWorkOrder)
	r.Post("/api/v1/work-orders/{id}/start", s.startWorkOrder)
	r.Post("/api/v1/work-orders/{id}/complete", s.completeWorkOrder)
	r.Post("/api/v1/work-orders/{id}/cancel", s.cancelWorkOrder)
	r.Post("/api/v1/work-orders/{id}/photos", s.uploadWorkOrderPhotos)
	r.Post("/api/v1/work-orders/{id}/parts", s.addWorkOrderParts)

	// Spare Parts
	r.Get("/api/v1/spare-parts", s.listSpareParts)
	r.Post("/api/v1/spare-parts", s.createSparePart)
	r.Get("/api/v1/spare-parts/low-stock", s.getLowStockParts)
	r.Get("/api/v1/spare-parts/{id}", s.getSparePart)
	r.Put("/api/v1/spare-parts/{id}", s.updateSparePart)
	r.Delete("/api/v1/spare-parts/{id}", s.deleteSparePart)
	r.Post("/api/v1/spare-parts/{id}/adjust", s.adjustSparePartStock)

	// Spare Part Categories
	r.Get("/api/v1/spare-parts/categories", s.listSparePartCategories)
	r.Post("/api/v1/spare-parts/categories", s.createSparePartCategory)
	r.Put("/api/v1/spare-parts/categories/{id}", s.updateSparePartCategory)
	r.Delete("/api/v1/spare-parts/categories/{id}", s.deleteSparePartCategory)

	// Sites
	r.Get("/api/v1/sites", s.listSites)
	r.Post("/api/v1/sites", s.createSite)
	r.Get("/api/v1/sites/{id}", s.getSite)
	r.Put("/api/v1/sites/{id}", s.updateSite)
	r.Delete("/api/v1/sites/{id}", s.deleteSite)

	// Technician Management
	r.Get("/api/v1/technicians/workload", s.getAllTechnicianWorkloads)
	r.Get("/api/v1/technicians/{id}/workload", s.getTechnicianWorkload)
	r.Put("/api/v1/technicians/{id}/skills", s.updateTechnicianSkills)

	// Technician Site Assignments
	r.Get("/api/v1/technician-assignments", s.listTechnicianSiteAssignments)
	r.Post("/api/v1/technician-assignments", s.createTechnicianSiteAssignment)
	r.Put("/api/v1/technician-assignments/{id}", s.updateTechnicianSiteAssignment)
	r.Delete("/api/v1/technician-assignments/{id}", s.deleteTechnicianSiteAssignment)

	// SLA & Reports
	r.Get("/api/v1/sla/config", s.getSLAConfig)
	r.Put("/api/v1/sla/config/{priority}", s.updateSLAConfig)
	r.Get("/api/v1/reports/maintenance", s.getMaintenanceReport)
	r.Get("/api/v1/reports/sla-compliance", s.getSLAComplianceReport)

	// Export (Excel/PDF)
	r.Get("/api/v1/export/maintenance/xlsx", s.exportMaintenanceXLSX)
	r.Get("/api/v1/export/maintenance/pdf", s.exportMaintenancePDF)
	r.Get("/api/v1/export/sla-compliance/xlsx", s.exportSLAComplianceXLSX)
	r.Get("/api/v1/export/sla-compliance/pdf", s.exportSLACompliancePDF)
	r.Get("/api/v1/export/work-orders/xlsx", s.exportWorkOrdersXLSX)
	r.Get("/api/v1/export/work-orders/pdf", s.exportWorkOrdersPDF)
	r.Get("/api/v1/export/spare-parts/xlsx", s.exportSparePartsXLSX)
	r.Get("/api/v1/export/spare-parts/pdf", s.exportSparePartsPDF)

	// Mobile API — rate-limited (100 req/min/IP)
	r.Group(func(r chi.Router) {
		r.Use(s.newRateLimiterMiddleware(100, time.Minute))
		r.Get("/api/v1/mobile/work-orders", s.listMobileWorkOrders)
		r.Get("/api/v1/mobile/work-orders/{id}", s.getMobileWorkOrder)
		r.Post("/api/v1/mobile/work-orders/{id}/start", s.startMobileWorkOrder)
		r.Post("/api/v1/mobile/work-orders/{id}/verify", s.handleVerifyWorkOrder)
		r.Post("/api/v1/mobile/work-orders/{id}/complete", s.completeMobileWorkOrder)
		r.Post("/api/v1/mobile/work-orders/{id}/photos", s.uploadMobileWorkOrderPhoto)
		r.Post("/api/v1/mobile/push-token", s.registerMobilePushToken)
		r.Get("/api/v1/mobile/profile", s.getMobileTechnicianProfile)
		r.Get("/api/v1/mobile/stats", s.getMobileTechnicianStats)
	})
}

// mountProtectedCMMSRoutes оборачивает CMMS-роутер в JWT-авторизацию.
func (s *Server) mountProtectedCMMSRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		s.mountCMMSRoutes(r)
	})
}
