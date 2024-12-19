package jwt

import (
	"fmt"
	"log"
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
		Expiry:         24 * time.Hour,
		RequiredFields: requiredFields,
	}
}

// GenerateToken는 주어진 데이터를 기반으로 JWT 토큰을 생성합니다.
func (m *JWTManager) GenerateToken(data map[string]interface{}) (string, error) {
	log.Println("Generating token with data:", data)
	for _, field := range m.RequiredFields {
		if _, exists := data[field]; !exists {
			return "", fmt.Errorf("missing required field: %s", field)
		}
	}

	currentTime := time.Now()
	claims := CustomClaims{
		Role: data["role"].(string),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   data["username"].(string),
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(m.Expiry)),
			IssuedAt:  jwt.NewNumericDate(currentTime),
			NotBefore: jwt.NewNumericDate(currentTime),
			Issuer:    "OpenAuth",
		},
	}

	log.Println("Claims:", claims)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	log.Println("Signing token with SecretKey:", string(m.SecretKey))

	signedToken, err := token.SignedString(m.SecretKey)
	if err != nil {
		log.Printf("Failed to sign token: %v", err)
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	log.Println("Generated signed token:", signedToken)
	log.Printf("Token issued at: %v, expires at: %v, duration: %v", *claims.IssuedAt, *claims.ExpiresAt, m.Expiry)
	return signedToken, nil
}

// ValidateToken는 JWT 토큰을 검증하고 클레임을 반환합니다.
func (m *JWTManager) ValidateToken(tokenStr string) (*CustomClaims, error) {
	log.Println("Validating token:", tokenStr)

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// 서명 방법 검증
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				errMsg := fmt.Sprintf("unexpected signing method: %v", token.Header["alg"])
				log.Println(errMsg)
				return nil, fmt.Errorf(errMsg)
			}
			return m.SecretKey, nil
		},
	)

	if err != nil {
		log.Printf("Error parsing token: %v", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		log.Println("Invalid token claims")
		return nil, fmt.Errorf("invalid token claims")
	}

	log.Println("Token claims:", claims)

	// 현재 시간 기록
	currentTime := time.Now()
	log.Printf("Current server time: %v", currentTime)

	// 토큰의 유효 기간과 ��재 시간을 비교
	expirationTime := claims.ExpiresAt.Time
	log.Printf("Token expires at: %v", expirationTime)
	if currentTime.After(expirationTime) {
		log.Println("Token has expired")
		return nil, fmt.Errorf("token is expired")
	}

	// 토큰이 유효한지 확인하기 위해 나머지 검증 수행
	// 필수 필드 검증
	for _, field := range m.RequiredFields {
		switch field {
		case "role":
			if claims.Role == "" {
				errMsg := "missing required field: role"
				log.Println(errMsg)
				return nil, fmt.Errorf(errMsg)
			}
		// 다른 필드도 필요에 따라 추가
		default:
			return nil, nil
		}
	}

	// 토큰이 유효하고 모든 필드가 검증되었음을 로그에 기록
	log.Printf("Token is valid. Time since issued: %v, time until expiration: %v", currentTime.Sub(claims.IssuedAt.Time), claims.ExpiresAt.Time.Sub(currentTime))
	return claims, nil
}
