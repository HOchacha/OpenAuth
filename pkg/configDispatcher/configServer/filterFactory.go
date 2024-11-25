package configServer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

// Filter 인터페이스 정의
type Filter interface {
	Process(c *gin.Context) bool
}

// RequestFilter 구조체 정의
type RequestFilter struct {
	RemoteServer  string            `yaml:"remote_server"`
	RequestFormat map[string]string `yaml:"request_format"`
	FieldsToSend  []string          `yaml:"fields_to_send"`
	client        *http.Client
}

// ConditionFilter 구조체 정의
type ConditionFilter struct {
	Conditions []Condition `yaml:"conditions"`
}

type Condition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

// Config 구조체 수정
type Config struct {
	Routes []RouteConfig `yaml:"routes"`
}

type ProxyConfig struct {
	Target  string `yaml:"target"`
	Timeout int    `yaml:"timeout"`
}

type RouteConfig struct {
	Path            string           `yaml:"path"`
	Method          string           `yaml:"method"`
	RequestFilters  []RequestFilter  `yaml:"request_filters"`
	ConditionFilter *ConditionFilter `yaml:"condition_filter,omitempty"`
	Proxy           *ProxyConfig     `yaml:"proxy,omitempty"`
}

func (rf *RequestFilter) Process(c *gin.Context) bool {
	if rf.client == nil {
		rf.client = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	// 현재 전달받은 요청의 HTTP Body 체크
	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		fmt.Printf("Error reading request body: %v\n", err)
		return false
	}

	// 지정된 필드만 선택하여 새로운 맵 생성 -> 새로운 Request Body
	filteredBody := make(map[string]interface{})
	for _, field := range rf.FieldsToSend {
		if value, exists := requestBody[field]; exists {
			filteredBody[field] = value
		}
	}

	// JSON으로 마샬링
	jsonBody, err := json.Marshal(filteredBody)
	if err != nil {
		fmt.Printf("Error marshaling filtered body: %v\n", err)
		return false
	}

	// 원격 서버로 요청 전송
	req, err := http.NewRequest("POST", rf.RemoteServer, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false
	}

	// 설정된 헤더 추가
	for key, value := range rf.RequestFormat {
		req.Header.Set(key, value)
	}

	resp, err := rf.client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// 응답 처리
	if resp.StatusCode == http.StatusOK {
		var result struct {
			Allow bool `json:"allow"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			return false
		}
		return result.Allow
	}

	return false
}

// ConditionFilter 구현
func (cf *ConditionFilter) Process(c *gin.Context) bool {
	for _, condition := range cf.Conditions {
		if !evaluateCondition(c, condition) {
			return false
		}
	}
	return true
}

func evaluateCondition(c *gin.Context, condition Condition) bool {
	var fieldValue string

	// 필드 값 가져오기
	switch condition.Field {
	case "header":
		fieldValue = c.GetHeader(condition.Value.(string))
	case "param":
		fieldValue = c.Param(condition.Value.(string))
	case "query":
		fieldValue = c.Query(condition.Value.(string))
	default:
		return false
	}

	// 조건 평가
	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value.(string)
	case "contains":
		return strings.Contains(fieldValue, condition.Value.(string))
	case "prefix":
		return strings.HasPrefix(fieldValue, condition.Value.(string))
	case "suffix":
		return strings.HasSuffix(fieldValue, condition.Value.(string))
	default:
		return false
	}
}

// 라우터 설정 함수
func setupRouter(config Config) *gin.Engine {
	router := gin.New()

	for _, route := range config.Routes {
		handlers := make([]gin.HandlerFunc, 0)

		// RequestFilter 처리
		for _, rf := range route.RequestFilters {
			handlers = append(handlers, createRequestFilterMiddleware(&rf))
		}

		// ConditionFilter 처리
		if route.ConditionFilter != nil {
			handlers = append(handlers, createConditionFilterMiddleware(route.ConditionFilter))
		}

		if route.Proxy != nil {
			handlers = append(handlers, createProxyHandler(route.Proxy))
		}

		router.Handle(route.Method, route.Path, handlers...)
	}

	return router
}

// Middleware 생성 함수들
func createRequestFilterMiddleware(filter *RequestFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !filter.Process(c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func createConditionFilterMiddleware(filter *ConditionFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !filter.Process(c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
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
