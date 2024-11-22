package controllers

import (
	"context"

	"github.com/sukryu/auth-service/pkg/apis/auth/v1alpha1"
)

type Store interface {
	Create(ctx context.Context, user *v1alpha1.User) error
	Get(ctx context.Context, name string) (*v1alpha1.User, error)
	Update(ctx context.Context, user *v1alpha1.User) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) (*v1alpha1.UserList, error)
}
