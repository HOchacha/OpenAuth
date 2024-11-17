package jwt

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
)

// Claims defines the JWT payload structure
type Claims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Exp     int64  `json:"exp"`
}

// GenerateJWT creates a JWT token from the private key and claims
func GenerateJWT(privateKey *rsa.PrivateKey, claims Claims) (string, error) {
	// Marshal the claims to JSON
	claimsData, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %v", err)
	}

	// For simplicity, we will not implement the full JWT structure here
	// For a real implementation, the claims would be signed using the private key
	// In this case, we are simulating the generation of the JWT token
	token := fmt.Sprintf("header.%s.signature", string(claimsData)) // A placeholder format

	return token, nil
}
