// Package api — export handlers for Excel/PDF report generation.
package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/reports"
)

// ── Export Handlers ──────────────────────────────────────────────────────────

// exportMaintenanceXLSX возвращает Excel-отчёт по обслуживанию.
func (s *Server) exportMaintenanceXLSX(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetMaintenanceReport(r.Context())
	if err != nil {
		respondError(w, r, NewInternalError("failed to get maintenance report", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	f, err := gen.MaintenanceReportXLSX(report)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate Excel", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write Excel", err))
		return
	}

	filename := fmt.Sprintf("maintenance_report_%s.xlsx", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportMaintenancePDF возвращает PDF-отчёт по обслуживанию.
func (s *Server) exportMaintenancePDF(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetMaintenanceReport(r.Context())
	if err != nil {
		respondError(w, r, NewInternalError("failed to get maintenance report", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	pdf, err := gen.MaintenanceReportPDF(report)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate PDF", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write PDF", err))
		return
	}

	filename := fmt.Sprintf("maintenance_report_%s.pdf", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportSLAComplianceXLSX возвращает Excel-отчёт по соблюдению SLA.
func (s *Server) exportSLAComplianceXLSX(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetSLAComplianceReport(r.Context())
	if err != nil {
		respondError(w, r, NewInternalError("failed to get SLA compliance report", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	f, err := gen.SLAComplianceReportXLSX(report)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate Excel", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write Excel", err))
		return
	}

	filename := fmt.Sprintf("sla_compliance_%s.xlsx", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportSLACompliancePDF возвращает PDF-отчёт по соблюдению SLA.
func (s *Server) exportSLACompliancePDF(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetSLAComplianceReport(r.Context())
	if err != nil {
		respondError(w, r, NewInternalError("failed to get SLA compliance report", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	pdf, err := gen.SLAComplianceReportPDF(report)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate PDF", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write PDF", err))
		return
	}

	filename := fmt.Sprintf("sla_compliance_%s.pdf", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportWorkOrdersXLSX возвращает Excel-файл со списком нарядов.
func (s *Server) exportWorkOrdersXLSX(w http.ResponseWriter, r *http.Request) {
	filters := map[string]interface{}{"limit": 10000}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	workOrders, err := s.cmmsRouter.GetWorkOrders(r.Context(), filters)
	if err != nil {
		respondError(w, r, NewInternalError("failed to get work orders", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	f, err := gen.WorkOrdersXLSX(workOrders)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate Excel", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write Excel", err))
		return
	}

	filename := fmt.Sprintf("work_orders_%s.xlsx", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportWorkOrdersPDF возвращает PDF-файл со списком нарядов.
func (s *Server) exportWorkOrdersPDF(w http.ResponseWriter, r *http.Request) {
	filters := map[string]interface{}{"limit": 10000}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	workOrders, err := s.cmmsRouter.GetWorkOrders(r.Context(), filters)
	if err != nil {
		respondError(w, r, NewInternalError("failed to get work orders", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	pdf, err := gen.WorkOrdersPDF(workOrders)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate PDF", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write PDF", err))
		return
	}

	filename := fmt.Sprintf("work_orders_%s.pdf", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportSparePartsXLSX возвращает Excel-файл с инвентаризацией запчастей.
func (s *Server) exportSparePartsXLSX(w http.ResponseWriter, r *http.Request) {
	parts, err := s.cmmsRouter.GetSpareParts(r.Context(), map[string]interface{}{"limit": 10000})
	if err != nil {
		respondError(w, r, NewInternalError("failed to get spare parts", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	f, err := gen.SparePartsXLSX(parts)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate Excel", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write Excel", err))
		return
	}

	filename := fmt.Sprintf("spare_parts_%s.xlsx", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// exportSparePartsPDF возвращает PDF-файл с инвентаризацией запчастей.
func (s *Server) exportSparePartsPDF(w http.ResponseWriter, r *http.Request) {
	parts, err := s.cmmsRouter.GetSpareParts(r.Context(), map[string]interface{}{"limit": 10000})
	if err != nil {
		respondError(w, r, NewInternalError("failed to get spare parts", err))
		return
	}

	gen := reports.New("CCTV Monitoring Platform")
	pdf, err := gen.SparePartsPDF(parts)
	if err != nil {
		respondError(w, r, NewInternalError("failed to generate PDF", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := pdf.Output(buf); err != nil {
		respondError(w, r, NewInternalError("failed to write PDF", err))
		return
	}

	filename := fmt.Sprintf("spare_parts_%s.pdf", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}
