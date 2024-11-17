package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"OpenAuth/pkg/jwt" // Replace with your actual module path
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Gin
	r := gin.Default()

	// Set up JWT Verifier (using a secret for simplicity)
	secret := []byte("your-secret-key")
	allowedUsers := map[string]bool{"user123": true} // Example allowed users
	verifier := jwt.NewJWTVerifier(secret, allowedUsers)

	// Define routes
	r.POST("/generate_jwt", GenerateJWTHandler)
	r.POST("/jwt_verifier", VerifyJWTHandler)

	// Protected route requiring JWT authentication
	r.GET("/protected", JWTAuthMiddleware(verifier), ProtectedHandler)

	// Start the server
	err := r.Run(":8080")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// Handler to generate a JWT
func GenerateJWTHandler(c *gin.Context) {
	// Step 1: Generate RSA Private Key for the JWT
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Error generating private key for JWT: %v", err)})
		return
	}

	// Step 2: Prepare JWT claims
	jwtClaims := jwt.Claims{
		Subject: "user123",
		Email:   "user123@example.com",
		Exp:     time.Now().Add(1 * time.Hour).Unix(), // 1 hour expiration
	}

	// Step 3: Generate JWT using the private key
	token, err := jwt.GenerateJWT(privateKey, jwtClaims)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Error generating JWT: %v", err)})
		return
	}

	// Return JWT in the response
	c.JSON(200, gin.H{
		"message": "JWT generated successfully",
		"token":   token,
	})
}

// Handler to verify a JWT
func VerifyJWTHandler(c *gin.Context) {
	// Step 1: Get JWT from the request body
	var request struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// Step 2: Generate a dummy public key (you can replace it with the actual public key used for signing)
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// Step 3: Verify the JWT using the public key
	isValid, err := jwt.VerifyJWT(request.Token, publicKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Error verifying JWT: %v", err)})
		return
	}

	// Return verification result
	if isValid {
		c.JSON(200, gin.H{
			"message": "JWT is valid",
		})
	} else {
		c.JSON(400, gin.H{
			"message": "JWT is invalid",
		})
	}
}

// Protected route requiring JWT authentication
func ProtectedHandler(c *gin.Context) {
	userID, _ := c.Get("userID")
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Hello, %s! You have access to the protected route.", userID),
	})
}
