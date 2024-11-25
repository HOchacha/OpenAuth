package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTManager_GenerateToken(t *testing.T) {
	// Setup
	manager := NewJWTManager("test-secret", 1*time.Hour)

	tests := []struct {
		name    string
		userID  string
		email   string
		wantErr bool
	}{
		{
			name:    "Valid token generation",
			userID:  "user123",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "Empty userID",
			userID:  "",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "Empty email",
			userID:  "user123",
			email:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token
			token, err := manager.GenerateToken(tt.userID, tt.email)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify token format
			parts := strings.Split(token, ".")
			assert.Equal(t, 3, len(parts), "Token should have three parts")

			// Parse and verify token contents
			parsedToken, err := jwt.ParseWithClaims(
				token,
				&CustomClaims{},
				func(token *jwt.Token) (interface{}, error) {
					return []byte("test-secret"), nil
				},
			)
			assert.NoError(t, err)

			claims, ok := parsedToken.Claims.(*CustomClaims)
			assert.True(t, ok)
			assert.Equal(t, tt.userID, claims.Subject)
			assert.Equal(t, tt.email, claims.Email)
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 1*time.Hour)

	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		checkFunc func(*CustomClaims) bool
	}{
		{
			name: "Valid token",
			setupFunc: func() string {
				token, _ := manager.GenerateToken("user123", "test@example.com")
				return token
			},
			wantErr: false,
			checkFunc: func(claims *CustomClaims) bool {
				return claims.Subject == "user123" &&
					claims.Email == "test@example.com"
			},
		},
		{
			name: "Expired token",
			setupFunc: func() string { // set wrong time
				expiredManager := NewJWTManager("test-secret", -1*time.Hour)
				token, _ := expiredManager.GenerateToken("user123", "test@example.com")
				return token
			},
			wantErr: true,
		},
		{
			name: "Invalid signature",
			setupFunc: func() string { //
				wrongManager := NewJWTManager("wrong-secret", 1*time.Hour)
				token, _ := wrongManager.GenerateToken("user123", "test@example.com")
				return token
			},
			wantErr: true,
		},
		{
			name: "Malformed token",
			setupFunc: func() string {
				return "malformed.token.here"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupFunc()
			claims, err := manager.ValidateToken(token)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				assert.True(t, tt.checkFunc(claims))
			}
		})
	}
}

// TestJWTManager_TokenExpiry tests token expiration logic
func TestJWTManager_TokenExpiry(t *testing.T) {
	// Setup manager with very short expiry
	shortManager := NewJWTManager("test-secret", 1*time.Second)

	// Generate token
	token, err := shortManager.GenerateToken("user123", "test@example.com")
	assert.NoError(t, err)

	// Verify token is initially valid
	claims, err := shortManager.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Verify token is now invalid
	_, err = shortManager.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token has expired")
}

// BenchmarkJWTManager_GenerateToken benchmarks token generation
func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	manager := NewJWTManager("test-secret", 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GenerateToken("user123", "test@example.com")
		if err != nil {
			b.Fatal(err)
		}
	}
}
