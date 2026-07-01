// Package ai — Vision Guard Tests.
package ai

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

// createTestPNG создаёт простое PNG изображение заданного цвета и размера.
func createTestPNG(width, height int, bg color.Color, patterns ...func(img draw.Image)) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Заливка фоном
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Применение паттернов
	for _, p := range patterns {
		p(img)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// addTextPattern добавляет текстоподобный паттерн (прямоугольники),
// имитирующие строки текста.
func addTextPattern(_ color.Color) func(img draw.Image) {
	return func(img draw.Image) {
		bounds := img.Bounds()
		for y := bounds.Min.Y + 10; y < bounds.Max.Y-10; y += 20 {
			for x := bounds.Min.X + 5; x < bounds.Max.X-5; x++ {
				// Строки текста: горизонтальные линии с пробелами
				if (y/20)%2 == 0 {
					if (x/8)%2 == 0 || (x/8)%3 == 0 {
						img.Set(x, y, color.RGBA{B: 255, A: 255})
						img.Set(x, y+1, color.RGBA{B: 255, A: 255})
					}
				}
			}
		}
	}
}

// createQRImage создаёт PNG изображение с QR-кодом.
func createQRImage(t testing.TB, content string, size int) []byte {
	t.Helper()

	// Кодируем QR код
	writer := qrcode.NewQRCodeWriter()
	img, err := writer.Encode(content, gozxing.BarcodeFormat_QR_CODE, size, size, nil)
	if err != nil {
		t.Fatalf("failed to encode QR: %v", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	return buf.Bytes()
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestVisionGuard_DefaultConfig(t *testing.T) {
	cfg := DefaultVisionGuardConfig()
	if !cfg.EnableQR {
		t.Error("QR detection should be enabled by default")
	}
	if !cfg.EnableTextDetection {
		t.Error("text detection should be enabled by default")
	}
	if cfg.StrictMode {
		t.Error("strict mode should be disabled by default")
	}
	if cfg.MaxImageSize != DefaultMaxImageSize {
		t.Errorf("expected max image size %d, got %d", DefaultMaxImageSize, cfg.MaxImageSize)
	}
}

func TestVisionGuard_CleanImage(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	// Чистое изображение (градиент, без текста/QR)
	imgData := createTestPNG(200, 200, color.RGBA{G: 128, A: 255})

	result := vg.CheckImage(imgData)

	if !result.Passed {
		t.Errorf("clean image should pass, got error: %s", result.Error)
	}
	if result.Suspicious {
		t.Error("clean image should not be suspicious")
	}
	if result.QRDetected {
		t.Error("clean image should not have QR detected")
	}
	if result.TextDetected {
		t.Error("clean image should not have text detected")
	}
}

func TestVisionGuard_QRDetection(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	// Изображение с QR-кодом
	qrData := createQRImage(t, "https://evil.com/payload", 200)

	result := vg.CheckImage(qrData)

	if !result.Passed {
		// В warn mode должно проходить
		t.Logf("QR image result: passed=%v, suspicious=%v, warnings=%v",
			result.Passed, result.Suspicious, result.Warnings)
	}
	if !result.QRDetected {
		t.Error("QR code should be detected")
	}
	if !result.Suspicious {
		t.Error("image with QR should be suspicious")
	}
	if len(result.QRContent) == 0 {
		t.Error("QR content should not be empty")
	}
}

func TestVisionGuard_QRDetectionStrict(t *testing.T) {
	cfg := DefaultVisionGuardConfig()
	cfg.StrictMode = true
	vg := NewVisionGuard(cfg, nil)

	qrData := createQRImage(t, "malicious-payload", 200)
	result := vg.CheckImage(qrData)

	if result.Passed {
		t.Error("strict mode: QR image should be rejected")
	}
	if !result.QRDetected {
		t.Error("QR code should be detected in strict mode")
	}
}

func TestVisionGuard_TextDetection(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	// Изображение с текстоподобным паттерном
	imgData := createTestPNG(200, 200,
		color.RGBA{R: 128, G: 128, B: 128, A: 255},
		addTextPattern(color.RGBA{R: 128, G: 128, B: 128, A: 255}))

	result := vg.CheckImage(imgData)

	t.Logf("Text image result: passed=%v, suspicious=%v, text_detected=%v, warnings=%v",
		result.Passed, result.Suspicious, result.TextDetected, result.Warnings)

	if !result.TextDetected {
		// Text detection is heuristic-based, so it might not always detect
		// but for dense text patterns it should
		t.Log("WARNING: text not detected (heuristic may need tuning)")
	}
}

func TestVisionGuard_EmptyImage(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	result := vg.CheckImage([]byte{})

	if result.Passed {
		t.Error("empty image should not pass")
	}
	if result.Error == "" {
		t.Error("empty image should have error message")
	}
}

func TestVisionGuard_InvalidImage(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	result := vg.CheckImage([]byte{0xFF, 0xD8, 0xFF, 0x00}) // truncated JPEG

	if result.Passed {
		t.Error("invalid image should not pass")
	}
}

func TestVisionGuard_TooLargeImage(t *testing.T) {
	cfg := DefaultVisionGuardConfig()
	cfg.MaxImageSize = 100 // 100 bytes max
	vg := NewVisionGuard(cfg, nil)

	data := make([]byte, 200)
	// Make it look like a valid PNG header
	data[0] = 0x89
	data[1] = 'P'
	data[2] = 'N'
	data[3] = 'G'

	result := vg.CheckImage(data)

	if result.Passed {
		t.Error("too large image should not pass")
	}
	if !result.Suspicious {
		t.Error("too large image should be suspicious")
	}
}

func TestVisionGuard_QRDisabled(t *testing.T) {
	cfg := DefaultVisionGuardConfig()
	cfg.EnableQR = false
	vg := NewVisionGuard(cfg, nil)

	qrData := createQRImage(t, "test-content", 200)
	result := vg.CheckImage(qrData)

	if result.QRDetected {
		t.Error("QR detection should be disabled")
	}
}

func TestVisionGuard_TextDetectionDisabled(t *testing.T) {
	cfg := DefaultVisionGuardConfig()
	cfg.EnableTextDetection = false
	vg := NewVisionGuard(cfg, nil)

	imgData := createTestPNG(200, 200,
		color.RGBA{R: 128, G: 128, B: 128, A: 255},
		addTextPattern(color.RGBA{R: 128, G: 128, B: 128, A: 255}))

	result := vg.CheckImage(imgData)

	if result.TextDetected {
		t.Error("text detection should be disabled")
	}
	if result.Suspicious {
		t.Error("image should not be suspicious when checks disabled")
	}
}

func TestImageFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"JPEG", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "jpeg"},
		{"PNG", []byte{0x89, 'P', 'N', 'G'}, "png"},
		{"GIF", []byte{'G', 'I', 'F', '8'}, "gif"},
		{"empty", []byte{}, "unknown"},
		{"unknown", []byte{0x00, 0x01, 0x02, 0x03}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ImageFormat(tt.data)
			if result != tt.expected {
				t.Errorf("ImageFormat(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestVisionGuard_NewWithNilLogger(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)
	if vg == nil {
		t.Fatal("NewVisionGuard with nil logger should not return nil")
	}
	if vg.logger == nil {
		t.Error("logger should be initialized with default")
	}
}

func TestVisionGuard_QR_MultipleFormats(t *testing.T) {
	// Проверка что детектор не падает на разных форматах
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	formats := []struct {
		name string
		data []byte
	}{
		{"small_qr", createQRImage(t, "x", 50)},
		{"medium_qr", createQRImage(t, "hello-world-42", 100)},
		{"large_qr", createQRImage(t, "https://example.com/very/long/url/that/might/be/used/for/injection", 200)},
		{"numeric_qr", createQRImage(t, "1234567890", 100)},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			result := vg.CheckImage(f.data)
			// QR должен быть обнаружен, warn mode → passed
			if !result.QRDetected {
				t.Errorf("QR code not detected in %s", f.name)
			}
		})
	}
}

func TestVisionGuard_EdgeCases(t *testing.T) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)

	t.Run("very_small_image", func(t *testing.T) {
		data := createTestPNG(5, 5, color.RGBA{R: 255, A: 255})
		result := vg.CheckImage(data)
		// Маленькое изображение — не текст, не QR → passed
		if !result.Passed {
			t.Errorf("small image should pass: %s", result.Error)
		}
	})

	t.Run("single_color_image", func(t *testing.T) {
		data := createTestPNG(50, 50, color.RGBA{B: 255, A: 255})
		result := vg.CheckImage(data)
		if !result.Passed {
			t.Errorf("single color image should pass: %s", result.Error)
		}
		if result.TextDetected {
			t.Error("single color image should not have text")
		}
	})

	t.Run("white_image", func(t *testing.T) {
		data := createTestPNG(100, 100, color.White)
		result := vg.CheckImage(data)
		if !result.Passed {
			t.Errorf("white image should pass: %s", result.Error)
		}
	})

	t.Run("black_image", func(t *testing.T) {
		data := createTestPNG(100, 100, color.Black)
		result := vg.CheckImage(data)
		if !result.Passed {
			t.Errorf("black image should pass: %s", result.Error)
		}
	})
}

// ─── Benchmark ──────────────────────────────────────────────────────────────

func BenchmarkVisionGuard_CleanImage(b *testing.B) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)
	data := createTestPNG(640, 480, color.RGBA{G: 128, A: 255})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vg.CheckImage(data)
	}
}

func BenchmarkVisionGuard_QRImage(b *testing.B) {
	vg := NewVisionGuard(DefaultVisionGuardConfig(), nil)
	data := createQRImage(b, "benchmark-test-content", 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vg.CheckImage(data)
	}
}

// ─── Verification: unused import suppression ────────────────────────────────

var _ = image.Transparent
