package main

import (
	"github.com/gin-gonic/gin"
)

// signup will be handled by filter
func (rm *RouterManager) handleSignup(c *gin.Context) {
	// 회원가입 로직 구현
	c.JSON(200, gin.H{"message": "Signup successful"})
}

// login will be handled by filter
// if the request is valid, it will generate a token
func (rm *RouterManager) handleLogin(c *gin.Context) {
	// 로그인 로직 구현 (JWT 토큰 발행)
	var loginData struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	token, err := rm.jwtManager.GenerateToken(loginData.UserID, loginData.Role)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(200, gin.H{"token": token})
}

// verify will handled by filter
// if the token is valid, it will return the JWT
func (rm *RouterManager) handleVerify(c *gin.Context) {
	// 토큰 검증 로직 구현
	var requestData struct {
		Token string `json:"token"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	claims, err := rm.jwtManager.ValidateToken(requestData.Token)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(200, gin.H{"message": "Token is valid", "claims": claims})
}

func (rm *RouterManager) getHandlerByType(handlerType string) gin.HandlerFunc {
	switch handlerType {
	case "signup":
		log.Debug("Signup handler")
		return rm.handleSignup
	case "login":
		log.Debug("Login handler")
		return rm.handleLogin
	case "verify":
		log.Debug("Verify handler")
		return rm.handleVerify
		//	default:
		//		return func(c *gin.Context) {
		//			c.JSON(404, gin.H{"error": "Handler not found"})
		//		}
	}
	return nil
}
