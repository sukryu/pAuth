package interfaces

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

type UserStore interface {
	Create(ctx context.Context, user *v1alpha1.User) error
	Get(ctx context.Context, name string) (*v1alpha1.User, error)
	Update(ctx context.Context, user *v1alpha1.User) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) (*v1alpha1.UserList, error)

	FindByEmail(ctx context.Context, email string) (*v1alpha1.User, error)
	FindByUsername(ctx context.Context, username string) (*v1alpha1.User, error)
	UpdatePassword(ctx context.Context, name string, hashedPassword string) error
	UpdateStatus(ctx context.Context, name string, active bool) error
	ListByRole(ctx context.Context, roleName string) (*v1alpha1.UserList, error)
}
