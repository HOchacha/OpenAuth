package configServer

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimitFilter struct {
	Rate     float64 `yaml:"rate"`
	Burst    int     `yaml:"burst"`
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

func NewRateLimitFilter(rate float64, burst int) *RateLimitFilter {
	return &RateLimitFilter{
		Rate:     rate,
		Burst:    burst,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimitFilter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.Rate), rl.Burst)
		rl.limiters[ip] = limiter
	}
	return limiter
}

func (rl *RateLimitFilter) Process(c *gin.Context) bool {
	ip := c.ClientIP()
	limiter := rl.getLimiter(ip)
	if !limiter.Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return false
	}
	return true
}

func CreateRateLimitFilterMiddleware(filter *RateLimitFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !filter.Process(c) {
			c.Abort()
			return
		}
		c.Next()
	}
}
