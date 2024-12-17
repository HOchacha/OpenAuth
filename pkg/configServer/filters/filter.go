package filters

import "github.com/gin-gonic/gin"

// Filter는 모든 필터가 구현해야 하는 인터페이스입니다.
type Filter interface {
	Process(c *gin.Context) bool
}
