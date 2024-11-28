package interfaces

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

type RoleBindingStore interface {
	Create(ctx context.Context, binding *v1alpha1.RoleBinding) error
	Get(ctx context.Context, name string) (*v1alpha1.RoleBinding, error)
	Update(ctx context.Context, binding *v1alpha1.RoleBinding) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*v1alpha1.RoleBinding, error)

	FindBySubject(ctx context.Context, subjectKind, subjectName string) ([]*v1alpha1.RoleBinding, error)
	FindByRole(ctx context.Context, roleName string) ([]*v1alpha1.RoleBinding, error)
	AddSubject(ctx context.Context, name string, subject v1alpha1.Subject) error
	RemoveSubject(ctx context.Context, name string, subject v1alpha1.Subject) error
	//ListByNamespace(ctx context.Context, namespace string) ([]*v1alpha1.RoleBinding, error)
}
