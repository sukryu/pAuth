package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/controllers"
)

type AuthHandler struct {
	controller controllers.AuthController
}

func NewAuthHandler(controller controllers.AuthController) *AuthHandler {
	return &AuthHandler{
		controller: controller,
	}
}

func (h *AuthHandler) Register(router *gin.Engine) {
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/users", h.createUser)
		auth.GET("/users/:name", h.getUser)
		auth.PUT("/users/:name", h.updateUser)
		auth.DELETE("/users/:name", h.deleteUser)
		auth.GET("/users", h.listUsers)
		auth.POST("/login", h.login)
		auth.PUT("/users/:name/password", h.changePassword)
		auth.PUT("/users/:name/roles", h.assignRoles)
	}
}

func (h *AuthHandler) createUser(c *gin.Context) {
	var user v1alpha1.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.controller.CreateUser(c.Request.Context(), &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.controller.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

func (h *AuthHandler) changePassword(c *gin.Context) {
	name := c.Param("name")
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.controller.ChangePassword(c.Request.Context(), name, req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

type assignRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

func (h *AuthHandler) assignRoles(c *gin.Context) {
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

func (h *AuthHandler) getUser(c *gin.Context) {
	name := c.Param("name")
	user, err := h.controller.GetUser(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) updateUser(c *gin.Context) {
	name := c.Param("name")
	var user v1alpha1.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.Name = name
	result, err := h.controller.UpdateUser(c.Request.Context(), &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) deleteUser(c *gin.Context) {
	name := c.Param("name")
	err := h.controller.DeleteUser(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) listUsers(c *gin.Context) {
	users, err := h.controller.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}
