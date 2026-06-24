package videoq

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// createTestImage создаёт тестовое изображение заданного цвета.
func createTestImage(width, height int, r, g, b, a uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, g, b, a})
		}
	}
	return img
}

// createGradientImage создаёт градиентное изображение.
func createGradientImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := uint8(float64(x) / float64(width) * 255)
			img.Set(x, y, color.RGBA{val, val, val, 255})
		}
	}
	return img
}

// createNoisyImage создаёт изображение с шумом.
func createNoisyImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			noise := uint8(math.Mod(float64(x*y), 50)) // простой "шум"
			img.Set(x, y, color.RGBA{noise, noise, noise, 255})
		}
	}
	return img
}

func TestAnalyzer_GoodImage(t *testing.T) {
	analyzer := NewAnalyzer()
	img := createGradientImage(640, 480)

	metrics, err := analyzer.Analyze(img)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if metrics.IsBlurry {
		t.Logf("Blur score: %.1f (may be blurry for gradient)", metrics.BlurScore)
	}
	if metrics.IsBlackScreen {
		t.Errorf("gradient should not be black screen (black=%.1f%%)", metrics.BlackPixelPct)
	}
	if metrics.IsTooDark {
		t.Errorf("gradient should not be too dark (brightness=%.1f)", metrics.Brightness)
	}
}

func TestAnalyzer_BlackScreen(t *testing.T) {
	analyzer := NewAnalyzer()
	img := createTestImage(320, 240, 0, 0, 0, 255) // pure black

	metrics, err := analyzer.Analyze(img)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !metrics.IsBlackScreen {
		t.Error("expected black screen detection")
	}
	if metrics.BlackPixelPct < 99 {
		t.Errorf("expected >99%% black pixels, got %.1f%%", metrics.BlackPixelPct)
	}
	if metrics.OverallScore >= 50 {
		t.Errorf("expected low overall score for black screen, got %.1f", metrics.OverallScore)
	}
}

func TestAnalyzer_Brightness(t *testing.T) {
	analyzer := NewAnalyzer()

	// Pure white
	white := createTestImage(100, 100, 255, 255, 255, 255)
	m, _ := analyzer.Analyze(white)
	if m.Brightness < 250 {
		t.Errorf("expected brightness ~255, got %.1f", m.Brightness)
	}
	if !m.IsTooBright {
		t.Error("expected white to be too bright")
	}

	// Pure black
	analyzer.Reset()
	black := createTestImage(100, 100, 0, 0, 0, 255)
	m, _ = analyzer.Analyze(black)
	if m.Brightness > 5 {
		t.Errorf("expected brightness ~0, got %.1f", m.Brightness)
	}
	if !m.IsTooDark {
		t.Error("expected black to be too dark")
	}
}

func TestAnalyzer_FrozenFrame(t *testing.T) {
	analyzer := NewAnalyzer()
	img := createTestImage(100, 100, 128, 128, 128, 255)

	// First frame
	m1, _ := analyzer.Analyze(img)
	if m1.IsFrozen != nil {
		t.Log("first frame has no previous to compare")
	}

	// Same frame again — should be frozen
	m2, _ := analyzer.Analyze(img)
	if m2.IsFrozen == nil {
		t.Fatal("expected frozen detection after second frame")
	}
	if !*m2.IsFrozen {
		t.Error("expected identical frame to be detected as frozen")
	}
	if *m2.FrozenSimilarity < 0.99 {
		t.Errorf("expected similarity >0.99, got %.4f", *m2.FrozenSimilarity)
	}
}

func TestAnalyzer_NotFrozen(t *testing.T) {
	analyzer := NewAnalyzer()

	// Different frames
	frame1 := createTestImage(100, 100, 128, 128, 128, 255)
	frame2 := createTestImage(100, 100, 200, 200, 200, 255)

	analyzer.Analyze(frame1)
	m, _ := analyzer.Analyze(frame2)

	if m.IsFrozen != nil && *m.IsFrozen {
		t.Error("different frames should not be detected as frozen")
	}
}

func TestAnalyzer_Contrast(t *testing.T) {
	analyzer := NewAnalyzer()

	// Solid gray = zero contrast
	solid := createTestImage(100, 100, 128, 128, 128, 255)
	m, _ := analyzer.Analyze(solid)
	if m.Contrast > 1 {
		t.Errorf("expected near-zero contrast for solid image, got %.1f", m.Contrast)
	}

	// Gradient = high contrast
	analyzer.Reset()
	gradient := createGradientImage(100, 100)
	m, _ = analyzer.Analyze(gradient)
	if m.Contrast < 50 {
		t.Errorf("expected high contrast for gradient, got %.1f", m.Contrast)
	}
}

