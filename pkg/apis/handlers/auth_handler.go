package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/controllers"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/utils/jwt"
)

type AuthHandler struct {
	controller     controllers.AuthController
	jwtManager     *jwt.JWTManager
	rbacController controllers.RBACController
}

func NewAuthHandler(controller controllers.AuthController, jwtManager *jwt.JWTManager, rbacController controllers.RBACController) *AuthHandler {
	return &AuthHandler{
		controller:     controller,
		jwtManager:     jwtManager,
		rbacController: rbacController,
	}
}

func (h *AuthHandler) Register(router *gin.Engine) {
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/users", h.CreateUser)
		auth.GET("/users/:name", h.GetUser)
		auth.PUT("/users/:name", h.UpdateUser)
		auth.DELETE("/users/:name", h.DeleteUser)
		auth.GET("/users", h.ListUsers)
		auth.POST("/login", h.Login)
		auth.PUT("/users/:name/password", h.ChangePassword)
		auth.PUT("/users/:name/roles", h.AssignRoles)
	}
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var user v1alpha1.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	result, err := h.controller.CreateUser(c.Request.Context(), &user)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token string         `json:"token"`
	User  *v1alpha1.User `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	user, err := h.controller.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.Error(err)
		return
	}

	// JWT 토큰 생성
	token, err := h.jwtManager.GenerateToken(user.Name, user.Spec.Roles)
	if err != nil {
		c.Error(errors.ErrInternal.WithReason("failed to generate token"))
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		Token: token,
		User:  user,
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	err := h.controller.ChangePassword(c.Request.Context(), name, req.OldPassword, req.NewPassword)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

type assignRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

func (h *AuthHandler) AssignRoles(c *gin.Context) {
	name := c.Param("name")
	var req assignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.controller.AssignRoles(c.Request.Context(), name, req.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	user, err := h.controller.GetUser(c.Request.Context(), name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	var user v1alpha1.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	user.Name = name
	result, err := h.controller.UpdateUser(c.Request.Context(), &user)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	err := h.controller.DeleteUser(c.Request.Context(), name)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.controller.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// RBAC 핸들러
func (h *AuthHandler) CreateRole(c *gin.Context) {
	var role v1alpha1.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	err := h.rbacController.CreateRole(c.Request.Context(), &role)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *AuthHandler) ListRoles(c *gin.Context) {
	roles, err := h.rbacController.ListRoles(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, roles)
}

func (h *AuthHandler) GetRole(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	role, err := h.rbacController.GetRole(c.Request.Context(), name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *AuthHandler) DeleteRole(c *gin.Context) {
	name := c.Param("name")
	err := h.rbacController.DeleteRole(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RoleBinding 핸들러
func (h *AuthHandler) CreateRoleBinding(c *gin.Context) {
	var binding v1alpha1.RoleBinding
	if err := c.ShouldBindJSON(&binding); err != nil {
		c.Error(errors.ErrInvalidInput.WithReason(err.Error()))
		return
	}

	err := h.rbacController.CreateRoleBinding(c.Request.Context(), &binding)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, binding)
}

func (h *AuthHandler) ListRoleBindings(c *gin.Context) {
	bindings, err := h.rbacController.ListRoleBindings(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, bindings)
}

func (h *AuthHandler) GetRoleBinding(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Error(errors.ErrInvalidInput.WithReason("name parameter is required"))
		return
	}

	binding, err := h.rbacController.GetRoleBinding(c.Request.Context(), name)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, binding)
}

func (h *AuthHandler) DeleteRoleBinding(c *gin.Context) {
	name := c.Param("name")
	err := h.rbacController.DeleteRoleBinding(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
