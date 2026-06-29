// Package templates — региональные шаблоны PDF для compliance отчётов CCTV Health Monitor.
//
// ═══════════════════════════════════════════════════════════════════════════════
// Compliance:
//   - KVKK №6698 (Kişisel Verilerin Korunması Kanunu — Закон о защите ПДн Турции)
//   - KVKK Art. 10 (Data Controller Obligations — Privacy Notice)
//   - KVKK Art. 12 (Data Security — Technical and Administrative Measures)
//   - KVKK Art. 15 (Data Subject Rights — Application procedures)
//   - VERBIS (Veri Sorumluları Sicili — Реестр операторов ПДн)
//   - TS EN 62676 (CCTV Standard Turkey)
//   - ISO 27001 A.5.1 (Information security policies — DPIA)
//   - ISO 31000 (Risk Management — PIA methodology)
//   - GDPR Art. 35 (DPIA as reference methodology)
//
// ═══════════════════════════════════════════════════════════════════════════════
package templates

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// KVKKPIAData — данные для Privacy Impact Assessment по KVKK №6698.
// Соответствие: KVKK Art. 10, 12, 15; VERBIS registration; TS EN 62676.
type KVKKPIAData struct {
	AssessmentID     string             // Идентификатор PIA (KVKK-PIA-YYYY-NNN)
	AssessmentDate   string             // Дата оценки (ДД.ММ.ГГГГ)
	OrganizationName string             // Наименование организации (Data Controller)
	OrganizationID   string             // VERBIS registration number
	DataOfficerName  string             // ФИО Data Protection Officer / ответственного
	DPOContact       string             // Контакт DPO (email/телефон)
	Purposes         []string           // Цели обработки ПД (KVKK Art. 4-6)
	DataCategories   []KVVKDataCategory // Категории обрабатываемых данных
	ProcessingDesc   string             // Описание процессов обработки
	LegalBasis       string             // Правовое основание (KVKK Art. 5)
	ThirdParties     []string           // Третьи лица / трансфер данных (KVKK Art. 8-9)
	RiskAssessment   string             // Оценка рисков
	Mitigations      []KVKKMitigation   // Меры защиты
	DataRetention    string             // Сроки хранения (KVKK Art. 7)
	Conclusion       string             // Заключение
	ReviewDate       string             // Дата следующей ревизии (ДД.ММ.ГГГГ)
}

// KVVKDataCategory — категория обрабатываемых данных.
type KVVKDataCategory struct {
	Name        string // Название категории
	Description string // Описание
	Purpose     string // Цель обработки
	Retention   string // Срок хранения
}

// KVKKMitigation — мера защиты данных.
type KVKKMitigation struct {
	Measure     string // Наименование меры
	Type        string // Тип: technical / administrative / legal
	Status      string // Статус: implemented / planned / n/a
	Description string // Описание
}

