package configDispatcher

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Config struct {
	Routes []RouteConfig `yaml:"routes"`
}

type RouteConfig struct {
	Path        string       `yaml:"path"`
	Method      string       `yaml:"method"`
	Middlewares []Middleware `yaml:"middlewares"`
	Proxy       *ProxyConfig `yaml:"proxy,omitempty"`
}

type Middleware struct {
	Name   string                 `yaml:"name"`
	Config map[string]interface{} `yaml:"config"`
}

type ProxyConfig struct {
	Target  string `yaml:"target"`
	Timeout int    `yaml:"timeout"`
}

type MiddlewareFactory func(config map[string]interface{}) gin.HandlerFunc

var middlewareRegistry = make(map[string]MiddlewareFactory)

func registerMiddleware(name string, factory MiddlewareFactory) {
	middlewareRegistry[name] = factory
}

func setupRouter(config Config) *gin.Engine {
	router := gin.New()

	for _, route := range config.Routes {
		handlers := make([]gin.HandlerFunc, 0)

		// 각 미들웨어 생성 및 추가
		for _, mw := range route.Middlewares {
			if factory, exists := middlewareRegistry[mw.Name]; exists {
				handler := factory(mw.Config)
				handlers = append(handlers, handler)
			}
		}

		// 프록시 설정이 있는 경우 최종 핸들러로 추가
		if route.Proxy != nil {
			handlers = append(handlers, createProxyHandler(route.Proxy))
		}

		// 라우트 등록
		router.Handle(route.Method, route.Path, handlers...)
	}

	return router
}

func main() {
	// 미들웨어 팩토리들 등록
	registerMiddleware("auth", createAuthMiddleware)
	registerMiddleware("logger", createLoggerMiddleware)
	registerMiddleware("ratelimit", createRateLimitMiddleware)

	// YAML 설정 파일 읽기
	configData, err := ioutil.ReadFile("routes.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		panic(err)
	}

	// 라우터 설정 및 실행
	router := setupRouter(config)
	router.Run(":8080")
}

// 미들웨어 팩토리 함수들
func createAuthMiddleware(config map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// config에서 설정을 읽어 인증 로직 구현
		realm := config["realm"].(string)
		if token := c.GetHeader("Authorization"); token == "" {
			c.Header("WWW-Authenticate", fmt.Sprintf("Basic realm=%s", realm))
			c.AbortWithStatus(401)
			return
		}
		c.Next()
	}
}

func createLoggerMiddleware(config map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// config에서 설정을 읽어 로깅 구현
		fmt.Printf("Request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

func createRateLimitMiddleware(config map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// config에서 설정을 읽어 레이트 리밋 구현
		limit := config["limit"].(float64)
		fmt.Printf("Rate limit: %v\n", limit)
		c.Next()
	}
}

// 프록시 핸들러 생성
func createProxyHandler(config *ProxyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 프록시 로직 구현
		fmt.Printf("Proxying to: %s\n", config.Target)
		c.JSON(200, gin.H{"proxied_to": config.Target})
	}
}
