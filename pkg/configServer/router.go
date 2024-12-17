// FILE: router.go
package configServer

import (
	"OpenAuth/pkg/configServer/filters"
	"OpenAuth/pkg/configServer/middleware"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Routes []RouteConfig `yaml:"routes"`
	//JWTConfig jwt.JWTConfig `yaml:"jwt_config"`
}

type RouteConfig struct {
	Path            string                   `yaml:"path"`
	Method          string                   `yaml:"method"`
	RequestFilters  []filters.RequestFilter  `yaml:"request_filters"`
	ConditionFilter *filters.ConditionFilter `yaml:"condition_filter,omitempty"`
	HandlerType     string                   `yaml:"handler_type"`
}

// this function is used on mockup server not for production
func SetupRouter(config Config) *gin.Engine {
	router := gin.New()

	for _, route := range config.Routes {
		handlers := make([]gin.HandlerFunc, 0)

		// RequestFilter 처리
		for _, rf := range route.RequestFilters {
			handlers = append(handlers, middleware.CreateRequestFilterMiddleware(&rf))
		}

		// ConditionFilter 처리
		if route.ConditionFilter != nil {
			handlers = append(handlers, middleware.CreateConditionFilterMiddleware(route.ConditionFilter))
		}

		router.Handle(route.Method, route.Path, handlers...)
	}

	return router
}
