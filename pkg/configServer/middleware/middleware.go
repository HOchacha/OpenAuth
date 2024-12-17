// FILE: middleware/middleware.go
package middleware

import (
	"net/http"

	"OpenAuth/pkg/configServer/filters"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("middleware")

// CreateRequestFilterMiddleware는 RequestFilter를 처리하는 Gin 미들웨어를 생성합니다.
func CreateRequestFilterMiddleware(filter *filters.RequestFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info("Starting RequestFilter middleware")

		ret, err := filter.Process(c)
		if ret && (err == nil) {
			log.Info("Filter passed - continuing to next middleware")
			c.Next()
		} else {
			log.Warning("Filter failed - returning 403")
			// JSON 응답과 함께 403 상태 코드 반환
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		}
	}
}

// CreateConditionFilterMiddleware는 ConditionFilter를 처리하는 Gin 미들웨어를 생성합니다.
func CreateConditionFilterMiddleware(filter *filters.ConditionFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info("Starting RequestFilter middleware")

		ret, err := filter.Process(c)
		if ret && (err == nil) {
			log.Info("Filter passed - continuing to next middleware")
			c.Next()
		} else {
			log.Warning("Filter failed - returning 403")
			// JSON 응답과 함께 403 상태 코드 반환
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		}
	}
}
