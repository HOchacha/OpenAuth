package main

import (
	"OpenAuth/pkg/configDispatcher/configServer"
	"OpenAuth/pkg/k8sQuery" // k8sQuery 패키지 import
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
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
			"default": {"oauth-configurator"}, // 설정 업데이트 권한을 가진 ServiceAccount
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
	log.Debugf("handleConfigUpdate: Received a new configuration update request")

	// 요청 본문 읽기
	yamlData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Debugf("Failed to read request body: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}
	log.Debugf("Request body successfully read. Length: %d bytes", len(yamlData))

	// YAML 파싱
	var newConfig configServer.Config
	if err := yaml.Unmarshal(yamlData, &newConfig); err != nil {
		log.Debugf("Failed to parse YAML: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse YAML: %v", err)})
		return
	}
	log.Debugf("YAML parsed successfully: %+v", newConfig)

	// 라우터 설정 업데이트
	log.Debugf("Attempting to update router configuration...")
	if err := rm.UpdateConfig(&newConfig); err != nil {
		log.Debugf("Failed to update router: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update router: %v", err)})
		return
	}

	log.Debugf("Router configuration updated successfully")
	c.JSON(200, gin.H{"message": "Router configuration updated successfully"})
}

func (rm *RouterManager) UpdateConfig(newConfig *configServer.Config) error {
	log.Debugf("UpdateConfig: Received request to update configuration")
	log.Debugf("New Config: %+v", newConfig)

	// 뮤텍스 활성화 후, gin 객체 재생성 및 새로 적용
	log.Debugf("Acquiring lock to update router configuration")
	rm.mu.Lock()
	defer rm.mu.Unlock()
	log.Debugf("Lock acquired")

	// 새로운 서버 엔진 생성
	log.Debugf("Creating a new Gin engine")
	newEngine := gin.Default()
	log.Debugf("New Gin engine created: %+v", newEngine)

	// 새로운 엔진에도 인증된 /config 엔드포인트 설정
	log.Debugf("Setting up /config endpoint with authentication middleware")
	configGroup := newEngine.Group("/config")
	configGroup.Use(k8sQuery.AuthMiddleware(rm.tokenValidator))
	configGroup.POST("", rm.handleConfigUpdate)
	log.Debugf("/config endpoint configured successfully")

	// 나머지 라우트 설정
	log.Debugf("Configuring additional routes")
	for _, route := range newConfig.Routes {
		log.Debugf("Processing route: Method=%s, Path=%s", route.Method, route.Path)

		handlers := make([]gin.HandlerFunc, 0)

		// RequestFilters 추가
		for _, rf := range route.RequestFilters {
			log.Debugf("Adding RequestFilter middleware: %+v", rf)
			handlers = append(handlers, configServer.CreateRequestFilterMiddleware(&rf))
		}

		// ConditionFilter 추가
		if route.ConditionFilter != nil {
			log.Debugf("Adding ConditionFilter middleware: %+v", route.ConditionFilter)
			handlers = append(handlers, configServer.CreateConditionFilterMiddleware(route.ConditionFilter))
		}

		// 핸들러 설정
		if len(handlers) > 0 {
			finalHandler := handlers[len(handlers)-1]
			handlers = handlers[:len(handlers)-1]
			log.Debugf("Final middleware added: %+v", finalHandler)
			newEngine.Handle(route.Method, route.Path, append(handlers, finalHandler)...)
		} else {
			log.Debugf("No handlers found for route: Method=%s, Path=%s", route.Method, route.Path)
		}
	}

	log.Debugf("All routes configured successfully")

	// 새로운 엔진과 설정 적용
	log.Debugf("Updating RouterManager with new configuration and engine")
	rm.engine = newEngine
	rm.config = newConfig
	log.Debugf("RouterManager updated successfully: Engine=%+v, Config=%+v", rm.engine, rm.config)

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
