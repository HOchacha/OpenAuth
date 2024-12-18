package configServer

import (
	"strconv"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestLoadJWTConfig(t *testing.T) {
	yamlData := `
routes:
  - path: "/signup"
    method: "POST"
    request_filters:
      - remote_server: "http://10.106.248.129/auth/signup"
        request_format:
          Content-Type: "application/json"
        fields_to_send: ["username", "password", "email", "role"]
    handler_type: "signup"

  - path: "/signin"
    method: "POST"
    request_filters:
      - remote_server: "http://10.106.248.129/auth/login"
        request_format:
          Content-Type: "application/json"
        fields_to_send: ["username", "password"]
    handler_type: "login"
    
  - path: "/verify"
    method: "POST"
    handler_type: "verify"

jwt_config:
  secret_key: "12345667"
  expiry: 86400      
  required_fields:
    - "username"
    - "role"
    - "email"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	expectedExpiry := 86400

	// 검증: SecretKey
	expectedSecret := "12345667"
	if config.JWTConfig.SecretKey != expectedSecret {
		t.Errorf("Expected SecretKey %s, got %s", expectedSecret, config.JWTConfig.SecretKey)
	}

	// 검증: Expiry
	configExpiry, err := strconv.Atoi(config.JWTConfig.Expiry)
	if err != nil {
		t.Fatalf("Failed to convert Expiry to int: %v", err)
	}
	if configExpiry != expectedExpiry {
		if configExpiry != expectedExpiry {
			t.Errorf("Expected Expiry %v, got %v", expectedExpiry, config.JWTConfig.Expiry)
		}
	}

	// 검증: RequiredFields
	expectedFields := []string{"username", "role", "email"}
	if len(config.JWTConfig.RequiredFields) != len(expectedFields) {
		t.Errorf("Expected RequiredFields length %d, got %d", len(expectedFields), len(config.JWTConfig.RequiredFields))
	}
	for i, field := range expectedFields {
		if config.JWTConfig.RequiredFields[i] != field {
			t.Errorf("Expected RequiredField[%d] = %s, got %s", i, field, config.JWTConfig.RequiredFields[i])
		}
	}

	// 추가 검증: Routes
	if len(config.Routes) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(config.Routes))
	}

	// 첫 번째 라우트 검증
	firstRoute := config.Routes[0]
	if firstRoute.Path != "/signup" || firstRoute.Method != "POST" || firstRoute.HandlerType != "signup" {
		t.Errorf("First route mismatch: %+v", firstRoute)
	}
	if len(firstRoute.RequestFilters) != 1 {
		t.Errorf("Expected 1 RequestFilter, got %d", len(firstRoute.RequestFilters))
	}
}
