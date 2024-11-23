package controllers

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

type Store interface {
	// User operations
	CreateUser(ctx context.Context, user *v1alpha1.User) error
	GetUser(ctx context.Context, name string) (*v1alpha1.User, error)
	UpdateUser(ctx context.Context, user *v1alpha1.User) error
	DeleteUser(ctx context.Context, name string) error
	ListUsers(ctx context.Context) (*v1alpha1.UserList, error)

	// Role operations
	CreateRole(ctx context.Context, role *v1alpha1.Role) error
	GetRole(ctx context.Context, name string) (*v1alpha1.Role, error)
	UpdateRole(ctx context.Context, role *v1alpha1.Role) error
	DeleteRole(ctx context.Context, name string) error
	ListRoles(ctx context.Context) ([]*v1alpha1.Role, error)

	// RoleBinding operations
	CreateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error
	GetRoleBinding(ctx context.Context, name string) (*v1alpha1.RoleBinding, error)
	UpdateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error
	DeleteRoleBinding(ctx context.Context, name string) error
	ListRoleBindings(ctx context.Context) ([]*v1alpha1.RoleBinding, error)
}
