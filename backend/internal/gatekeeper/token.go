package gatekeeper

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// VerificationClaims — клеймы verification-токена.
// Токен выпускается после успешной верификации и передаётся в CompleteWorkOrder.
type VerificationClaims struct {
	WorkOrderID  string `json:"work_order_id"`
	TechnicianID string `json:"technician_id"`
	GPSPassed    bool   `json:"gps_passed"`
	EXIFPassed   bool   `json:"exif_passed"`
	AIPassed     bool   `json:"ai_passed"`
	GPSSkipped   bool   `json:"gps_skipped"`
	jwt.RegisteredClaims
}

const (
	// VerificationTokenTTL — время жизни verification-токена.
	// Техник должен успеть закрыть наряд в течение этого времени после верификации.
	VerificationTokenTTL = 10 * time.Minute
)

// getJWTSecret возвращает JWT_SECRET из переменных окружения.
func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is required")
	}
	return []byte(secret)
}

// GenerateVerificationToken создаёт JWT-токен, подтверждающий успешную верификацию.
// Токен действует 10 минут и должен быть передан в CompleteWorkOrder.
func GenerateVerificationToken(workOrderID, technicianID string, gpsPassed, exifPassed, aiPassed, gpsSkipped bool) (string, error) {
	now := time.Now()
	claims := VerificationClaims{
		WorkOrderID:  workOrderID,
		TechnicianID: technicianID,
		GPSPassed:    gpsPassed,
		EXIFPassed:   exifPassed,
		AIPassed:     aiPassed,
		GPSSkipped:   gpsSkipped,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "gatekeeper",
			ExpiresAt: jwt.NewNumericDate(now.Add(VerificationTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        workOrderID + "_" + now.Format(time.RFC3339),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ValidateVerificationToken проверяет verification-токен и возвращает клеймы.
func ValidateVerificationToken(tokenString string) (*VerificationClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &VerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*VerificationClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	if claims.Subject != "gatekeeper" {
		return nil, jwt.ErrTokenInvalidSubject
	}

	return claims, nil
}
