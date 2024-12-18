package main

import (
	"OpenAuth/pkg/configServer"
	"OpenAuth/pkg/configServer/middleware"
	"OpenAuth/pkg/jwt"
	"OpenAuth/pkg/k8sQuery"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

type HandlerSwitcher struct {
	mu      sync.RWMutex
	handler http.Handler
}

func (hs *HandlerSwitcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hs.mu.RLock()
	handler := hs.handler
	hs.mu.RUnlock()
	handler.ServeHTTP(w, r)
}

func (hs *HandlerSwitcher) UpdateHandler(newHandler http.Handler) {
	hs.mu.Lock()
	hs.handler = newHandler
	hs.mu.Unlock()
}

type RouterManager struct {
	engine          *gin.Engine
	server          *http.Server
	handlerSwitcher *HandlerSwitcher
	mu              sync.RWMutex
	config          *configServer.Config
	tokenValidator  *k8sQuery.TokenValidator
	jwtManager      *jwt.JWTManager
}

// this creates bear gin engine and set /config endpoint
// To utilize this, you must config the router by /config endpoint with configuration yaml.
func NewRouterManager() (*RouterManager, error) {

	// Create TokenValidator for Integrity
	// TokenValidator checks the JWT token signed by k8s api server with the given ServiceAccount
	// If the token is named as below, it will be allowed to access the /config endpoint.
	saConfig := k8sQuery.ServiceAccountConfig{
		AllowedAccounts: map[string][]string{
			"default": {"oauth-configurator"},
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

	// /config endpoint
	configGroup := rm.engine.Group("/config")
	configGroup.Use(k8sQuery.AuthMiddleware(validator))
	configGroup.POST("", rm.handleConfigUpdate)

	return rm, nil
}

// the function handle /config endpoint
// this must embed on the gin engine in initiative time.
func (rm *RouterManager) handleConfigUpdate(c *gin.Context) {
	log.Debugf("handleConfigUpdate: Received a new configuration update request")

	// read request body
	yamlData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Debugf("Failed to read request body: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}
	log.Debugf("Request body successfully read. Length: %d bytes", len(yamlData))

	// Parse YAML
	var newConfig configServer.Config
	if err := yaml.Unmarshal(yamlData, &newConfig); err != nil {
		log.Debugf("Failed to parse YAML: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse YAML: %v", err)})
		return
	}
	log.Debugf("YAML parsed successfully: %+v", newConfig)

	// Update Router Configuration
	log.Debugf("Attempting to update router configuration...")
	if err := rm.UpdateConfig(&newConfig); err != nil {
		log.Debugf("Failed to update router: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update router: %v", err)})
		return
	}

	log.Debugf("Router configuration updated successfully")
	c.JSON(200, gin.H{"message": "Router configuration updated successfully"})
}

// this function updates the configuration of the router
func (rm *RouterManager) UpdateConfig(newConfig *configServer.Config) error {
	log.Debugf("UpdateConfig: Received request to update configuration")
	log.Debugf("New Config: %+v", newConfig)

	// for thread safety, acquire lock.
	// gin.engine shared by all requests, so it must be updated atomically.
	log.Debugf("Acquiring lock to update router configuration")

	/* Critical Section Start */

	rm.mu.Lock()
	defer rm.mu.Unlock() // defines unlock after function returns
	log.Debugf("Lock acquired")

	rm.jwtManager = jwt.NewJWTManager(
		newConfig.JWTConfig.SecretKey,
		newConfig.JWTConfig.RequiredFields,
	)

	log.Debugf("Creating a new Gin engine")
	newEngine := gin.Default()
	log.Debugf("New Gin engine created: %+v", newEngine)

	// Set /config endpoint
	log.Debugf("Setting up /config endpoint with authentication middleware")
	configGroup := newEngine.Group("/config")
	configGroup.Use(k8sQuery.AuthMiddleware(rm.tokenValidator))
	configGroup.POST("", rm.handleConfigUpdate)
	log.Debugf("/config endpoint configured successfully")

	// set the router with the given configuration via /config endpoint
	log.Debugf("Configuring additional routes")
	for _, route := range newConfig.Routes {
		log.Debugf("Processing route: Method=%s, Path=%s", route.Method, route.Path)

		handlers := make([]gin.HandlerFunc, 0)

		// Add RequestFilters
		for _, rf := range route.RequestFilters {
			log.Debugf("Adding RequestFilter middleware: %+v", rf)
			handlers = append(handlers, middleware.CreateRequestFilterMiddleware(&rf))
		}

		// Add ConditionFilter
		if route.ConditionFilter != nil {
			log.Debugf("Adding ConditionFilter middleware: %+v", route.ConditionFilter)
			handlers = append(handlers, middleware.CreateConditionFilterMiddleware(route.ConditionFilter))
		}

		// set final handler
		log.Debugf("Final Handler Type: %s", route.HandlerType)
		finalHandler := rm.getHandlerByType(route.HandlerType)
		if finalHandler == nil {
			panic("wrong configuration")
		}
		handlers = append(handlers, finalHandler)
		log.Debugf("Final middleware added: %s", finalHandler)
		newEngine.Handle(route.Method, route.Path, handlers...)
	}

	log.Debugf("All routes configured successfully")

	// Set new engine and configuration
	log.Debugf("Updating RouterManager with new configuration and engine")
	rm.engine = newEngine
	rm.config = newConfig
	log.Debugf("RouterManager updated successfully: Engine=%+v\n\n Config=%+v\n", rm.engine, rm.config)

	if rm.handlerSwitcher != nil {
		rm.handlerSwitcher.UpdateHandler(rm.engine)
	}

	return nil

	/* Critical Section ended by defer */
}

func (rm *RouterManager) GetEngine() *gin.Engine {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.engine
}

// StartServer create initiative server with engine and address
func StartServer(address string) error {
	routerManager, err := NewRouterManager()
	if err != nil {
		return fmt.Errorf("failed to create router manager: %v", err)
	}

	handlerSwitcher := &HandlerSwitcher{handler: routerManager.GetEngine()}
	routerManager.handlerSwitcher = handlerSwitcher

	routerManager.server = &http.Server{
		Addr:    address,
		Handler: handlerSwitcher,
	}

	return routerManager.server.ListenAndServe()
}
