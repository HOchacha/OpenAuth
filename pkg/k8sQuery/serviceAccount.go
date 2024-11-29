package k8sQuery

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServiceAccountConfig는 허용된 ServiceAccount 설정을 관리합니다
type ServiceAccountConfig struct {
	AllowedAccounts map[string][]string
}

// TokenValidator는 k8s 토큰 검증을 담당합니다
type TokenValidator struct {
	k8sClient *kubernetes.Clientset
	config    ServiceAccountConfig
}

/*
Package k8sQuery implements a Kubernetes ServiceAccount token validation system.

# Types
ServiceAccountConfig:
- Manages allowed ServiceAccount configurations
- Contains map of namespace to allowed service account names

TokenValidator:
- Handles Kubernetes token validation
- Maintains kubernetes client and service account configuration

AuthResponse:
- Represents authentication response structure
- Contains authorization status, username, and potential error messages

# Main Functions
- NewTokenValidator(saConfig): Creates new TokenValidator instance with given configuration
- ValidateToken(token): Validates ServiceAccount token and checks if it's allowed
- validateServiceAccountToken(token): Basic token validation without allowed account checking
- isAllowedServiceAccount(namespace, name): Checks if ServiceAccount is in allowed list
- UpdateAllowedAccounts(accounts): Updates the list of allowed service accounts

# Middleware
AuthMiddleware: (for gin-gonic)
- Gin middleware for ServiceAccount token authentication
- Validates Bearer tokens from Authorization header
- Adds authenticated username to request context

# Default Configuration
Default allowed service accounts:
- namespace: "default", accounts: ["oauth-configurator"]
- namespace: "kube-system", accounts: ["oauth-admin"]

The package provides token validation and authentication middleware for Kubernetes
ServiceAccounts, ensuring only allowed service accounts can access protected resources.
*/

// NewTokenValidator는 새로운 TokenValidator를 생성합니다
func NewTokenValidator(saConfig ServiceAccountConfig) (*TokenValidator, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	// 기본 설정이 없는 경우 기본값 설정
	if saConfig.AllowedAccounts == nil {
		saConfig.AllowedAccounts = map[string][]string{
			"default":     {"oauth-configurator"},
			"kube-system": {"oauth-admin"},
		}
	}

	return &TokenValidator{
		k8sClient: clientset,
		config:    saConfig,
	}, nil
}

func validateServiceAccountToken(token string) (bool, error) {
	// 클러스터 내부 설정 가져오기
	config, err := rest.InClusterConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get cluster config: %v", err)
	}

	// 쿠버네티스 클라이언트 생성
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, fmt.Errorf("failed to create client: %v", err)
	}

	// TokenReview 객체 생성
	tokenReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	// API 서버에 검증 요청
	result, err := clientset.AuthenticationV1().TokenReviews().Create(
		context.TODO(),
		tokenReview,
		metav1.CreateOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("token review failed: %v", err)
	}

	// 검증 결과 확인
	if !result.Status.Authenticated {
		return false, nil
	}

	// 추가적인 검증 (예: ServiceAccount인지 확인)
	if !strings.HasPrefix(result.Status.User.Username, "system:serviceaccount:") {
		return false, nil
	}

	return true, nil
}

// Check received serviceaccount is valid
func (tv *TokenValidator) ValidateToken(token string) (bool, string, error) {
	review := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := tv.k8sClient.AuthenticationV1().TokenReviews().Create(
		context.TODO(),
		review,
		metav1.CreateOptions{},
	)
	if err != nil {
		return false, "", fmt.Errorf("token review failed: %v", err)
	}

	if !result.Status.Authenticated {
		return false, "", nil
	}

	username := result.Status.User.Username
	if !strings.HasPrefix(username, "system:serviceaccount:") {
		return false, "", nil
	}

	parts := strings.Split(username, ":")
	if len(parts) != 4 {
		return false, "", nil
	}

	namespace := parts[2]
	serviceAccountName := parts[3]

	if !tv.isAllowedServiceAccount(namespace, serviceAccountName) {
		return false, "", nil
	}

	return true, username, nil
}

// isAllowedServiceAccount는 주어진 ServiceAccount가 허용되는지 확인합니다
func (tv *TokenValidator) isAllowedServiceAccount(namespace, name string) bool {
	if allowedNames, exists := tv.config.AllowedAccounts[namespace]; exists {
		for _, allowedName := range allowedNames {
			if name == allowedName {
				return true
			}
		}
	}
	return false
}

// represent serviceaccount authentication
type AuthResponse struct {
	Authorized bool   `json:"authorized"`
	Username   string `json:"username,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Middleware for gin-gonic middleware Authentication
// this server connect to k8s api-server
func AuthMiddleware(validator *TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Authorized: false,
				Error:      "invalid authorization header",
			})
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		valid, username, err := validator.ValidateToken(token)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, AuthResponse{
				Authorized: false,
				Error:      fmt.Sprintf("token validation failed: %v", err),
			})
			return
		}

		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Authorized: false,
				Error:      "invalid token or unauthorized service account",
			})
			return
		}

		// 인증 성공 시 사용자 정보를 컨텍스트에 저장
		c.Set("username", username)
		c.Next()
	}
}

func (tv *TokenValidator) UpdateAllowedAccounts(accounts map[string][]string) {
	tv.config.AllowedAccounts = accounts
}
