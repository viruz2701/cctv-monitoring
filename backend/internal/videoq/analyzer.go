// Package videoq — Video Quality Metrics для CCTV изображений.
//
// CCTV-2.1.1: Анализ качества изображений без видеопотоков.
// Работает с отдельными кадрами (скриншотами, snapshot'ами с камер).
//
// Метрики:
//   - Blur: размытие изображения (Laplacian variance)
//   - Brightness: средняя яркость
//   - Contrast: контрастность (std dev)
//   - BlackScreen: процент чёрных пикселей
//   - FrozenImage: сравнение с предыдущим кадром
//   - NoiseLevel: уровень шума
//   - Blockiness: артефакты сжатия (block boundary)
//
// Compliance:
//   - CCTV Core IP (уникальная фича)
//   - Apache 2.0: все зависимости permissive
package videoq

import (
	"fmt"
	"image"
	"math"
)

// ═══════════════════════════════════════════════════════════════════════
// QualityMetrics — результаты анализа качества изображения.
// ═══════════════════════════════════════════════════════════════════════

type QualityMetrics struct {
	// Blur detection (Laplacian variance)
	// < 100 = blurry, 100-200 = normal, > 200 = sharp
	BlurScore    float64 `json:"blur_score"`
	IsBlurry     bool    `json:"is_blurry"`

	// Brightness (0-255)
	// < 40 = too dark, 40-200 = normal, > 200 = too bright
	Brightness   float64 `json:"brightness"`
	IsTooDark    bool    `json:"is_too_dark"`
	IsTooBright  bool    `json:"is_too_bright"`

	// Contrast (std dev of pixel values)
	// < 30 = low contrast (washed out), 30-80 = normal, > 80 = high contrast
	Contrast     float64 `json:"contrast"`
	IsLowContrast bool   `json:"is_low_contrast"`

	// Black screen detection
	// > 80% black pixels = black screen
	BlackPixelPct float64 `json:"black_pixel_percent"`
	IsBlackScreen bool    `json:"is_black_screen"`

	// Frozen image detection (requires previous frame)
	// < 95% similarity = not frozen
	FrozenSimilarity *float64 `json:"frozen_similarity,omitempty"`
	IsFrozen         *bool    `json:"is_frozen,omitempty"`

	// Noise level (std dev of flat regions)
	// > 10 = noisy
	NoiseLevel   float64 `json:"noise_level"`
	IsNoisy      bool    `json:"is_noisy"`

	// Blockiness (compression artifacts)
	// > 15 = blocky
	Blockiness   float64 `json:"blockiness"`
	IsBlocky     bool    `json:"is_blocky"`

	// Overall score (0-100)
	OverallScore float64 `json:"overall_score"`
	Status       string  `json:"status"` // "good", "degraded", "poor"
}

// ═══════════════════════════════════════════════════════════════════════
// Analyzer
// ═══════════════════════════════════════════════════════════════════════

// Analyzer анализирует качество изображений CCTV.
type Analyzer struct {
	prevFrame  *image.Gray  // предыдущий кадр (для frozen detection)
}

