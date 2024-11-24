package interfaces

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

type RoleStore interface {
	Create(ctx context.Context, role *v1alpha1.Role) error
	Get(ctx context.Context, name string) (*v1alpha1.Role, error)
	Update(ctx context.Context, role *v1alpha1.Role) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*v1alpha1.Role, error)
}
