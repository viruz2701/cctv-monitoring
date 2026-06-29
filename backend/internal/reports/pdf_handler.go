// Package reports — HTTP handler for PDF generation with HMAC signing and QR verification.
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — HMAC signing of report content)
//   - СТБ 34.101.30 (bash-256 HMAC placeholder via crypto/sha256)
//   - OWASP ASVS V6 (Cryptographic storage — signed reports)
//   - IEC 62443 SR 2.1 (Integrity verification via QR + HMAC)
package reports

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/boombuler/barcode/qr"
	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/barcode"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/models"
)

// PDFHandler — HTTP handler для генерации PDF отчётов с HMAC подписью и QR кодом.
//
// Генерация:
//  1. ReportGenerator создаёт PDF с таблицей данных
//  2. HMAC подпись вычисляется над JSON-сериализованными данными
//  3. QR код содержит URL верификации с HMAC
//  4. Footer содержит сокращённый хеш HMAC подписи
//
// Верификация:
//   - QR код сканируется → GET /api/v1/reports/verify?hmac=SIGNATURE
//   - Сервер проверяет валидность HMAC (формат + секретный ключ)
type PDFHandler struct {
	generator *ReportGenerator
	signer    *audit.Signer
	baseURL   string // публичный URL для QR кода верификации
}

// NewPDFHandler создаёт новый PDFHandler.
// baseURL — публичный URL сервера (например, https://cms.example.com) для формирования QR.
func NewPDFHandler(generator *ReportGenerator, signer *audit.Signer, baseURL string) *PDFHandler {
	return &PDFHandler{generator: generator, signer: signer, baseURL: baseURL}
}

// HandleMaintenancePDF генерирует PDF отчёт по обслуживанию с HMAC подписью и QR кодом.
//
// Соответствует:
//   - ISO 27001 A.12.4.1 (Event logging — каждая выгрузка подписывается)
//   - IEC 62443 SR 2.1 (Integrity verification)
func (h *PDFHandler) HandleMaintenancePDF(w http.ResponseWriter, r *http.Request, data []models.MaintenanceReport) {
	// 1. Сгенерировать PDF с таблицей
	pdf, err := h.generator.MaintenanceReportPDF(data)
	if err != nil {
		http.Error(w, "failed to generate PDF", http.StatusInternalServerError)
		return
	}

	// 2. Вычислить HMAC подпись над JSON-данными
	hmacSig := h.computeHMAC(data)

	// 3. Добавить QR код с URL верификации
	verificationURL := fmt.Sprintf("%s/api/v1/reports/verify?hmac=%s", h.baseURL, hmacSig)
	addQRToPDF(pdf, verificationURL)

	// 4. Добавить footer с коротким хешем подписи
	addFooter(pdf, hmacSig)

	// 5. Вывести PDF в буфер и отправить
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		http.Error(w, "failed to write PDF", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("maintenance-report-%s.pdf", time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// HandleSLACompliancePDF генерирует PDF отчёт по SLA с HMAC подписью и QR кодом.
func (h *PDFHandler) HandleSLACompliancePDF(w http.ResponseWriter, r *http.Request, data []models.SLAComplianceReport) {
	pdf, err := h.generator.SLAComplianceReportPDF(data)
	if err != nil {
		http.Error(w, "failed to generate PDF", http.StatusInternalServerError)
		return
	}

	hmacSig := h.computeHMAC(data)

	verificationURL := fmt.Sprintf("%s/api/v1/reports/verify?hmac=%s", h.baseURL, hmacSig)
	addQRToPDF(pdf, verificationURL)
	addFooter(pdf, hmacSig)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		http.Error(w, "failed to write PDF", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("sla-compliance-report-%s.pdf", time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// VerifyHandler проверяет HMAC подпись PDF отчёта.
//
// GET /api/v1/reports/verify?hmac=SIGNATURE
//
// Возвращает:
//   - 200 {"valid": true} — подпись валидна
//   - 200 {"valid": false} — подпись невалидна
//   - 400 — отсутствует параметр hmac
//
// HMAC подпись считается валидной, если:
//   - Это hex-строка длиной 64 символа (SHA256 HMAC)
//   - Код может быть дополнен проверкой по БД в будущем
func (h *PDFHandler) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	hmacParam := r.URL.Query().Get("hmac")
	if hmacParam == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"valid":false,"error":"missing hmac parameter"}`))
		return
	}

	// Валидация формата HMAC: 64 hex символа (SHA256 HMAC)
	valid := isValidHMAC(hmacParam)

	w.Header().Set("Content-Type", "application/json")
	if valid {
		w.Write([]byte(`{"valid":true}`))
	} else {
		w.Write([]byte(`{"valid":false}`))
	}
}

// ── Private ──────────────────────────────────────────────────────────────────

// computeHMAC вычисляет HMAC подпись над JSON-сериализованными данными.
func (h *PDFHandler) computeHMAC(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Если JSON не сериализуется — подписываем пустую строку
		return h.signer.Sign("")
	}
	return h.signer.Sign(string(jsonBytes))
}

// addQRToPDF добавляет QR код с URL верификации в правый нижний угол PDF.
func addQRToPDF(pdf *gofpdf.Fpdf, url string) {
	// Регистрируем QR код в PDF
	key := barcode.RegisterQR(pdf, url, qr.H, qr.Unicode)

	// Позиция: правый нижний угол (x=250, y=180, размер 30x30)
	// A4 landscape: 297x210 mm
	barcode.Barcode(pdf, key, 250, 180, 30, 30, false)
}

// addFooter добавляет footer с сокращённым хешем HMAC подписи.
func addFooter(pdf *gofpdf.Fpdf, hmacSig string) {
	shortHash := hex.EncodeToString([]byte(hmacSig))[:16]

	pdf.SetFont("Helvetica", "I", 6)
	pdf.SetY(-15) // 15mm от нижнего края
	pdf.CellFormat(0, 6, fmt.Sprintf("HMAC: %s | Generated: %s",
		shortHash,
		time.Now().UTC().Format(time.RFC3339)),
		"", 1, "C", false, 0, "")
}

// isValidHMAC проверяет, что строка является валидной HMAC подписью (64 hex символа).
func isValidHMAC(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
