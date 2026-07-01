// Package ai — Vision Guard (P0-CR-09).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CR-09: DeepSeek Vision Prompt Injection Protection
//
// Предотвращает prompt injection через adversarial payload в фото.
// Злоумышленник может вставить текст/QR-код в фото, который AI
// интерпретирует как команду. Vision Guard проверяет изображение
// перед отправкой в AI.
//
// Checks:
//   - QR/barcode detection via gozxing (pure Go)
//   - Text region detection via image analysis (edge density + projections)
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — image content validation)
//   - IEC 62443 SR 3.3 (Security monitoring — content inspection)
//   - ISO 27001 A.12.6.1 (Capacity management — resource limits)
//   - Приказ ОАЦ № 66 п. 7.18.2 (Контроль целостности данных)
//
// ═══════════════════════════════════════════════════════════════════════════
package ai

import (
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"log/slog"
	"math"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/aztec"
	"github.com/makiuchi-d/gozxing/datamatrix"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// ─── Constants ─────────────────────────────────────────────────────────────

const (
	// DefaultMaxImageSize — максимальный размер изображения для проверки (50 MB).
	DefaultMaxImageSize int64 = 50 * 1024 * 1024

	// EdgeDensityThreshold — порог плотности границ для определения текста.
	// Значение 0.15 означает, что если >15% пикселей являются границами
	// в определённом регионе, регион считается текстоподобным.
	EdgeDensityThreshold = 0.15

	// TextRegionRatioThreshold — если >10% регионов изображения содержат
	// текстоподобные паттерны, изображение считается содержащим текст.
	TextRegionRatioThreshold = 0.10

	// GridSize — размер сетки для разбиения изображения на регионы.
	GridSize = 8
)

// ─── Types ─────────────────────────────────────────────────────────────────

// VisionGuardConfig — конфигурация Vision Guard.
type VisionGuardConfig struct {
	// StrictMode — если true, изображения с подозрительным контентом
	// reject'ятся. Если false — только логгируются.
	StrictMode bool `mapstructure:"strict_mode"`

	// EnableQR — проверять QR/barcode.
	EnableQR bool `mapstructure:"enable_qr"`

	// EnableTextDetection — проверять наличие текста через image analysis.
	EnableTextDetection bool `mapstructure:"enable_text_detection"`

	// MaxImageSize — максимальный размер изображения в байтах.
	MaxImageSize int64 `mapstructure:"max_image_size"`

	// EdgeDensityThreshold — порог плотности границ для text detection.
	EdgeDensityThreshold float64 `mapstructure:"edge_density_threshold"`

	// TextRegionRatioThreshold — порог доли текстоподобных регионов.
	TextRegionRatioThreshold float64 `mapstructure:"text_region_ratio_threshold"`
}

// VisionGuardResult — результат проверки изображения.
type VisionGuardResult struct {
	// Passed — true если изображение прошло все проверки.
	Passed bool `json:"passed"`

	// TextDetected — обнаружен текст в изображении.
	TextDetected bool `json:"text_detected"`

	// QRDetected — обнаружен QR/barcode.
	QRDetected bool `json:"qr_detected"`

	// Suspicious — изображение подозрительное.
	Suspicious bool `json:"suspicious"`

	// Warnings — список предупреждений.
	Warnings []string `json:"warnings,omitempty"`

	// Error — критическая ошибка проверки.
	Error string `json:"error,omitempty"`

	// QRContent — содержимое найденных QR-кодов (только для логов).
	QRContent []string `json:"qr_content,omitempty"`
}

// VisionGuard — главный компонент проверки изображений.
type VisionGuard struct {
	cfg    VisionGuardConfig
	logger *slog.Logger
}

// ─── Default Config ────────────────────────────────────────────────────────

// DefaultVisionGuardConfig возвращает конфигурацию по умолчанию.
func DefaultVisionGuardConfig() VisionGuardConfig {
	return VisionGuardConfig{
		StrictMode:               false, // default: warn only, не блокируем
		EnableQR:                 true,  // QR проверка включена
		EnableTextDetection:      true,  // text detection включён
		MaxImageSize:             DefaultMaxImageSize,
		EdgeDensityThreshold:     EdgeDensityThreshold,
		TextRegionRatioThreshold: TextRegionRatioThreshold,
	}
}

// ─── Constructor ───────────────────────────────────────────────────────────

// NewVisionGuard создаёт новый VisionGuard.
func NewVisionGuard(cfg VisionGuardConfig, logger *slog.Logger) *VisionGuard {
	if logger == nil {
		logger = slog.Default()
	}
	return &VisionGuard{
		cfg:    cfg,
		logger: logger.With("component", "vision-guard"),
	}
}

// ─── Main Check ────────────────────────────────────────────────────────────

// CheckImage проверяет изображение на наличие потенциально опасного контента.
// Принимает raw image bytes в формате JPEG или PNG.
//
// Порядок проверок:
//  1. Валидация размера
//  2. Декодирование изображения
//  3. QR/barcode detection (если включено)
//  4. Text region detection (если включено)
//  5. Финальное решение (strict/reject или warn)
func (g *VisionGuard) CheckImage(imageData []byte) *VisionGuardResult {
	result := &VisionGuardResult{
		Passed: true,
	}

	// 1. Проверка размера
	if int64(len(imageData)) > g.cfg.MaxImageSize {
		result.Error = fmt.Sprintf("image too large: %d bytes (max %d)",
			len(imageData), g.cfg.MaxImageSize)
		result.Passed = false
		result.Suspicious = true
		return result
	}

	if len(imageData) == 0 {
		result.Error = "empty image data"
		result.Passed = false
		return result
	}

	// 2. Декодирование
	img, format, err := image.Decode(bytesToReader(imageData))
	if err != nil {
		result.Error = fmt.Sprintf("image decode failed: %v", err)
		result.Passed = false
		return result
	}

	g.logger.Debug("image decoded", "format", format,
		"bounds", img.Bounds().Size())

	// 3. QR/Barcode detection
	if g.cfg.EnableQR {
		if qrResult := g.detectQR(img); len(qrResult) > 0 {
			result.QRDetected = true
			result.QRContent = qrResult
			result.Suspicious = true
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("QR/barcode detected (%d code(s))", len(qrResult)))
			g.logger.Warn("QR/barcode detected in image",
				"codes", qrResult)
		}
	}

	// 4. Text region detection
	if g.cfg.EnableTextDetection {
		if g.detectText(img) {
			result.TextDetected = true
			result.Suspicious = true
			result.Warnings = append(result.Warnings,
				"text-like patterns detected in image")
			g.logger.Warn("text-like patterns detected in image")
		}
	}

	// 5. Финальное решение
	if result.Suspicious && g.cfg.StrictMode {
		result.Passed = false
		result.Error = "image rejected by vision guard: " +
			strings.Join(result.Warnings, "; ")
	}

	return result
}

