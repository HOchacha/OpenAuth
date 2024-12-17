package _

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("filter")

type Filter interface {
	Process(c *gin.Context) bool
}

type RequestFilter struct {
	RemoteServer  string            `yaml:"remote_server"`
	RequestFormat map[string]string `yaml:"request_format"`
	FieldsToSend  []string          `yaml:"fields_to_send"`
	client        *http.Client
}

type ConditionFilter struct {
	Conditions []Condition `yaml:"conditions"`
}

type Condition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

type Config struct {
	Routes []RouteConfig `yaml:"routes"`
}

type RouteConfig struct {
	Path            string           `yaml:"path"`
	Method          string           `yaml:"method"`
	RequestFilters  []RequestFilter  `yaml:"request_filters"`
	ConditionFilter *ConditionFilter `yaml:"condition_filter,omitempty"`
	HandlerType     string           `yaml:"handler_type"`
}

func (rf *RequestFilter) Process(c *gin.Context) bool {
	if rf.client == nil {
		log.Debug("Initializing HTTP client")
		rf.client = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		}
	}

	log.Info("=== RequestFilter Process Start ===")
	log.Debugf("RemoteServer: %s", rf.RemoteServer)

	// 1. 요청 본문을 읽음
	var requestBody map[string]interface{}
	rawData, err := c.GetRawData()
	if err != nil {
		log.Errorf("Error reading raw request body: %v", err)
		return false
	}

	// 2. 본문을 다시 Context에 설정
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData))

	// 3. JSON 파싱
	if err := json.Unmarshal(rawData, &requestBody); err != nil {
		log.Errorf("Error parsing request body: %v", err)
		return false
	}
	log.Debugf("Received request body: %+v", requestBody)

	// 4. 다음 미들웨어를 위해 본문 다시 설정
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData))

	// 지정된 필드만 선택하여 새로운 맵 생성 -> 새로운 Request Body
	filteredBody := make(map[string]interface{})
	for _, field := range rf.FieldsToSend {
		if value, exists := requestBody[field]; exists {
			filteredBody[field] = value
		}
	}
	log.Debugf("Fields to send: %v", rf.FieldsToSend)
	log.Debugf("Filtered request body: %+v", filteredBody)

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
	log.Debugf("Request headers: %+v", req.Header)

	resp, err := rf.client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body: %v", err)
			return false
		}
		log.Debugf("Raw response body: %s", string(body))

		// 실제 응답 구조에 맞게 수정
		var result struct {
			Message string `json:"message"`
			OTP     int    `json:"otp"`
		}

		// body를 다시 읽을 수 있도록 새로운 Reader 생성
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Errorf("Failed to decode response: %v", err)
			return false
		}

		// 200 OK와 message가 있으면 성공으로 처리
		log.Debugf("Decoded message: %s, OTP: %d", result.Message, result.OTP)
		return true // StatusOK이면 성공으로 처리
	}

	log.Debugf("Response from FilterServer: %+v\n", resp)

	log.Warning("Request not allowed (non-200 status code)")
	log.Info("=== RequestFilter Process End ===")

	return false
}

// ConditionFilter 구현
// ConditionFilter에서는 HTTP 헤더, HTTP Param, Path Query String에 따라서
// 동일한지, 포함하고 있는지, 접두사-접미사가 동일한지 검사한다.
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

func SetupRouter(config Config) *gin.Engine {
	router := gin.New()

	for _, route := range config.Routes {
		handlers := make([]gin.HandlerFunc, 0)

		// RequestFilter 처리
		for _, rf := range route.RequestFilters {
			handlers = append(handlers, CreateRequestFilterMiddleware(&rf))
		}

		// ConditionFilter 처리
		if route.ConditionFilter != nil {
			handlers = append(handlers, CreateConditionFilterMiddleware(route.ConditionFilter))
		}

		router.Handle(route.Method, route.Path, handlers...)
	}

	return router
}

func CreateRequestFilterMiddleware(filter *RequestFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info("Starting RequestFilter middleware")

		if filter.Process(c) {
			log.Info("Filter passed - continuing to next middleware")
			c.Next()
		} else {
			log.Warning("Filter failed - returning 403")
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

func CreateConditionFilterMiddleware(filter *ConditionFilter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !filter.Process(c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
