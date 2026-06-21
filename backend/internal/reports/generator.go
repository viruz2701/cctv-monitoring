// Package reports — генерация Excel и PDF отчётов для CMMS.
package reports

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"

	"gb-telemetry-collector/internal/models"
)

// ReportGenerator строит Excel/PDF отчёты.
type ReportGenerator struct {
	companyName string
}

// New создаёт ReportGenerator.
func New(companyName string) *ReportGenerator {
	return &ReportGenerator{companyName: companyName}
}

// ── Excel ────────────────────────────────────────────────────────────────────

// MaintenanceReportXLSX возвращает Excel-файл с отчётом по обслуживанию.
func (g *ReportGenerator) MaintenanceReportXLSX(data []models.MaintenanceReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Maintenance Report"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Device ID", "Device Name", "MTBF (hours)", "MTTR (min)", "Total WO", "Completed", "Overdue", "Total Cost"}
	g.writeXLSXHeader(f, sheet, headers)

	for i, r := range data {
		row := i + 2
		g.writeXLSXRow(f, sheet, row, []interface{}{
			r.DeviceID, r.DeviceName,
			fmt.Sprintf("%.1f", r.MTBF), fmt.Sprintf("%.1f", r.MTTR),
			r.TotalWorkOrders, r.CompletedCount, r.OverdueCount,
			fmt.Sprintf("%.2f", r.TotalCost),
		})
	}

	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 30)
	f.SetColWidth(sheet, "C", "H", 16)
	return f, nil
}

// SLAComplianceReportXLSX возвращает Excel-файл с отчётом по SLA.
func (g *ReportGenerator) SLAComplianceReportXLSX(data []models.SLAComplianceReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "SLA Compliance"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Priority", "Total WO", "Within SLA", "Breached", "Compliance %", "Avg Response (min)", "Avg Resolution (min)"}
	g.writeXLSXHeader(f, sheet, headers)

	for i, r := range data {
		row := i + 2
		g.writeXLSXRow(f, sheet, row, []interface{}{
			r.Priority, r.TotalWorkOrders, r.WithinSLA, r.BreachedSLA,
			fmt.Sprintf("%.1f%%", r.CompliancePercent),
			fmt.Sprintf("%.1f", r.AvgResponseTime), fmt.Sprintf("%.1f", r.AvgResolutionTime),
		})
	}

	f.SetColWidth(sheet, "A", "G", 18)
	return f, nil
}

// WorkOrdersXLSX возвращает Excel-файл со списком нарядов.
func (g *ReportGenerator) WorkOrdersXLSX(data []models.WorkOrder) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Work Orders"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Device", "Type", "Status", "Priority", "SLA Status", "Assigned To", "Created", "Completed"}
	g.writeXLSXHeader(f, sheet, headers)

	for i, wo := range data {
		row := i + 2
		created := wo.CreatedAt.Format("2006-01-02 15:04")
		completed := ""
		if wo.CompletedAt != nil {
			completed = wo.CompletedAt.Format("2006-01-02 15:04")
		}
		assignee := ""
		if wo.AssigneeName != "" {
			assignee = wo.AssigneeName
		}
		g.writeXLSXRow(f, sheet, row, []interface{}{
			wo.ID, wo.DeviceName, wo.Type, wo.Status, wo.Priority,
			wo.SLAStatus, assignee, created, completed,
		})
	}

	f.SetColWidth(sheet, "A", "A", 38)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetColWidth(sheet, "C", "F", 14)
	f.SetColWidth(sheet, "G", "G", 20)
	f.SetColWidth(sheet, "H", "I", 18)
	return f, nil
}

// SparePartsXLSX возвращает Excel-файл с инвентаризацией запчастей.
func (g *ReportGenerator) SparePartsXLSX(data []models.SparePart) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Spare Parts"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Name", "SKU", "Category", "Stock", "Min Stock", "Location", "Cost", "Supplier"}
	g.writeXLSXHeader(f, sheet, headers)

	for i, sp := range data {
		row := i + 2
		g.writeXLSXRow(f, sheet, row, []interface{}{
			sp.ID, sp.Name, sp.SKU, sp.Category, sp.Stock, sp.MinStock,
			sp.Location, fmt.Sprintf("%.2f", sp.Cost), sp.Supplier,
		})
	}

	f.SetColWidth(sheet, "A", "A", 38)
	f.SetColWidth(sheet, "B", "C", 22)
	f.SetColWidth(sheet, "D", "D", 16)
	f.SetColWidth(sheet, "E", "F", 12)
	f.SetColWidth(sheet, "G", "I", 18)
	return f, nil
}

// ── PDF ──────────────────────────────────────────────────────────────────────

func (g *ReportGenerator) newPDF(title string) *gofpdf.Fpdf {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Заголовок
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, g.companyName, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, title, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s", time.Now().UTC().Format(time.RFC3339)), "", 1, "C", false, 0, "")
	pdf.Ln(4)
	return pdf
}

