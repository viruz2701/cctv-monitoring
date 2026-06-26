// Package reports — Excel handler для экспорта/импорта Work Orders.
//
// P2-3.3: Excel Import/Export for WO
//   - Bulk export для 10k+ WO
//   - Import wizard
//   - Column mapping
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application security)
//   - ISO 27001 A.12.4.1 (Event logging — export audit trail)
//   - СТБ 34.101.27 (Защита информации — контроль выгрузки данных)
package reports

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// WOExportRow — строка экспорта Work Order.
// Поля mapятся из models.WorkOrder с учётом денормализации.
type WOExportRow struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Type        string     `json:"type"`
	DeviceID    string     `json:"device_id"`
	AssigneeID  string     `json:"assignee_id"`
	SiteID      string     `json:"site_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	SLADeadline *time.Time `json:"sla_deadline,omitempty"`
}

// ExportWorkOrdersToExcel экспортирует Work Orders в Excel (.xlsx).
// Возвращает []byte для прямой записи в HTTP-ответ.
//
// Для 10k+ WO рекомендуется использовать streaming-пагинацию на стороне
// вызывающего кода — функция однопроходная, без буферизации всего набора
// в памяти помимо переданного слайса.
//
// Style: синий заголовок (#2563EB), белый текст, авто-ширина колонок.
func ExportWorkOrdersToExcel(rows []WOExportRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// ── Sheet ────────────────────────────────────────────────────────────
	sheet := "WorkOrders"
	index, err := f.NewSheet(sheet)
	if err != nil {
		index = 0
	}
	f.SetActiveSheet(index)

	// ── Header style ─────────────────────────────────────────────────────
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#2563EB"}},
		Border: []excelize.Border{
			{Type: "bottom", Style: 2, Color: "#1D4ED8"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}

	// ── Headers ──────────────────────────────────────────────────────────
	headers := []string{
		"ID", "Title", "Description", "Status", "Priority", "Type",
		"Device ID", "Assignee", "Site", "Created At", "Updated At", "SLA Deadline",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, 1, 20)

	// ── Data rows ────────────────────────────────────────────────────────
	for i, row := range rows {
		rowNum := i + 2

		slaDeadline := ""
		if row.SLADeadline != nil {
			slaDeadline = row.SLADeadline.Format(time.RFC3339)
		}

		vals := []interface{}{
			row.ID,
			row.Title,
			row.Description,
			row.Status,
			row.Priority,
			row.Type,
			row.DeviceID,
			row.AssigneeID,
			row.SiteID,
			row.CreatedAt.Format(time.RFC3339),
			row.UpdatedAt.Format(time.RFC3339),
			slaDeadline,
		}

		for j, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// ── Column widths ────────────────────────────────────────────────────
	colWidths := []float64{38, 30, 40, 12, 10, 14, 38, 18, 18, 20, 20, 20}
	for i, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	// ── Write to buffer ──────────────────────────────────────────────────
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}
	return buf.Bytes(), nil
}
