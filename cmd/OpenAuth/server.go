package main

import (
	configServer "OpenAuth/pkg/configDispatcher/configServer"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
)

type RouterManager struct {
	engine *gin.Engine
	mu     sync.RWMutex
	config *configServer.Config
}

func NewRouterManager() *RouterManager {
	rm := &RouterManager{
		engine: gin.Default(),
	}

	rm.engine.POST("/config", rm.handleConfigUpdate)

	return rm
}

// "/config" 경로는 기본적으로 존재하는 path로, 원격으로 yaml 구성을 받을 수 있도록 한다.
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
	rm.mu.Lock()
	defer rm.mu.Unlock()

	newEngine := gin.Default()

	newEngine.POST("/config", rm.handleConfigUpdate)

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
	routerManager := NewRouterManager()
	return routerManager.GetEngine().Run(address)
}
