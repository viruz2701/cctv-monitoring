// Package templates — региональные шаблоны PDF для compliance отчётов CCTV Health Monitor.
//
// ═══════════════════════════════════════════════════════════════════════════════
// Compliance:
//   - Приказ МЧС РК № 55 (Лицензирование охранной деятельности, формат акта)
//   - Закон РК «О лицензировании» от 16.05.2014 № 202-V
//   - Закон РК «О разрешениях и уведомлениях» от 16.05.2014 № 202-V
//   - ISO 27001 A.8 (Asset management — перечень оборудования)
//   - Вступает в силу с 01.02.2026
//
// ═══════════════════════════════════════════════════════════════════════════════
package templates

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// MChSKZActData — данные для акта лицензирования по форме МЧС РК (Приказ №55).
// Вступает в силу с 01.02.2026.
type MChSKZActData struct {
	ActNumber         string                // Номер акта
	ActDate           string                // Дата составления (ДД.ММ.ГГГГ)
	LicenseNumber     string                // Номер лицензии МЧС
	LicenseSeries     string                // Серия лицензии
	LicenseIssueDate  string                // Дата выдачи лицензии (ДД.ММ.ГГГГ)
	LicenseExpiryDate string                // Срок действия лицензии (ДД.ММ.ГГГГ)
	OrganizationName  string                // Наименование организации-лицензиата
	OrganizationBIN   string                // БИН организации
	DirectorName      string                // ФИО руководителя
	ActType           string                // Тип акта: первичный / плановый / внеплановый
	EquipmentList     []MChSKZEquipmentItem // Перечень оборудования
	PremisesAddress   string                // Адрес места осуществления деятельности
	Conclusion        string                // Заключение
	InspectorName     string                // ФИО инспектора
}

// MChSKZEquipmentItem — единица оборудования в акте лицензирования МЧС РК.
type MChSKZEquipmentItem struct {
	Number    string // № п/п
	Name      string // Наименование оборудования
	Model     string // Марка/модель
	Serial    string // Заводской номер
	Quantity  string // Количество
	Condition string // Техническое состояние
	CertInfo  string // Сертификат соответствия
}

// MChSKZAct генерирует акт лицензирования по форме МЧС РК (Приказ №55).
//
// Формат: A4 portrait
// Вступает в силу с 01.02.2026.
//
// Содержит:
//   - Шапка с грифом МЧС РК
//   - Номер и срок действия лицензии
//   - Перечень оборудования (таблица)
//   - Заключение инспектора
//   - Подписи сторон
func MChSKZAct(data MChSKZActData) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 25)
	pdf.AddPage()

	// ── Шапка ─────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "Министерство по чрезвычайным ситуациям Республики Казахстан", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "Комитет противопожарной службы", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, "Бланк лицензирования (Приказ МЧС №55)", "", 1, "C", false, 0, " ")
	pdf.Ln(2)

	// ── Заголовок ─────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 8, "АКТ", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("лицензионного контроля № %s", data.ActNumber), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// ── Дата ──────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Дата составления: %s", data.ActDate), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Тип проверки: %s", data.ActType), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// ── Информация о лицензии ─────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "1. Сведения о лицензии", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Номер лицензии: %s %s", data.LicenseSeries, data.LicenseNumber), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Дата выдачи: %s", data.LicenseIssueDate), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Срок действия до: %s", data.LicenseExpiryDate), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// ── Информация о лицензиате ───────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "2. Сведения о лицензиате", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Организация: %s", data.OrganizationName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("БИН: %s", data.OrganizationBIN), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Руководитель: %s", data.DirectorName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Адрес осуществления деятельности: %s", data.PremisesAddress), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// ── Таблица оборудования ──────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "3. Перечень оборудования систем видеонаблюдения", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Приложение к акту лицензионного контроля", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Заголовки таблицы
	colWidths := []float64{8, 45, 30, 28, 14, 25, 40}
	headers := []string{"№", "Наименование", "Марка/модель", "Заводской №", "Кол-во", "Состояние", "Сертификат"}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Строки таблицы
	pdf.SetFont("Helvetica", "", 7)
	for _, eq := range data.EquipmentList {
		row := []string{eq.Number, eq.Name, eq.Model, eq.Serial, eq.Quantity, eq.Condition, eq.CertInfo}
		for i, c := range row {
			pdf.CellFormat(colWidths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(4)

	// ── Заключение ────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "4. Заключение по результатам проверки", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, data.Conclusion, "", "L", false)
	pdf.Ln(4)

	// ── Подписи ───────────────────────────────────────────────────────────
	sigY := pdf.GetY()
	if sigY > 240 {
		pdf.AddPage()
		sigY = pdf.GetY()
	}
	pdf.SetY(sigY)

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(95, 6, "Инспектор МЧС РК:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.CellFormat(95, 6, fmt.Sprintf("(%s)", data.InspectorName), "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.CellFormat(95, 6, "Лицензиат (руководитель):", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.CellFormat(95, 6, fmt.Sprintf("(%s)", data.DirectorName), "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.CellFormat(0, 6, "М.П.", "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// ── Footer ────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "I", 6)
	pdf.SetY(-18)
	pdf.CellFormat(0, 4, fmt.Sprintf("Приказ МЧС РК №55 | Лицензия %s № %s от %s | Действителен до: %s",
		data.LicenseSeries, data.LicenseNumber, data.LicenseIssueDate, data.LicenseExpiryDate), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("Сгенерировано: %s | Вступает в силу с 01.02.2026", time.Now().UTC().Format(time.RFC3339)), "", 1, "C", false, 0, "")

	return pdf
}
