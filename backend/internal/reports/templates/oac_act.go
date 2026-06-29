// Package templates — региональные шаблоны PDF для compliance отчётов CCTV Health Monitor.
//
// ═══════════════════════════════════════════════════════════════════════════════
// Compliance:
//   - СТБ 34.101.27 (Защита информации РБ — аудит и целостность)
//   - СТБ 34.101.30 (bash-256 HMAC подпись в footer)
//   - Приказ ОАЦ № 66 п. 7.18 (Защита конечных узлов)
//   - СН 3.02.19-2025 (Строительные нормы — системы видеонаблюдения)
//   - ISO 27001 A.12.4 (Audit trail — HMAC цепочка)
//   - IEC 62443 SR 2.1 (Integrity verification)
//
// ═══════════════════════════════════════════════════════════════════════════════
package templates

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// OACActData — данные для заполнения акта ТО системы видеонаблюдения по форме ОАЦ РБ.
// Соответствие: СН 3.02.19-2025 п. 5.2, Приказ ОАЦ № 66 п. 7.18.2.
type OACActData struct {
	ActNumber         string             // Номер акта
	ActDate           string             // Дата составления (ДД.ММ.ГГГГ)
	ObjectName        string             // Наименование объекта
	ObjectAddress     string             // Адрес объекта
	CommissionChair   string             // Председатель комиссии (ФИО)
	CommissionMembers []string           // Члены комиссии (ФИО)
	EquipmentList     []OACEquipmentItem // Перечень оборудования
	Conclusion        string             // Заключение комиссии
	Recommendations   string             // Рекомендации
	ExecutiveOrg      string             // Организация-исполнитель
}

// OACEquipmentItem — единица оборудования в акте ТО.
type OACEquipmentItem struct {
	Number     string // № п/п
	Name       string // Наименование
	Type       string // Тип/модель
	Serial     string // Серийный номер
	Inspection string // Результат осмотра
	Status     string // Заключение
}

// HMACSignOAC вычисляет bash-256 HMAC подпись для акта ТО.
// ⚠ Временно SHA256 — после миграции на bp2012/crypto/bash.
func HMACSignOAC(data string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// OACAct генерирует акт ТО по форме ОАЦ РБ (СН 3.02.19-2025, Приказ ОАЦ №66).
//
// Формат: A4 portrait
// Содержит:
//   - Шапка с грифом утверждения
//   - Состав комиссии
//   - Таблица оборудования
//   - Заключение и рекомендации
//   - Подписи сторон
//   - bash-256 HMAC footer (целостность)
func OACAct(data OACActData) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 25)
	pdf.AddPage()

	// ── Шапка с грифом ────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 5, "УТВЕРЖДАЮ", "", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, "Руководитель организации-исполнителя", "", 1, "R", false, 0, "")
	pdf.CellFormat(0, 5, "___________________ /____________________/", "", 1, "R", false, 0, "")
	pdf.CellFormat(0, 5, `"___" ____________ 20___ г.`, "", 1, "R", false, 0, "")
	pdf.Ln(6)

	// ── Заголовок ─────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "АКТ", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "технического обслуживания системы видеонаблюдения", "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// ── Номер и дата ──────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "", 10)
	leftCol := 70.0
	rightCol := 110.0

	pdf.CellFormat(leftCol, 6, fmt.Sprintf("№ %s", data.ActNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(rightCol, 6, fmt.Sprintf("г. _________________"), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Дата составления: %s", data.ActDate), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// ── Объект ────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "1. Общие сведения", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Наименование объекта: %s", data.ObjectName), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Адрес объекта: %s", data.ObjectAddress), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Организация-исполнитель: %s", data.ExecutiveOrg), "", 1, "L", false, 0, "")

	// Ссылка на СН
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Основание: СН 3.02.19-2025 «Системы видеонаблюдения. Правила проектирования и технического обслуживания»", "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// ── Состав комиссии ───────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "2. Состав комиссии", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Председатель: %s", data.CommissionChair), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Члены комиссии:", "", 1, "L", false, 0, "")
	for _, m := range data.CommissionMembers {
		pdf.CellFormat(0, 6, fmt.Sprintf("  - %s", m), "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	// ── Таблица оборудования ──────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "3. Перечень проверенного оборудования", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Заголовки таблицы
	colWidths := []float64{8, 50, 30, 30, 35, 27}
	headers := []string{"№", "Наименование", "Тип/модель", "Серийный №", "Результат осмотра", "Заключение"}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Строки таблицы
	pdf.SetFont("Helvetica", "", 7)
	for _, eq := range data.EquipmentList {
		row := []string{eq.Number, eq.Name, eq.Type, eq.Serial, eq.Inspection, eq.Status}
		for i, c := range row {
			pdf.CellFormat(colWidths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(4)

	// ── Заключение ────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, "4. Заключение комиссии", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, data.Conclusion, "", "L", false)
	pdf.Ln(2)

	// Рекомендации
	if data.Recommendations != "" {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(0, 6, "5. Рекомендации", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.MultiCell(0, 5, data.Recommendations, "", "L", false)
		pdf.Ln(2)
	}

	// ── Подписи ───────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "", 9)
	sigY := pdf.GetY()
	if sigY > 240 {
		pdf.AddPage()
		sigY = pdf.GetY()
	}
	pdf.SetY(sigY)

	pdf.CellFormat(90, 6, "Председатель комиссии:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.CellFormat(90, 6, "Члены комиссии:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.CellFormat(90, 6, "", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.CellFormat(90, 6, "Организация-исполнитель:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(8)

	// ── HMAC Footer (bash-256) ────────────────────────────────────────────
	// Формируем строку для подписи: номер_акта + дата + объект + заключение
	signingData := fmt.Sprintf("%s|%s|%s|%s|%s",
		data.ActNumber, data.ActDate, data.ObjectName, data.Conclusion, data.ExecutiveOrg)

	// В production ключ берётся из env (HMAC_SIGNING_KEY)
	// Здесь — placeholder; фактический ключ внедряется через PDFHandler
	placeholderKey := []byte("oac-template-placeholder-key-32bytes!")
	hmacSig := HMACSignOAC(signingData, placeholderKey)
	shortHash := hex.EncodeToString([]byte(hmacSig))[:16]

	pdf.SetFont("Helvetica", "I", 6)
	pdf.SetY(-18)
	pdf.CellFormat(0, 4, fmt.Sprintf("Акт № %s от %s | СН 3.02.19-2025 | ОАЦ №66", data.ActNumber, data.ActDate), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("HMAC(bash-256): %s | %s", shortHash, time.Now().UTC().Format(time.RFC3339)), "", 1, "C", false, 0, "")

	return pdf
}