// AnalyzerConfig — конфигурация порогов для анализа.
type AnalyzerConfig struct {
	BlurThreshold      float64 // default: 100
	DarkThreshold      float64 // default: 40
	BrightThreshold    float64 // default: 200
	ContrastThreshold  float64 // default: 30
	BlackScreenPct     float64 // default: 80
	FrozenSimilarity   float64 // default: 0.95
	NoiseThreshold     float64 // default: 10
	BlockinessThreshold float64 // default: 15
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() AnalyzerConfig {
	return AnalyzerConfig{
		BlurThreshold:      100.0,
		DarkThreshold:      40.0,
		BrightThreshold:    200.0,
		ContrastThreshold:  30.0,
		BlackScreenPct:     80.0,
		FrozenSimilarity:   0.95,
		NoiseThreshold:     10.0,
		BlockinessThreshold: 15.0,
	}
}

// NewAnalyzer создаёт Analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze анализирует одно изображение и возвращает метрики качества.
func (a *Analyzer) Analyze(img image.Image) (*QualityMetrics, error) {
	gray := toGrayscale(img)
	metrics := &QualityMetrics{}

	// 1. Blur detection
	metrics.BlurScore = detectBlur(gray)
	metrics.IsBlurry = metrics.BlurScore < DefaultConfig().BlurThreshold

	// 2. Brightness
	metrics.Brightness = calculateBrightness(gray)
	metrics.IsTooDark = metrics.Brightness < DefaultConfig().DarkThreshold
	metrics.IsTooBright = metrics.Brightness > DefaultConfig().BrightThreshold

	// 3. Contrast
	metrics.Contrast = calculateContrast(gray)
	metrics.IsLowContrast = metrics.Contrast < DefaultConfig().ContrastThreshold

	// 4. Black screen
	metrics.BlackPixelPct = detectBlackScreen(gray)
	metrics.IsBlackScreen = metrics.BlackPixelPct > DefaultConfig().BlackScreenPct

	// 5. Frozen image (сравнение с предыдущим кадром)
	if a.prevFrame != nil {
		similarity := compareFrames(a.prevFrame, gray)
		metrics.FrozenSimilarity = &similarity
		isFrozen := similarity > DefaultConfig().FrozenSimilarity
		metrics.IsFrozen = &isFrozen
	}

	// 6. Noise level
	metrics.NoiseLevel = estimateNoise(gray)
	metrics.IsNoisy = metrics.NoiseLevel > DefaultConfig().NoiseThreshold

	// 7. Blockiness
	metrics.Blockiness = detectBlockiness(gray)
	metrics.IsBlocky = metrics.Blockiness > DefaultConfig().BlockinessThreshold

	// Overall score
	metrics.OverallScore = calculateOverall(metrics)
	metrics.Status = classifyQuality(metrics.OverallScore)

	// Сохраняем для следующего сравнения
	a.prevFrame = gray

	return metrics, nil
}

// Reset сбрасывает сохранённый предыдущий кадр.
func (a *Analyzer) Reset() {
	a.prevFrame = nil
}

// ═══════════════════════════════════════════════════════════════════════
// Metric calculations
// ═══════════════════════════════════════════════════════════════════════

// detectBlur использует дисперсию Лапласиана для оценки размытия.
// Чем выше значение — тем резче изображение.
func detectBlur(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width < 3 || height < 3 {
		return 0
	}

	// Применяем ядро Лапласиана: [[0, -1, 0], [-1, 4, -1], [0, -1, 0]]
	var sum, count float64

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			center := float64(gray.GrayAt(x, y).Y)
			top := float64(gray.GrayAt(x, y-1).Y)
			bottom := float64(gray.GrayAt(x, y+1).Y)
			left := float64(gray.GrayAt(x-1, y).Y)
			right := float64(gray.GrayAt(x+1, y).Y)

			laplacian := math.Abs(4*center - top - bottom - left - right)
			sum += laplacian
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / count
}

// calculateBrightness вычисляет среднюю яркость (0-255).
func calculateBrightness(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	var sum float64
	count := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sum += float64(gray.GrayAt(x, y).Y)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// calculateContrast вычисляет контрастность (std dev пикселей).
func calculateContrast(gray *image.Gray) float64 {
	mean := calculateBrightness(gray)
	bounds := gray.Bounds()
	var varianceSum float64
	count := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			diff := float64(gray.GrayAt(x, y).Y) - mean
			varianceSum += diff * diff
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return math.Sqrt(varianceSum / float64(count))
}

// detectBlackScreen определяет процент чёрных пикселей (< 30 яркость).
func detectBlackScreen(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	blackPixels := 0
	totalPixels := 0

	const darkThreshold = 30.0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if float64(gray.GrayAt(x, y).Y) < darkThreshold {
				blackPixels++
			}
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0
	}
	return float64(blackPixels) / float64(totalPixels) * 100
}

// compareFrames сравнивает два кадра для детекции frozen image.
// Возвращает коэффициент схожести (0-1).
func compareFrames(prev, curr *image.Gray) float64 {
	bounds := prev.Bounds()
	if bounds != curr.Bounds() {
		return 0
	}

	matchingPixels := 0
	totalPixels := 0
	const threshold = 10.0 // допуск 10/255

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			diff := math.Abs(float64(prev.GrayAt(x, y).Y) - float64(curr.GrayAt(x, y).Y))
			if diff < threshold {
				matchingPixels++
			}
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0
	}
	return float64(matchingPixels) / float64(totalPixels)
}

// estimateNoise оценивает уровень шума через std dev в плоских регионах.
func estimateNoise(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width < 10 || height < 10 {
		return 0
	}

	// Берём центральный регион 50x50 для оценки шума
	startX := width / 2 - 25
	startY := height / 2 - 25
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}
	endX := startX + 50
	endY := startY + 50
	if endX > width {
		endX = width
	}
	if endY > height {
		endY = height
	}

	var values []float64
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			values = append(values, float64(gray.GrayAt(x, y).Y))
		}
	}

	if len(values) == 0 {
		return 0
	}

	// Среднее
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Std dev
	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}

	return math.Sqrt(varianceSum / float64(len(values)))
}

