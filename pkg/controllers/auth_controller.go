package controllers

import (
	"context"
	"fmt"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthController defines authentication operations
type AuthController interface {
	CreateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error)
	GetUser(ctx context.Context, name string) (*v1alpha1.User, error)
	UpdateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error)
	DeleteUser(ctx context.Context, name string) error
	ListUsers(ctx context.Context) (*v1alpha1.UserList, error)
	Login(ctx context.Context, username, password string) (*v1alpha1.User, error)
	ChangePassword(ctx context.Context, name, oldPassword, newPassword string) error
	AssignRoles(ctx context.Context, name string, roles []string) error
	ValidateToken(ctx context.Context, token string) (*v1alpha1.User, error)
}

type authController struct {
	store Store
}

func NewAuthController(store Store) AuthController {
	return &authController{
		store: store,
	}
}

func (c *authController) CreateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error) {
	if user.ObjectMeta.Name == "" {
		return nil, fmt.Errorf("user name cannot be empty")
	}

	if user.Spec.PasswordHash == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Spec.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	user.Spec.PasswordHash = string(hashedPassword)

	// Set TypeMeta
	user.TypeMeta = metav1.TypeMeta{
		APIVersion: "auth.service/v1alpha1",
		Kind:       "User",
	}

	// Set status
	user.Status = v1alpha1.UserStatus{
		Active: true,
	}

	// Set metadata
	now := metav1.Now()
	user.ObjectMeta.CreationTimestamp = now

	// Store the user
	err = c.store.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return user, nil
}

func (c *authController) GetUser(ctx context.Context, name string) (*v1alpha1.User, error) {
	if name == "" {
		return nil, fmt.Errorf("user name cannot be empty")
	}

	user, err := c.store.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return user, nil
}

func (c *authController) UpdateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error) {
	if user.ObjectMeta.Name == "" {
		return nil, fmt.Errorf("user name cannot be empty")
	}

	existing, err := c.store.Get(ctx, user.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing user: %v", err)
	}

	// Preserve password hash and creation timestamp
	user.Spec.PasswordHash = existing.Spec.PasswordHash
	user.ObjectMeta.CreationTimestamp = existing.ObjectMeta.CreationTimestamp

	err = c.store.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	return user, nil
}

func (c *authController) DeleteUser(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("user name cannot be empty")
	}

	err := c.store.Delete(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

func (c *authController) ListUsers(ctx context.Context) (*v1alpha1.UserList, error) {
	users, err := c.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %v", err)
	}

	return users, nil
}

func (c *authController) Login(ctx context.Context, username, password string) (*v1alpha1.User, error) {
	user, err := c.store.Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login time
	now := metav1.Now()
	user.Status.LastLogin = &now

	err = c.store.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update last login time: %v", err)
	}

	return user, nil
}

func (c *authController) ChangePassword(ctx context.Context, name, oldPassword, newPassword string) error {
	user, err := c.store.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.PasswordHash), []byte(oldPassword))
	if err != nil {
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %v", err)
	}

	user.Spec.PasswordHash = string(hashedPassword)

	err = c.store.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}

	return nil
}

func (c *authController) AssignRoles(ctx context.Context, name string, roles []string) error {
	user, err := c.store.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	user.Spec.Roles = roles

	err = c.store.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to assign roles: %v", err)
	}

	return nil
}

func (c *authController) ValidateToken(ctx context.Context, token string) (*v1alpha1.User, error) {
	// TODO: Implement JWT token validation
	return nil, fmt.Errorf("not implemented")
}
