package jwt

import (
	"crypto/rsa"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// JWTVerifier is a struct for verifying JWT tokens
type JWTVerifier struct {
	secret       []byte
	allowedUsers map[string]bool
}

// NewJWTVerifier creates a new JWTVerifier
func NewJWTVerifier(secret []byte, allowedUsers map[string]bool) *JWTVerifier {
	return &JWTVerifier{secret: secret, allowedUsers: allowedUsers}
}

// VerifyToken validates a JWT token and checks if the user is allowed
func (verifier *JWTVerifier) VerifyToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return verifier.secret, nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["userID"].(string)
		if !ok || !verifier.allowedUsers[userID] {
			return "", errors.New("user not allowed or invalid token")
		}
		return userID, nil
	}
	return "", errors.New("invalid token")
}

// VerifyJWT verifies a JWT using the given public key and returns a boolean indicating validity.
func VerifyJWT(tokenString string, publicKey *rsa.PublicKey) (bool, error) {
	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})

	if err != nil || !token.Valid {
		return false, err
	}
	return true, nil
}
