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
