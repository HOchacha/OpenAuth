package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT operations
type JWTManager struct {
	SecretKey      []byte
	Expiry         time.Duration
	RequiredFields []string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, requiredFields []string) *JWTManager {
	return &JWTManager{
		SecretKey:      []byte(secretKey),
		Expiry:         86400,
		RequiredFields: requiredFields,
	}
}

// GenerateToken creates a new JWT token for a user with required fields
func (m *JWTManager) GenerateToken(data map[string]interface{}) (string, error) {
	// Check for required fields
	for _, field := range m.RequiredFields {
		if _, exists := data[field]; !exists {
			return "", fmt.Errorf("missing required field: %s", field)
		}
	}

	claims := CustomClaims{
		Role: data["role"].(string),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   data["user_id"].(string),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.Expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "OpenAuth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.SecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates the JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenStr string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.SecretKey, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
