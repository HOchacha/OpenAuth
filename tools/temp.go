package configDispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

// FilterContext는 필터 체인을 통해 전달되는 컨텍스트
type FilterContext struct {
	*gin.Context
	CollectedData map[string]interface{}
}

// Filter 인터페이스 정의
type Filter interface {
	Process(fc *FilterContext) error
}

// RequestFilter 구조체 정의
type RequestFilter struct {
	RemoteServer  string            `yaml:"remote_server"`
	RequestFormat map[string]string `yaml:"request_format"`
	FieldsToSend  []string          `yaml:"fields_to_send"`
	DataMapping   DataMapping       `yaml:"data_mapping"`
	client        *http.Client
}

// DataMapping은 외부 서버 응답에서 어떤 필드를 어떻게 저장할지 정의
type DataMapping struct {
	Fields map[string]string `yaml:"fields"` // 외부 응답 필드 -> 내부 저장 키
	Target string            `yaml:"target"` // 데이터를 저장할 최상위 키 (예: "auth", "otp" 등)
}

// RouteConfig 구조체 수정
type RouteConfig struct {
	Path           string          `yaml:"path"`
	Method         string          `yaml:"method"`
	RequestFilters []RequestFilter `yaml:"request_filters"`
	JWT            *JWTConfig      `yaml:"jwt"`
}

// JWTConfig 구조체 정의
type JWTConfig struct {
	SecretKey string        `yaml:"secret_key"`
	Expires   time.Duration `yaml:"expires"`
	Claims    []string      `yaml:"claims"` // JWT에 포함할 CollectedData의 키들
}

// RequestFilter Process 메서드 구현
func (rf *RequestFilter) Process(fc *FilterContext) error {
	if rf.client == nil {
		rf.client = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	// 1. 요청 본문 준비
	requestBody := make(map[string]interface{})

	// 이전 필터에서 수집된 데이터 활용
	for _, field := range rf.FieldsToSend {
		// gin.Context에서 먼저 찾기
		if value, exists := fc.Get(field); exists {
			requestBody[field] = value
			continue
		}
		// CollectedData에서 찾기
		for _, data := range fc.CollectedData {
			if m, ok := data.(map[string]interface{}); ok {
				if value, exists := m[field]; exists {
					requestBody[field] = value
					break
				}
			}
		}
	}

	// 2. 외부 서버로 요청 전송
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", rf.RemoteServer, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	for key, value := range rf.RequestFormat {
		req.Header.Set(key, value)
	}

	resp, err := rf.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// 3. 응답 처리 및 데이터 저장
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	// 매핑 규칙에 따라 데이터 저장
	mappedData := make(map[string]interface{})
	for responseKey, targetKey := range rf.DataMapping.Fields {
		if value, exists := responseData[responseKey]; exists {
			mappedData[targetKey] = value
		}
	}

	// CollectedData에 저장
	fc.CollectedData[rf.DataMapping.Target] = mappedData

	return nil
}

// JWT 토큰 생성 함수
func generateJWTToken(collectedData map[string]interface{}, config *JWTConfig) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(config.Expires).Unix(),
		"iat": time.Now().Unix(),
	}

	// 지정된 claims만 JWT에 포함
	for _, claimKey := range config.Claims {
		if data, exists := collectedData[claimKey]; exists {
			claims[claimKey] = data
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

// 필터 체인 미들웨어 생성
func createFilterChain(filters []Filter, jwtConfig *JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		fc := &FilterContext{
			Context:       c,
			CollectedData: make(map[string]interface{}),
		}

		// 모든 필터 실행
		for _, filter := range filters {
			if err := filter.Process(fc); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// JWT 토큰 생성
		if jwtConfig != nil {
			token, err := generateJWTToken(fc.CollectedData, jwtConfig)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}
			c.Header("Authorization", "Bearer "+token)
		}

		// CollectedData를 컨텍스트에 저장
		c.Set("collected_data", fc.CollectedData)
		c.Next()
	}
}
