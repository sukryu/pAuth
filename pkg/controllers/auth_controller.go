package controllers

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

// AuthController defines authentication operations
type AuthController interface {
	CreateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error)
	GetUser(ctx context.Context, name string) (*v1alpha1.User, error)
	UpdateUser(ctx context.Context, user *v1alpha1.User) (*v1alpha1.User, error)
	DeleteUser(ctx context.Context, name string) error
	ListUsers(ctx context.Context) (*v1alpha1.UserList, error)
}

type authController struct {
	store Store
}
