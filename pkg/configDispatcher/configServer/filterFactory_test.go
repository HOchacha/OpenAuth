package configServer

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestFilter_Process(t *testing.T) {
	// 테스트용 서버 설정
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 바디 확인
		var receivedBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&receivedBody)

		// Content-Type 헤더 확인
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// username과 password 필드 확인
		if username, ok := receivedBody["username"]; ok && username == "testuser" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"allow": true})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]bool{"allow": false})
		}
	}))
	defer testServer.Close()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		fieldsToSend   []string
		requestFormat  map[string]string
		expectedResult bool
	}{
		{
			name: "valid request with correct fields",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "testpass",
				"extra":    "field",
			},
			fieldsToSend: []string{"username", "password"},
			requestFormat: map[string]string{
				"Content-Type": "application/json",
			},
			expectedResult: true,
		},
		{
			name: "invalid username",
			requestBody: map[string]interface{}{
				"username": "wronguser",
				"password": "testpass",
			},
			fieldsToSend: []string{"username", "password"},
			requestFormat: map[string]string{
				"Content-Type": "application/json",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 테스트 컨텍스트 설정
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// 요청 바디 설정
			jsonBody, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			// RequestFilter 설정
			rf := &RequestFilter{
				RemoteServer:  testServer.URL,
				RequestFormat: tt.requestFormat,
				FieldsToSend:  tt.fieldsToSend,
			}

			// 테스트 실행
			result := rf.Process(c)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestConditionFilter_Process(t *testing.T) {
	tests := []struct {
		name       string
		conditions []Condition
		headers    map[string]string
		query      map[string]string
		params     map[string]string
		expected   bool
	}{
		{
			name: "matching header condition",
			conditions: []Condition{
				{
					Field:    "header",
					Operator: "equals",
					Value:    "X-Test-Header",
				},
			},
			headers: map[string]string{
				"X-Test-Header": "X-Test-Header",
			},
			expected: true,
		},
		{
			name: "non-matching header condition",
			conditions: []Condition{
				{
					Field:    "header",
					Operator: "equals",
					Value:    "X-Test-Header",
				},
			},
			headers: map[string]string{
				"X-Test-Header": "wrong-value",
			},
			expected: false,
		},
		{
			name: "matching query condition",
			conditions: []Condition{
				{
					Field:    "query",
					Operator: "contains",
					Value:    "test",
				},
			},
			query: map[string]string{
				"test": "value",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 테스트 컨텍스트 설정
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			// 헤더 설정
			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}

			// 쿼리 파라미터 설정
			if len(tt.query) > 0 {
				q := c.Request.URL.Query()
				for k, v := range tt.query {
					q.Add(k, v)
				}
				c.Request.URL.RawQuery = q.Encode()
			}

			// ConditionFilter 설정
			cf := &ConditionFilter{
				Conditions: tt.conditions,
			}

			// 테스트 실행
			result := cf.Process(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetupRouter(t *testing.T) {
	config := Config{
		Routes: []RouteConfig{
			{
				Path:   "/test",
				Method: "POST",
				RequestFilters: []RequestFilter{
					{
						RemoteServer: "http://localhost:8080",
						RequestFormat: map[string]string{
							"Content-Type": "application/json",
						},
						FieldsToSend: []string{"username", "password"},
					},
				},
				ConditionFilter: &ConditionFilter{
					Conditions: []Condition{
						{
							Field:    "header",
							Operator: "equals",
							Value:    "X-Test-Header",
						},
					},
				},
			},
		},
	}

	router := setupRouter(config)
	assert.NotNil(t, router)
}