// detectBlockiness детектирует артефакты сжатия (JPEG block boundaries).
func detectBlockiness(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width < 16 || height < 16 {
		return 0
	}

	// Анализируем разницу на границах блоков 8x8 (JPEG macroblocks)
	var blockDifference float64
	var count float64

	// Вертикальные границы блоков
	for by := 8; by < height-8; by += 8 {
		for x := 0; x < width; x++ {
			diff := math.Abs(float64(gray.GrayAt(x, by).Y) - float64(gray.GrayAt(x, by-1).Y))
			blockDifference += diff
			count++
		}
	}

	// Горизонтальные границы блоков
	for bx := 8; bx < width-8; bx += 8 {
		for y := 0; y < height; y++ {
			diff := math.Abs(float64(gray.GrayAt(bx, y).Y) - float64(gray.GrayAt(bx-1, y).Y))
			blockDifference += diff
			count++
		}
	}

	if count == 0 {
		return 0
	}
	blockDifference /= count

	// Фоновый уровень (средняя разница между соседними пикселями не на границе)
	var backgroundDiff float64
	var bgCount float64

	for y := 1; y < height; y++ {
		for x := 1; x < width; x++ {
			// Пропускаем границы блоков
			if x%8 == 0 || y%8 == 0 {
				continue
			}
			diff := math.Abs(float64(gray.GrayAt(x, y).Y) - float64(gray.GrayAt(x-1, y-1).Y))
			backgroundDiff += diff
			bgCount++
		}
	}

	if bgCount == 0 {
		return 0
	}
	backgroundDiff /= bgCount

	// Blockiness = отношение разницы на границах к фону
	if backgroundDiff == 0 {
		return 0
	}
	return blockDifference / backgroundDiff * 10
}

// ═══════════════════════════════════════════════════════════════════════
// Overall scoring
// ═══════════════════════════════════════════════════════════════════════

// calculateOverall вычисляет общую оценку качества (0-100).
func calculateOverall(m *QualityMetrics) float64 {
	score := 100.0

	// Blur: -30 за сильное размытие
	if m.IsBlurry {
		penalty := math.Min(30, (DefaultConfig().BlurThreshold-m.BlurScore)/DefaultConfig().BlurThreshold*30)
		score -= math.Max(0, penalty)
	}

	// Brightness: -20 за темноту, -10 за пересвет
	if m.IsTooDark {
		score -= 20
	} else if m.IsTooBright {
		score -= 10
	}

	// Contrast: -15 за низкий контраст
	if m.IsLowContrast {
		score -= 15
	}

	// Black screen: -50
	if m.IsBlackScreen {
		score -= 50
	}

	// Frozen: -40
	if m.IsFrozen != nil && *m.IsFrozen {
		score -= 40
	}

	// Noise: -15
	if m.IsNoisy {
		score -= 15
	}

	// Blockiness: -10
	if m.IsBlocky {
		score -= 10
	}

	return math.Max(0, math.Min(100, score))
}

// classifyQuality классифицирует качество по общей оценке.
func classifyQuality(score float64) string {
	switch {
	case score >= 80:
		return "good"
	case score >= 50:
		return "degraded"
	default:
		return "poor"
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// toGrayscale конвертирует image.Image в *image.Gray.
func toGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}

	return gray
}

// Summary возвращает краткое текстовое описание метрик.
func (m *QualityMetrics) Summary() string {
	issues := make([]string, 0)

	if m.IsBlurry {
		issues = append(issues, fmt.Sprintf("blurry(%.1f)", m.BlurScore))
	}
	if m.IsTooDark {
		issues = append(issues, "too dark")
	} else if m.IsTooBright {
		issues = append(issues, "overexposed")
	}
	if m.IsLowContrast {
		issues = append(issues, "low contrast")
	}
	if m.IsBlackScreen {
		issues = append(issues, "black screen")
	}
	if m.IsFrozen != nil && *m.IsFrozen {
		issues = append(issues, "frozen")
	}
	if m.IsNoisy {
		issues = append(issues, "noisy")
	}
	if m.IsBlocky {
		issues = append(issues, "blocky")
	}

	if len(issues) == 0 {
		return fmt.Sprintf("quality=%.0f/100 (good)", m.OverallScore)
	}

	result := fmt.Sprintf("quality=%.0f/100 (%s)", m.OverallScore, m.Status)
	for _, issue := range issues {
		result += ", " + issue
	}
	return result
}
