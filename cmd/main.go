package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

// 로깅 미들웨어
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 다음 미들웨어로 진행
		c.Next()

		// 요청 처리가 완료된 후 로그 기록
		duration := time.Since(start)
		log.Printf("요청 처리 완료: %s %s (소요시간: %v)",
			c.Request.Method,
			c.Request.URL.Path,
			duration,
		)
	}
}

// 인증 미들웨어
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"error": "인증이 필요합니다"})
			c.Abort()
			return
		}
		// 사용자 정보를 컨텍스트에 저장
		c.Set("userId", "user123")
		c.Next()
	}
}

// CORS 미들웨어
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	r := gin.New()

	// 전역 미들웨어 설정
	r.Use(Logger())
	r.Use(CORS())

	// 퍼블릭 API (인증 불필요)
	r.GET("/public", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "누구나 접근 가능한 API"})
	})

	// 인증이 필요한 API 그룹
	authorized := r.Group("/api")
	authorized.Use(Auth()) // 이 그룹의 모든 라우트에 인증 미들웨어 적용
	{
		authorized.GET("/profile", func(c *gin.Context) {
			// Auth 미들웨어에서 설정한 userId 가져오기
			userId, _ := c.Get("userId")
			c.JSON(200, gin.H{
				"message": "프로필 정보",
				"userId":  userId,
			})
		})

		authorized.POST("/upload", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "파일 업로드 완료"})
		})
	}

	// 특정 라우트에 여러 미들웨어 적용
	r.GET("/special",
		Logger(), // 추가 로깅
		Auth(),   // 인증 체크
		func(c *gin.Context) { // 최종 핸들러
			c.JSON(200, gin.H{"message": "특별한 엔드포인트"})
		},
	)

	r.Run(":8080")
}