func TestAnalyzer_Noise(t *testing.T) {
	analyzer := NewAnalyzer()

	// Solid = no noise
	solid := createTestImage(100, 100, 128, 128, 128, 255)
	m, _ := analyzer.Analyze(solid)
	if m.NoiseLevel > 1 {
		t.Errorf("expected near-zero noise for solid, got %.1f", m.NoiseLevel)
	}

	// Noisy image
	analyzer.Reset()
	noisy := createNoisyImage(100, 100)
	m, _ = analyzer.Analyze(noisy)
	if m.NoiseLevel < 5 {
		t.Errorf("expected higher noise, got %.1f", m.NoiseLevel)
	}
}

func TestAnalyzer_OverallScore(t *testing.T) {
	tests := []struct {
		name  string
		img   image.Image
		minScore float64
	}{
		{"gradient (good)", createGradientImage(640, 480), 60},
		{"solid gray", createTestImage(100, 100, 128, 128, 128, 255), 40},
		{"black screen", createTestImage(100, 100, 0, 0, 0, 255), 0},
		{"white", createTestImage(100, 100, 255, 255, 255, 255), 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer()
			m, err := analyzer.Analyze(tt.img)
			if err != nil {
				t.Fatalf("Analyze failed: %v", err)
			}
			if m.OverallScore < tt.minScore {
				t.Errorf("expected score >= %.0f, got %.1f", tt.minScore, m.OverallScore)
			}
			t.Logf("  score=%.1f status=%s blur=%.1f brightness=%.1f contrast=%.1f noise=%.1f",
				m.OverallScore, m.Status, m.BlurScore, m.Brightness, m.Contrast, m.NoiseLevel)
		})
	}
}

func TestAnalyzer_BlurDetection(t *testing.T) {
	analyzer := NewAnalyzer()

	// Sharp image (gradient has edges)
	sharp := createGradientImage(200, 200)
	m, _ := analyzer.Analyze(sharp)
	if m.BlurScore <= 0 {
		t.Errorf("expected positive blur score for gradient, got %.1f", m.BlurScore)
	} else {
		t.Logf("blur score for gradient: %.1f (anything > 0 is ok)", m.BlurScore)
	}

	// Very blurry = solid color (no edges)
	analyzer.Reset()
	solid := createTestImage(200, 200, 128, 128, 128, 255)
	m, _ = analyzer.Analyze(solid)
	if m.BlurScore > 1 {
		t.Errorf("expected near-zero blur score for solid, got %.1f", m.BlurScore)
	}
	if !m.IsBlurry {
		t.Error("solid image should be detected as blurry")
	}
}

func TestAnalyzer_Summary(t *testing.T) {
	analyzer := NewAnalyzer()
	img := createGradientImage(320, 240)
	m, _ := analyzer.Analyze(img)

	summary := m.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	t.Logf("Summary: %s", summary)
}

func TestAnalyzer_Reset(t *testing.T) {
	analyzer := NewAnalyzer()
	analyzer.Analyze(createTestImage(10, 10, 128, 128, 128, 255))
	analyzer.Reset()

	if analyzer.prevFrame != nil {
		t.Error("expected prevFrame to be nil after reset")
	}
}

func TestBlockiness(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 64, 64))
	// Fill with pattern that has 8x8 block boundaries
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			// Emphasize 8x8 block boundaries
			val := uint8(128)
			if x%8 == 0 || y%8 == 0 {
				val = 200 // brighter at block boundaries
			}
			gray.Set(x, y, color.Gray{val})
		}
	}

	blockiness := detectBlockiness(gray)
	if blockiness < 1 {
		t.Errorf("expected blockiness > 1 for blocky image, got %.2f", blockiness)
	}
}

func TestGrayscale(t *testing.T) {
	rgb := image.NewRGBA(image.Rect(0, 0, 10, 10))
	rgb.Set(5, 5, color.RGBA{100, 150, 200, 255})

	gray := toGrayscale(rgb)
	if gray == nil {
		t.Fatal("expected non-nil grayscale image")
	}

	// Check that the color was converted
	c := gray.GrayAt(5, 5)
	if c.Y == 0 {
		t.Error("expected non-zero grayscale value")
	}
}

func TestBrightnessZeroDivision(t *testing.T) {
	empty := image.NewGray(image.Rect(0, 0, 0, 0))
	b := calculateBrightness(empty)
	if b != 0 {
		t.Errorf("expected 0 for empty image, got %.1f", b)
	}
}
