// FILE: filters/request_filter.go
package filters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("filter")

type RequestFilter struct {
	FilterName    string            `yaml:"filter_name"`
	RemoteServer  string            `yaml:"remote_server"`
	RequestFormat map[string]string `yaml:"request_format"`
	FieldsToSend  []string          `yaml:"fields_to_send"`
	client        *http.Client
}

func (rf *RequestFilter) Process(c *gin.Context) (bool, error) {
	log.Infof("start request filter: %s", rf.FilterName)
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
		return false, fmt.Errorf("error reading raw request body: %v", err)
	}

	// 2. 본문을 다시 Context에 설정
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawData))

	// 3. JSON 파싱
	if err := json.Unmarshal(rawData, &requestBody); err != nil {
		log.Errorf("Error parsing request body: %v", err)
		return false, fmt.Errorf("error parsing request body: %v", err)
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
		return false, fmt.Errorf("error marshaling filtered body: %v", err)
	}

	req, err := http.NewRequest("POST", rf.RemoteServer, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return false, fmt.Errorf("error creating request: %v", err)
	}

	for key, value := range rf.RequestFormat {
		req.Header.Set(key, value)
	}
	log.Debugf("Request headers: %+v", req.Header)

	resp, err := rf.client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return false, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body: %v", err)
			return false, fmt.Errorf("failed to read response body: %v", err)
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
			return false, fmt.Errorf("failed to decode response: %v", err)
		}

		return true, nil // StatusOK이면 성공으로 처리
	}

	log.Debugf("Response from FilterServer: %+v\n", resp)

	log.Warning("Request not allowed (non-200 status code)")
	log.Info("=== RequestFilter Process End ===")

	return false, fmt.Errorf("request not allowed (non-200 status code)")
}
