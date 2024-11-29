package main

import (
	"OpenAuth/pkg/configDispatcher/configServer"
	"OpenAuth/pkg/k8sQuery" // k8sQuery 패키지 import
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type RouterManager struct {
	engine         *gin.Engine
	mu             sync.RWMutex
	config         *configServer.Config
	tokenValidator *k8sQuery.TokenValidator // TokenValidator 추가
}

func NewRouterManager() (*RouterManager, error) {
	// TokenValidator 초기화
	saConfig := k8sQuery.ServiceAccountConfig{
		AllowedAccounts: map[string][]string{
			"default": {"config-updater"}, // 설정 업데이트 권한을 가진 ServiceAccount
		},
	}

	validator, err := k8sQuery.NewTokenValidator(saConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create token validator: %v", err)
	}

	rm := &RouterManager{
		engine:         gin.Default(),
		tokenValidator: validator,
	}

	// /config 엔드포인트에 인증 미들웨어 적용
	configGroup := rm.engine.Group("/config")
	configGroup.Use(k8sQuery.AuthMiddleware(validator))
	configGroup.POST("", rm.handleConfigUpdate)

	return rm, nil
}

func (rm *RouterManager) handleConfigUpdate(c *gin.Context) {
	yamlData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}

	var newConfig configServer.Config
	if err := yaml.Unmarshal(yamlData, &newConfig); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse YAML: %v", err)})
		return
	}

	if err := rm.UpdateConfig(&newConfig); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update router: %v", err)})
		return
	}

	c.JSON(200, gin.H{"message": "Router configuration updated successfully"})
}

func (rm *RouterManager) UpdateConfig(newConfig *configServer.Config) error {
	// 뮤텍스 활성화 후, gin 객체 재생성 및 새로 적용
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 새로운 서버 엔진 생성
	newEngine := gin.Default()

	// 새로운 엔진에도 인증된 /config 엔드포인트 설정
	configGroup := newEngine.Group("/config")
	configGroup.Use(k8sQuery.AuthMiddleware(rm.tokenValidator))
	configGroup.POST("", rm.handleConfigUpdate)

	// 나머지 라우트 설정
	for _, route := range newConfig.Routes {
		handlers := make([]gin.HandlerFunc, 0)
		for _, rf := range route.RequestFilters {
			handlers = append(handlers, configServer.CreateRequestFilterMiddleware(&rf))
		}
		if route.ConditionFilter != nil {
			handlers = append(handlers, configServer.CreateConditionFilterMiddleware(route.ConditionFilter))
		}
		if route.Proxy != nil {
			handlers = append(handlers, configServer.CreateProxyHandler(route.Proxy))
		}
		finalHandler := handlers[len(handlers)-1]
		handlers = handlers[:len(handlers)-1]
		newEngine.Handle(route.Method, route.Path, append(handlers, finalHandler)...)
	}

	rm.engine = newEngine
	rm.config = newConfig
	return nil
}

func (rm *RouterManager) GetEngine() *gin.Engine {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine
}

func StartServer(address string) error {
	routerManager, err := NewRouterManager()
	if err != nil {
		return fmt.Errorf("failed to create router manager: %v", err)
	}
	return routerManager.GetEngine().Run(address)
}
