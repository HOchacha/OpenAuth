package handler

import (
	jwt "OpenAuth/pkg/jwt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// JWTAuthMiddleware is a Gin middleware for authenticating requests with JWT
func JWTAuthMiddleware(verifier *jwt.JWTVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Parse the token from the header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := verifier.VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Set userID in the context, so it can be used in handlers
		c.Set("userID", userID)
		c.Next()
	}
}
