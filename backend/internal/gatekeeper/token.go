package gatekeeper

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"gb-telemetry-collector/internal/auth"
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

// GenerateVerificationToken создаёт JWT-токен с подписью bign-curve256v1 (ECDSA P-256),
// подтверждающий успешную верификацию. Токен действует 10 минут.
//
// Возвращает error если BIGN_PRIVATE_KEY не установлен (graceful degradation).
func GenerateVerificationToken(workOrderID, technicianID string, gpsPassed, exifPassed, aiPassed, gpsSkipped bool) (string, error) {
	key, err := auth.GetBignPrivateKey()
	if err != nil {
		return "", err
	}

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

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(key)
}

// ValidateVerificationToken проверяет verification-токен и возвращает клеймы.
//
// Возвращает error если BIGN_PRIVATE_KEY не установлен или токен невалиден.
func ValidateVerificationToken(tokenString string) (*VerificationClaims, error) {
	key, err := auth.GetBignPublicKey()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &VerificationClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return key, nil
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