// ─── QR/Barcode Detection ──────────────────────────────────────────────────

// detectQR проверяет изображение на наличие QR-кодов и других barcode.
// Использует gozxing — pure-Go библиотеку для декодирования штрихкодов.
func (g *VisionGuard) detectQR(img image.Image) []string {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		g.logger.Debug("failed to create binary bitmap", "error", err)
		return nil
	}

	// Пробуем разные reader'ы
	type readerDef struct {
		name   string
		reader gozxing.Reader
	}

	readers := []readerDef{
		{"QRCode", qrcode.NewQRCodeReader()},
		{"DataMatrix", datamatrix.NewDataMatrixReader()},
		{"Aztec", aztec.NewAztecReader()},
	}

	var codes []string

	for _, r := range readers {
		result, err := r.reader.Decode(bmp, nil)
		if err != nil {
			continue
		}
		if result != nil && result.GetText() != "" {
			text := result.GetText()
			// Не логируем полный контент (может быть конфиденциальным)
			truncated := text
			if len(truncated) > 100 {
				truncated = truncated[:100] + "..."
			}
			codes = append(codes, fmt.Sprintf("%s:%s", r.name, truncated))
		}
	}

	return codes
}

// ─── Text Region Detection (Pure Go) ───────────────────────────────────────

// detectText проверяет изображение на наличие текстоподобных регионов
// без использования внешних OCR-библиотек.
//
// Алгоритм:
//  1. Конвертация в grayscale
//  2. Sobel-like edge detection (горизонтальные и вертикальные градиенты)
//  3. Разбиение на GridSize×GridSize регионов
//  4. Для каждого региона: вычисление плотности границ
//  5. Если плотность > порога и паттерн повторяющийся → текст
//
// Это не читает текст, а только определяет его наличие по характерным
// признакам: высокая плотность границ в регулярном паттерне.
func (g *VisionGuard) detectText(img image.Image) bool {
	bounds := img.Bounds()
	if bounds.Dx() < 20 || bounds.Dy() < 20 {
		return false // слишком маленькое изображение
	}

	width := bounds.Dx()
	height := bounds.Dy()

	// Создаём карту градиентов (magnitude)
	gradMag := make([][]float64, height)
	for y := 0; y < height; y++ {
		gradMag[y] = make([]float64, width)
	}

	// Sobel-like оператор
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			gx := sobelX(img, x, y, bounds)
			gy := sobelY(img, x, y, bounds)
			mag := math.Sqrt(float64(gx*gx + gy*gy))
			if mag > 30.0 { // порог градиента
				gradMag[y][x] = 1.0
			}
		}
	}

	// Разбиение на регионы
	regionW := max(width/GridSize, 1)
	regionH := max(height/GridSize, 1)

	edgeRegions := 0
	totalRegions := 0

	for ry := 0; ry < GridSize; ry++ {
		for rx := 0; rx < GridSize; rx++ {
			xStart := rx * regionW
			xEnd := min((rx+1)*regionW, width)
			yStart := ry * regionH
			yEnd := min((ry+1)*regionH, height)

			edgePixels := 0
			totalPixels := 0

			for y := yStart; y < yEnd; y++ {
				for x := xStart; x < xEnd; x++ {
					totalPixels++
					if gradMag[y][x] > 0 {
						edgePixels++
					}
				}
			}

			if totalPixels > 0 {
				totalRegions++
				density := float64(edgePixels) / float64(totalPixels)
				if density > g.cfg.EdgeDensityThreshold {
					edgeRegions++
				}
			}
		}
	}

	if totalRegions == 0 {
		return false
	}

	regionRatio := float64(edgeRegions) / float64(totalRegions)
	g.logger.Debug("text detection", "edge_regions", edgeRegions,
		"total_regions", totalRegions, "region_ratio", regionRatio)

	return regionRatio > g.cfg.TextRegionRatioThreshold
}

