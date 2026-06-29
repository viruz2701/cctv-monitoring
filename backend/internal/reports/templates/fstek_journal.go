// Package templates — региональные шаблоны PDF для compliance отчётов CCTV Health Monitor.
//
// ═══════════════════════════════════════════════════════════════════════════════
// Compliance:
//   - 149-ФЗ «Об информации, информационных технологиях и о защите информации»
//   - Приказ ФСТЭК России № 17 от 11.02.2013 (Защита КИИ)
//   - 152-ФЗ «О персональных данных» (ПДн)
//   - ISO 27001 A.12.4 (Audit trail — журнал проверок)
//   - IEC 62443 SR 2.8 (Audit events)
//   - OWASP ASVS V6 (Cryptographic storage — HMAC footer)
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

// FSTekJournalData — данные для заполнения журнала проверок КИИ по форме ФСТЭК РФ.
// Соответствие: 149-ФЗ ст. 11, Приказ ФСТЭК № 17 п. 15-18.
type FSTekJournalData struct {
	OrganizationName string               // Наименование организации
	KIIObjectName    string               // Наименование объекта КИИ
	KIICategory      string               // Категория объекта КИИ (1-3)
	Records          []FSTekJournalRecord // Записи журнала
}

// FSTekJournalRecord — одна запись в журнале проверок.
type FSTekJournalRecord struct {
	Date            string // Дата проверки (ДД.ММ.ГГГГ)
	InspectorName   string // ФИО проверяющего
	ViolationType   string // Тип нарушения / предмет проверки
	Deadline        string // Срок устранения (ДД.ММ.ГГГГ)
	RemediationNote string // Отметка об устранении
	RemediationDate string // Дата устранения (ДД.ММ.ГГГГ)
}

// FSTekJournal генерирует журнал проверок КИИ по форме ФСТЭК РФ.
//
// Формат: A4 landscape (для таблицы)
// Содержит:
//   - Шапка с реквизитами организации и объекта КИИ
//   - Ссылка на 149-ФЗ и Приказ ФСТЭК № 17
//   - Таблица проверок (дата, проверяющий, нарушение, срок, отметка)
//   - HMAC footer для целостности
//
// Ссылки: 149-ФЗ ст. 11 (КИИ), Приказ ФСТЭК № 17 п. 15-18 (журнал проверок)
func FSTekJournal(data FSTekJournalData) *gofpdf.Fpdf {
	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape для широкой таблицы
	pdf.SetMargins(10, 15, 10)
	pdf.SetAutoPageBreak(true, 25)
	pdf.AddPage()

	// ── Шапка ─────────────────────────────────────────────────────────────
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "ЖУРНАЛ ПРОВЕРОК", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 7, "состояния информационной безопасности объекта КИИ", "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Реквизиты
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(50, 5, "Организация:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 5, data.OrganizationName, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(50, 5, "Объект КИИ:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 5, data.KIIObjectName, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(50, 5, "Категория КИИ:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 5, fmt.Sprintf("Категория %s", data.KIICategory), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Нормативная ссылка
	pdf.SetFont("Helvetica", "I", 7)
	pdf.MultiCell(0, 4,
		"Основание: 149-ФЗ «Об информации, информационных технологиях и о защите информации» (ст. 11 - КИИ); "+
			"Приказ ФСТЭК России № 17 от 11.02.2013 «Об утверждении требований к защите информации, "+
			"содержащейся в информационных системах общего пользования»; "+
			"152-ФЗ «О персональных данных».",
		"", "L", false)
	pdf.Ln(4)

	// ── Таблица проверок ──────────────────────────────────────────────────
	// Формат: Дата | Проверяющий | Тип нарушения | Срок устранения | Отметка об устранении | Дата устранения
	colWidths := []float64{22, 38, 80, 30, 72, 30}
	headers := []string{"Дата", "Проверяющий", "Тип нарушения / предмет проверки", "Срок устранения", "Отметка об устранении", "Дата устранения"}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(226, 232, 240)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Строки таблицы
	pdf.SetFont("Helvetica", "", 7)
	if len(data.Records) == 0 {
		// Пустая строка для незаполненного журнала
		emptyRow := []string{"", "", "Записей нет", "", "", ""}
		for i, c := range emptyRow {
			pdf.CellFormat(colWidths[i], 6, c, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)
	} else {
		for _, rec := range data.Records {
			row := []string{
				rec.Date,
				rec.InspectorName,
				rec.ViolationType,
				rec.Deadline,
				rec.RemediationNote,
				rec.RemediationDate,
			}
			for i, c := range row {
				pdf.CellFormat(colWidths[i], 6, c, "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
	}
	pdf.Ln(4)

	// ── Подпись ───────────────────────────────────────────────────────────
	sigY := pdf.GetY()
	if sigY > 170 {
		pdf.AddPage()
		sigY = pdf.GetY()
	}
	pdf.SetY(sigY)

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(90, 6, "Ответственный за безопасность КИИ:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.CellFormat(90, 6, "Руководитель организации:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "___________________ /____________________/", "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.CellFormat(90, 6, "", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, "М.П.", "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// ── HMAC Footer ───────────────────────────────────────────────────────
	// Формируем строку для подписи (количество записей + организация + объект)
	recordCount := len(data.Records)
	signingData := fmt.Sprintf("%s|%s|%s|%d|journal",
		data.OrganizationName, data.KIIObjectName, data.KIICategory, recordCount)

	placeholderKey := []byte("fstek-journal-placeholder-key-32bytes!")
	mac := hmac.New(sha256.New, placeholderKey)
	mac.Write([]byte(signingData))
	hmacSig := hex.EncodeToString(mac.Sum(nil))
	shortHash := hex.EncodeToString([]byte(hmacSig))[:16]

	pdf.SetFont("Helvetica", "I", 6)
	pdf.SetY(-18)
	pdf.CellFormat(0, 4, fmt.Sprintf("149-ФЗ | Приказ ФСТЭК №17 | Записей: %d", recordCount), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("HMAC: %s | %s", shortHash, time.Now().UTC().Format(time.RFC3339)), "", 1, "C", false, 0, "")

	return pdf
}
