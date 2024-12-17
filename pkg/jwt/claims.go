package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims defines custom claims extending jwt.RegisteredClaims
type CustomClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}