// KVVKPIA генерирует Privacy Impact Assessment по требованиям KVKK №6698.
//
// Формат: A4 portrait
// Содержит:
//   - Шапка с VERBIS registration reference
//   - Data Processing Inventory (реестр обработки)
//   - Risk Assessment (оценка рисков)
//   - Mitigation Measures (меры защиты)
//   - Заключение и подписи
//   - Compliance footer с KVKK ссылками
//
// Ссылки: KVKK №6698 Art. 10, 12, 15; VERBIS; TS EN 62676
func KVVKPIA(data KVKKPIAData) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 25)
	pdf.AddPage()

	// ── Шапка ─────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(0, 51, 153) // Dark blue header
	pdf.CellFormat(0, 6, "PRIVACY IMPACT ASSESSMENT (PIA)", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 5, "Kişisel Verilerin Korunması Kanunu (KVKK) No. 6698", "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, fmt.Sprintf("PIA ID: %s", data.AssessmentID), "", 1, "R", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Assessment Date: %s", data.AssessmentDate), "", 1, "R", false, 0, "")
	pdf.Ln(3)

	// ── VERBIS Registration ────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 240, 255)
	pdf.CellFormat(0, 6, "VERBIS Registration Reference", "", 1, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, fmt.Sprintf("Data Controller: %s", data.OrganizationName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("VERBIS Reg. No: %s", data.OrganizationID), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("DPO: %s (%s)", data.DataOfficerName, data.DPOContact), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// ── 1. Purposes of Processing ──────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "1. Processing Purposes (KVKK Art. 4-6)", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, p := range data.Purposes {
		pdf.CellFormat(0, 5, fmt.Sprintf("  - %s", p), "", 1, "L", false, 0, "")
	}
	pdf.Ln(2)

	// ── 2. Legal Basis ─────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "2. Legal Basis (KVKK Art. 5)", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, data.LegalBasis, "", "L", false)
	pdf.Ln(2)

	// ── 3. Processing Description ──────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "3. Processing Description", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, data.ProcessingDesc, "", "L", false)
	pdf.Ln(2)

	// ── 4. Data Processing Inventory (Table) ──────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "4. Data Processing Inventory", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	colWidths := []float64{35, 55, 50, 50}
	headers := []string{"Category", "Description", "Purpose", "Retention"}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 7)
	for _, cat := range data.DataCategories {
		row := []string{cat.Name, cat.Description, cat.Purpose, cat.Retention}
		for i, c := range row {
			pdf.CellFormat(colWidths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(3)

	// ── 5. Third Party Transfers ───────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "5. Third Party Transfers (KVKK Art. 8-9)", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	if len(data.ThirdParties) == 0 {
		pdf.CellFormat(0, 5, "  No third party transfers", "", 1, "L", false, 0, "")
	} else {
		for _, tp := range data.ThirdParties {
			pdf.CellFormat(0, 5, fmt.Sprintf("  - %s", tp), "", 1, "L", false, 0, "")
		}
	}
	pdf.Ln(2)

	// ── 6. Risk Assessment ─────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "6. Risk Assessment", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, data.RiskAssessment, "", "L", false)
	pdf.Ln(2)

	// ── 7. Mitigation Measures (Table) ─────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "7. Technical & Administrative Measures (KVKK Art. 12)", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	mitColWidths := []float64{45, 30, 25, 90}
	mitHeaders := []string{"Measure", "Type", "Status", "Description"}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range mitHeaders {
		pdf.CellFormat(mitColWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 7)
	for _, m := range data.Mitigations {
		row := []string{m.Measure, m.Type, m.Status, m.Description}
		for i, c := range row {
			pdf.CellFormat(mitColWidths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(3)

	// ── 8. Data Retention ──────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "8. Data Retention (KVKK Art. 7)", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, fmt.Sprintf("Retention policy: %s", data.DataRetention), "", "L", false)
	pdf.Ln(2)

	// ── 9. Conclusion ──────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 6, "9. Conclusion", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, data.Conclusion, "", "L", false)
	pdf.Ln(2)

	pdf.CellFormat(0, 5, fmt.Sprintf("Next review: %s", data.ReviewDate), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// ── Signatures ─────────────────────────────────────────────────────────
	sigY := pdf.GetY()
	if sigY > 230 {
		pdf.AddPage()
		sigY = pdf.GetY()
	}
	pdf.SetY(sigY)

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(95, 6, "Data Protection Officer:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.CellFormat(95, 6, fmt.Sprintf("(%s)", data.DataOfficerName), "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.CellFormat(95, 6, "Data Controller (Authorized):", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// ── Footer ────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "I", 6)
	pdf.SetY(-18)
	pdf.CellFormat(0, 4, fmt.Sprintf("KVKK No. 6698 | VERBIS: %s | PIA ID: %s", data.OrganizationID, data.AssessmentID), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("TS EN 62676 | Generated: %s | Next review: %s",
		time.Now().UTC().Format(time.RFC3339), data.ReviewDate), "", 1, "C", false, 0, "")

	return pdf
}
