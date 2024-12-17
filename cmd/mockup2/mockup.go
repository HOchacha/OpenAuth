package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// Config 구조체 정의 (예제)
type Config struct {
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Method          string           `yaml:"method"`
	Path            string           `yaml:"path"`
	RequestFilters  []RequestFilter  `yaml:"request_filters"`
	ConditionFilter *ConditionFilter `yaml:"condition_filter"`
	HandlerType     string           `yaml:"handler_type"`
}

type RequestFilter struct {
	FilterName    string            `yaml:"filter_name"`
	RemoteServer  string            `yaml:"remote_server"`
	RequestFormat map[string]string `yaml:"request_format"`
	FieldsToSend  []string          `yaml:"fields_to_send"`
	client        *http.Client
}

func (rf *RequestFilter) Process(c *gin.Context) bool {
	if rf.client == nil {
		rf.client = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		}
	}

	// 1. 요청 본문을 읽음
	var requestBody map[string]interface{}
	rawData, err := c.GetRawData()
	if err != nil {
		return false
	}

	// 2. 본문을 다시 Context에 설정
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData))

	// 3. JSON 파싱
	if err := json.Unmarshal(rawData, &requestBody); err != nil {
		return false
	}

	// 4. 다음 미들웨어를 위해 본문 다시 설정
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData))

	// 지정된 필드만 선택하여 새로운 맵 생성 -> 새로운 Request Body
	filteredBody := make(map[string]interface{})
	for _, field := range rf.FieldsToSend {
		if value, exists := requestBody[field]; exists {
			filteredBody[field] = value
		}
	}

	jsonBody, err := json.Marshal(filteredBody)
	if err != nil {
		fmt.Printf("Error marshaling filtered body: %v\n", err)
		return false
	}

	req, err := http.NewRequest("POST", rf.RemoteServer, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false
	}

	for key, value := range rf.RequestFormat {
		req.Header.Set(key, value)
	}

	resp, err := rf.client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		// 실제 응답 구조에 맞게 수정
		var result struct {
			Message string `json:"message"`
			OTP     int    `json:"otp"`
		}

		// body를 다시 읽을 수 있도록 새로운 Reader 생성
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return false
		}

		// 200 OK와 message가 있으면 성공으로 처리
		return true // StatusOK이면 성공으로 처리
	}

	return true
}

type ConditionFilter struct {
	// 필터 필드 정의
}

// RouterManager 구조체 정의
type RouterManager struct {
	mu             sync.RWMutex
	engine         *gin.Engine
	config         *Config
	tokenValidator TokenValidator
}

// TokenValidator 인터페이스 정의 (예제)
type TokenValidator interface {
	ValidateToken(token string) bool
}

// handleConfigUpdate 함수 정의
func (rm *RouterManager) handleConfigUpdate(c *gin.Context) {
	log.Println("handleConfigUpdate: Received a new configuration update request")

	// read request body
	yamlData, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to read request body: %v", err)})
		return
	}
	log.Printf("Request body successfully read. Length: %d bytes", len(yamlData))

	// Parse YAML
	var newConfig Config
	if err := yaml.Unmarshal(yamlData, &newConfig); err != nil {
		log.Printf("Failed to parse YAML: %v", err)
		c.JSON(400, gin.H{"error": fmt.Sprintf("Failed to parse YAML: %v", err)})
		return
	}
	log.Printf("YAML parsed successfully: %+v", newConfig)

	// Update Router Configuration
	log.Println("Attempting to update router configuration...")
	if err := rm.UpdateConfig(&newConfig); err != nil {
		log.Printf("Failed to update router: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to update router: %v", err)})
		return
	}

	log.Println("Router configuration updated successfully")
	c.JSON(200, gin.H{"message": "Router configuration updated successfully"})
}

// UpdateConfig 함수 정의
func (rm *RouterManager) UpdateConfig(newConfig *Config) error {
	log.Println("UpdateConfig: Received request to update configuration")
	log.Printf("New Config: %+v", newConfig)

	// for thread safety, acquire lock.
	log.Println("Acquiring lock to update router configuration")

	rm.mu.Lock()
	defer rm.mu.Unlock()
	log.Println("Lock acquired")

	log.Println("Creating a new Gin engine")
	newEngine := gin.Default()
	log.Printf("New Gin engine created: %+v", newEngine)

	// Set /config endpoint
	log.Println("Setting up /config endpoint with authentication middleware")
	configGroup := newEngine.Group("/config")
	configGroup.Use(AuthMiddleware(rm.tokenValidator))
	configGroup.POST("", rm.handleConfigUpdate)
	log.Println("/config endpoint configured successfully")

	// set the router with the given configuration via /config endpoint
	log.Println("Configuring additional routes")
	for _, route := range newConfig.Routes {
		log.Printf("Processing route: Method=%s, Path=%s", route.Method, route.Path)

		handlers := make([]gin.HandlerFunc, 0)

		// Add RequestFilters
		for _, rf := range route.RequestFilters {
			log.Printf("Adding RequestFilter middleware: %+v", rf)
			handlers = append(handlers, CreateRequestFilterMiddleware(rf))
		}

		// Add ConditionFilter
		if route.ConditionFilter != nil {
			log.Printf("Adding ConditionFilter middleware: %+v", route.ConditionFilter)
			handlers = append(handlers, CreateConditionFilterMiddleware(route.ConditionFilter))
		}

		// set final handler
		finalHandler := getHandlerByType(route.HandlerType)
		handlers = append(handlers, finalHandler)
		log.Printf("Final middleware added: %s", finalHandler)
		newEngine.Handle(route.Method, route.Path, handlers...)
	}

	log.Println("All routes configured successfully")

	// Set new engine and configuration
	log.Println("Updating RouterManager with new configuration and engine")
	rm.engine = newEngine
	rm.config = newConfig
	log.Printf("RouterManager updated successfully: Engine=%+v\n\n Config=%+v\n", rm.engine, rm.config)

	return nil
}

// AuthMiddleware 예제
func AuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 인증 로직
		c.Next()
	}
}

// CreateRequestFilterMiddleware는 RequestFilter를 처리하는 Gin 미들웨어를 생성합니다.
func CreateRequestFilterMiddleware(filter RequestFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if filter.Process(c) {
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

// CreateConditionFilterMiddleware 예제
func CreateConditionFilterMiddleware(cf *ConditionFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("ConditionFilterMiddleware: Before request")
		// 요청 처리 로직
		c.Next()
		log.Println("ConditionFilterMiddleware: After request")
	}
}

// getHandlerByType 예제
func getHandlerByType(handlerType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("Handling request with handler type: %s", handlerType)
		c.JSON(http.StatusOK, gin.H{"message": "Request processed successfully"})
	}
}

func main() {
	// YAML 파일 경로를 명령줄 인수로 받음
	configFile := flag.String("config", "config.yaml", "Path to the configuration YAML file")
	flag.Parse()

	// YAML 파일 읽기
	yamlData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// YAML 파싱
	var config Config
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// RouterManager 인스턴스 생성
	rm := &RouterManager{
		engine: gin.Default(),
		config: &config,
		// 필요한 필드 초기화
	}

	// 초기 구성 설정
	if err := rm.UpdateConfig(&config); err != nil {
		log.Fatalf("Failed to update router configuration: %v", err)
	}

	// 서버 실행
	if err := rm.engine.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
