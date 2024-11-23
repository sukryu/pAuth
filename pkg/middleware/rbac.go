package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RBACMiddleware(rbacController controllers.RBACController) gin.HandlerFunc {
	return func(c *gin.Context) {
		// JWT 미들웨어에서 설정한 사용자 정보 가져오기
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// 요청 정보 추출
		verb := getVerb(c.Request.Method)
		resource := getResource(c.FullPath())
		apiGroup := "auth.service"

		// 사용자 정보 가져오기
		user := &v1alpha1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: userID.(string),
			},
		}

		// 접근 권한 확인
		allowed, err := rbacController.CheckAccess(c.Request.Context(), user, verb, resource, apiGroup)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check access"})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func getVerb(method string) string {
	switch method {
	case "GET":
		return "get"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return ""
	}
}

func getResource(path string) string {
	// path에서 리소스 추출 로직 구현
	// 예: /api/v1/auth/users -> users
	return "users" // 실제 구현은 더 복잡할 수 있음
}