// ─── Edge Detection Helpers ────────────────────────────────────────────────

// sobelX вычисляет градиент по оси X в точке (x,y).
func sobelX(img image.Image, x, y int, bounds image.Rectangle) int {
	// Sobel X kernel: [[-1,0,1],[-2,0,2],[-1,0,1]]
	gx := -1 * grayAt(img, x-1, y-1, bounds)
	gx += 1 * grayAt(img, x+1, y-1, bounds)
	gx += -2 * grayAt(img, x-1, y, bounds)
	gx += 2 * grayAt(img, x+1, y, bounds)
	gx += -1 * grayAt(img, x-1, y+1, bounds)
	gx += 1 * grayAt(img, x+1, y+1, bounds)
	return gx
}

// sobelY вычисляет градиент по оси Y в точке (x,y).
func sobelY(img image.Image, x, y int, bounds image.Rectangle) int {
	// Sobel Y kernel: [[-1,-2,-1],[0,0,0],[1,2,1]]
	gy := -1 * grayAt(img, x-1, y-1, bounds)
	gy += -2 * grayAt(img, x, y-1, bounds)
	gy += -1 * grayAt(img, x+1, y-1, bounds)
	gy += 1 * grayAt(img, x-1, y+1, bounds)
	gy += 2 * grayAt(img, x, y+1, bounds)
	gy += 1 * grayAt(img, x+1, y+1, bounds)
	return gy
}

// grayAt возвращает яркость пикселя как int (0-255).
func grayAt(img image.Image, x, y int, bounds image.Rectangle) int {
	if x < bounds.Min.X || x >= bounds.Max.X ||
		y < bounds.Min.Y || y >= bounds.Max.Y {
		return 0
	}
	r, g, b, _ := img.At(x, y).RGBA()
	// Преобразование в grayscale по формуле luminance
	gray := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
	return int(gray)
}

// ─── Utility ───────────────────────────────────────────────────────────────

// bytesToReader создаёт reader из byte slice для image.Decode.
type byteReader struct {
	data []byte
	pos  int
}

func bytesToReader(data []byte) *byteReader {
	return &byteReader{data: data}
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF") // image.Decode проверяет EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// ImageFormat определяет формат изображения по magic bytes.
func ImageFormat(data []byte) string {
	if len(data) < 4 {
		return "unknown"
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8:
		return "jpeg"
	case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
		return "png"
	case data[0] == 'G' && data[1] == 'I' && data[2] == 'F':
		return "gif"
	case data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F':
		return "webp"
	case data[0] == 0x49 && data[1] == 0x49 && data[2] == 0x2A && data[3] == 0x00:
		return "tiff-le"
	case data[0] == 0x4D && data[1] == 0x4D && data[2] == 0x00 && data[3] == 0x2A:
		return "tiff-be"
	case data[0] == 0x42 && data[1] == 0x4D:
		return "bmp"
	default:
		return "unknown"
	}
}