// MaintenanceReportPDF возвращает PDF-файл с отчётом по обслуживанию.
func (g *ReportGenerator) MaintenanceReportPDF(data []models.MaintenanceReport) (*gofpdf.Fpdf, error) {
	pdf := g.newPDF("Maintenance Report")
	headers := []string{"Device ID", "Device Name", "MTBF", "MTTR", "Total", "Done", "Overdue", "Cost"}
	widths := []float64{30, 50, 20, 20, 18, 18, 22, 22}
	g.writePDFHeader(pdf, headers, widths)

	for _, r := range data {
		row := []string{
			r.DeviceID, r.DeviceName,
			fmt.Sprintf("%.1fh", r.MTBF), fmt.Sprintf("%.1fm", r.MTTR),
			fmt.Sprintf("%d", r.TotalWorkOrders), fmt.Sprintf("%d", r.CompletedCount),
			fmt.Sprintf("%d", r.OverdueCount), fmt.Sprintf("%.2f", r.TotalCost),
		}
		g.writePDFRow(pdf, row, widths)
	}
	return pdf, nil
}

// SLAComplianceReportPDF возвращает PDF-файл с отчётом по SLA.
func (g *ReportGenerator) SLAComplianceReportPDF(data []models.SLAComplianceReport) (*gofpdf.Fpdf, error) {
	pdf := g.newPDF("SLA Compliance Report")
	headers := []string{"Priority", "Total", "Within SLA", "Breached", "Compliance", "Avg Resp", "Avg Resol"}
	widths := []float64{30, 25, 30, 30, 35, 30, 30}
	g.writePDFHeader(pdf, headers, widths)

	for _, r := range data {
		row := []string{
			r.Priority, fmt.Sprintf("%d", r.TotalWorkOrders),
			fmt.Sprintf("%d", r.WithinSLA), fmt.Sprintf("%d", r.BreachedSLA),
			fmt.Sprintf("%.1f%%", r.CompliancePercent),
			fmt.Sprintf("%.1fm", r.AvgResponseTime), fmt.Sprintf("%.1fm", r.AvgResolutionTime),
		}
		g.writePDFRow(pdf, row, widths)
	}
	return pdf, nil
}

// WorkOrdersPDF возвращает PDF-файл со списком нарядов.
func (g *ReportGenerator) WorkOrdersPDF(data []models.WorkOrder) (*gofpdf.Fpdf, error) {
	pdf := g.newPDF("Work Orders")
	headers := []string{"ID", "Device", "Type", "Status", "Priority", "SLA", "Assignee", "Created"}
	widths := []float64{40, 38, 22, 22, 22, 22, 30, 36}
	g.writePDFHeader(pdf, headers, widths)

	for _, wo := range data {
		assignee := ""
		if wo.AssigneeName != "" {
			assignee = wo.AssigneeName
		}
		row := []string{
			wo.ID, wo.DeviceName, wo.Type, wo.Status, wo.Priority,
			wo.SLAStatus, assignee, wo.CreatedAt.Format("2006-01-02"),
		}
		g.writePDFRow(pdf, row, widths)
	}
	return pdf, nil
}

// SparePartsPDF возвращает PDF-файл с инвентаризацией запчастей.
func (g *ReportGenerator) SparePartsPDF(data []models.SparePart) (*gofpdf.Fpdf, error) {
	pdf := g.newPDF("Spare Parts Inventory")
	headers := []string{"ID", "Name", "SKU", "Category", "Stock", "Min", "Location", "Cost"}
	widths := []float64{35, 38, 25, 22, 14, 14, 30, 22}
	g.writePDFHeader(pdf, headers, widths)

	for _, sp := range data {
		row := []string{
			sp.ID, sp.Name, sp.SKU, sp.Category,
			fmt.Sprintf("%d", sp.Stock), fmt.Sprintf("%d", sp.MinStock),
			sp.Location, fmt.Sprintf("%.2f", sp.Cost),
		}
		g.writePDFRow(pdf, row, widths)
	}
	return pdf, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (g *ReportGenerator) writeXLSXHeader(f *excelize.File, sheet string, headers []string) {
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#94A3B8", Style: 1},
		},
	})
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}

func (g *ReportGenerator) writeXLSXRow(f *excelize.File, sheet string, row int, values []interface{}) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, v)
	}
}

func (g *ReportGenerator) writePDFHeader(pdf *gofpdf.Fpdf, headers []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
}

func (g *ReportGenerator) writePDFRow(pdf *gofpdf.Fpdf, cells []string, widths []float64) {
	pdf.SetFont("Helvetica", "", 8)
	for i, c := range cells {
		pdf.CellFormat(widths[i], 6, c, "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
}
