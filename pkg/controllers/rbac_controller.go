package controllers

import (
	"context"
	"fmt"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
)

type RBACController interface {
	CreateRole(ctx context.Context, role *v1alpha1.Role) error
	GetRole(ctx context.Context, name string) (*v1alpha1.Role, error)
	ListRoles(ctx context.Context) ([]*v1alpha1.Role, error)
	DeleteRole(ctx context.Context, name string) error

	CreateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error
	GetRoleBinding(ctx context.Context, name string) (*v1alpha1.RoleBinding, error)
	ListRoleBindings(ctx context.Context) ([]*v1alpha1.RoleBinding, error)
	DeleteRoleBinding(ctx context.Context, name string) error

	CheckAccess(ctx context.Context, user *v1alpha1.User, verb, resource, apiGroup string) (bool, error)
}

type rbacController struct {
	store Store
}

func NewRBACController(store Store) RBACController {
	return &rbacController{store: store}
}

func (c *rbacController) CreateRole(ctx context.Context, role *v1alpha1.Role) error {
	if role == nil {
		return errors.ErrInvalidInput.WithReason("role cannot be nil")
	}
	if role.Name == "" {
		return errors.ErrInvalidInput.WithReason("role name is required")
	}
	if len(role.Rules) == 0 {
		return errors.ErrInvalidInput.WithReason("at least one rule is required")
	}

	// 각 rule의 유효성 검사
	for i, rule := range role.Rules {
		if len(rule.Verbs) == 0 {
			return errors.ErrInvalidInput.WithReason(fmt.Sprintf("verbs are required in rule %d", i))
		}
		if len(rule.Resources) == 0 {
			return errors.ErrInvalidInput.WithReason(fmt.Sprintf("resources are required in rule %d", i))
		}
		if len(rule.APIGroups) == 0 {
			return errors.ErrInvalidInput.WithReason(fmt.Sprintf("apiGroups are required in rule %d", i))
		}
	}

	return c.store.CreateRole(ctx, role)
}

func (c *rbacController) GetRole(ctx context.Context, name string) (*v1alpha1.Role, error) {
	if name == "" {
		return nil, errors.ErrInvalidInput.WithReason("role name is required")
	}

	role, err := c.store.GetRole(ctx, name)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (c *rbacController) ListRoles(ctx context.Context) ([]*v1alpha1.Role, error) {
	return c.store.ListRoles(ctx)
}

func (c *rbacController) DeleteRole(ctx context.Context, name string) error {
	if name == "" {
		return errors.ErrInvalidInput.WithReason("role name is required")
	}

	// Role이 존재하는지 확인
	_, err := c.store.GetRole(ctx, name)
	if err != nil {
		return err
	}

	// 이 Role을 참조하는 RoleBinding이 있는지 확인
	bindings, err := c.store.ListRoleBindings(ctx)
	if err != nil {
		return errors.ErrInternal.WithReason("failed to list role bindings")
	}

	for _, binding := range bindings {
		if binding.RoleRef.Name == name {
			return errors.ErrInvalidInput.WithReason(fmt.Sprintf("role %s is still referenced by role binding %s", name, binding.Name))
		}
	}

	return c.store.DeleteRole(ctx, name)
}

func (c *rbacController) CreateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	if binding == nil {
		return errors.ErrInvalidInput.WithReason("role binding cannot be nil")
	}
	if binding.Name == "" {
		return errors.ErrInvalidInput.WithReason("role binding name is required")
	}
	if binding.RoleRef.Name == "" {
		return errors.ErrInvalidInput.WithReason("role reference name is required")
	}
	if len(binding.Subjects) == 0 {
		return errors.ErrInvalidInput.WithReason("at least one subject is required")
	}

	// 참조된 Role이 존재하는지 확인
	_, err := c.store.GetRole(ctx, binding.RoleRef.Name)
	if err != nil {
		return err
	}

	return c.store.CreateRoleBinding(ctx, binding)
}

func (c *rbacController) GetRoleBinding(ctx context.Context, name string) (*v1alpha1.RoleBinding, error) {
	if name == "" {
		return nil, errors.ErrInvalidInput.WithReason("role binding name is required")
	}

	return c.store.GetRoleBinding(ctx, name)
}

func (c *rbacController) ListRoleBindings(ctx context.Context) ([]*v1alpha1.RoleBinding, error) {
	bindings, err := c.store.ListRoleBindings(ctx)
	if err != nil {
		return nil, errors.ErrInternal.WithReason("failed to list role bindings")
	}
	return bindings, nil
}

func (c *rbacController) DeleteRoleBinding(ctx context.Context, name string) error {
	if name == "" {
		return errors.ErrInvalidInput.WithReason("role binding name is required")
	}

	// RoleBinding이 존재하는지 확인
	_, err := c.store.GetRoleBinding(ctx, name)
	if err != nil {
		return err
	}

	return c.store.DeleteRoleBinding(ctx, name)
}

func (c *rbacController) CheckAccess(ctx context.Context, user *v1alpha1.User, verb, resource, apiGroup string) (bool, error) {
	if user == nil {
		return false, errors.ErrInvalidInput.WithReason("user cannot be nil")
	}

	bindings, err := c.store.ListRoleBindings(ctx)
	if err != nil {
		return false, errors.ErrInternal.WithReason("failed to list role bindings")
	}

	// Find user's role bindings
	userBindings := make([]*v1alpha1.RoleBinding, 0)
	for _, binding := range bindings {
		for _, subject := range binding.Subjects {
			if subject.Kind == "User" && subject.Name == user.Name {
				userBindings = append(userBindings, binding)
			}
		}
	}

	// Check permissions from each role
	for _, binding := range userBindings {
		role, err := c.store.GetRole(ctx, binding.RoleRef.Name)
		if err != nil {
			continue // Skip if role not found
		}

		// Check rules
		for _, rule := range role.Rules {
			// Check API Group
			if !contains(rule.APIGroups, apiGroup) && !contains(rule.APIGroups, "*") {
				continue
			}

			// Check Resource
			if !contains(rule.Resources, resource) && !contains(rule.Resources, "*") {
				continue
			}

			// Check Verb
			if contains(rule.Verbs, verb) || contains(rule.Verbs, "*") {
				return true, nil
			}
		}
	}

	return false, nil
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
