package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/pkg/apis/handlers"
	"github.com/sukryu/pAuth/pkg/controllers"
	"github.com/sukryu/pAuth/pkg/middleware"
	"github.com/sukryu/pAuth/pkg/utils/jwt"
)

type Router struct {
	authHandler    *handlers.AuthHandler
	jwtManager     *jwt.JWTManager
	rbacController controllers.RBACController
}

func NewRouter(
	authHandler *handlers.AuthHandler,
	jwtManager *jwt.JWTManager,
	rbacController controllers.RBACController,
) *Router {
	return &Router{
		authHandler:    authHandler,
		jwtManager:     jwtManager,
		rbacController: rbacController,
	}
}

func (r *Router) Setup() *gin.Engine {
	router := gin.Default()

	// 에러 핸들링 미들웨어
	router.Use(middleware.ErrorMiddleware())

	// Public routes
	public := router.Group("/api/v1/auth")
	{
		public.POST("/login", r.authHandler.Login)
		public.POST("/users", r.authHandler.CreateUser)
	}

	// Protected routes
	protected := router.Group("/api/v1/auth")
	protected.Use(middleware.JWTAuth(r.jwtManager))
	protected.Use(middleware.RBACMiddleware(r.rbacController))
	{
		protected.GET("/users/:name", r.authHandler.GetUser)
		protected.PUT("/users/:name", r.authHandler.UpdateUser)
		protected.DELETE("/users/:name", r.authHandler.DeleteUser)
		protected.GET("/users", r.authHandler.ListUsers)
		protected.PUT("/users/:name/password", r.authHandler.ChangePassword)
		protected.PUT("/users/:name/roles", r.authHandler.AssignRoles)

		// RBAC 관련 라우트
		protected.POST("/roles", r.authHandler.CreateRole)
		protected.GET("/roles", r.authHandler.ListRoles)
		protected.GET("/roles/:name", r.authHandler.GetRole)
		protected.DELETE("/roles/:name", r.authHandler.DeleteRole)

		protected.POST("/rolebindings", r.authHandler.CreateRoleBinding)
		protected.GET("/rolebindings", r.authHandler.ListRoleBindings)
		protected.GET("/rolebindings/:name", r.authHandler.GetRoleBinding)
		protected.DELETE("/rolebindings/:name", r.authHandler.DeleteRoleBinding)
	}

	return router
}
