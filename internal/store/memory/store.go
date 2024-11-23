package memory

import (
	"context"
	"sync"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
)

type memoryStore struct {
	mu           sync.RWMutex
	users        map[string]*v1alpha1.User
	roles        map[string]*v1alpha1.Role
	roleBindings map[string]*v1alpha1.RoleBinding
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{
		users:        make(map[string]*v1alpha1.User),
		roles:        make(map[string]*v1alpha1.Role),
		roleBindings: make(map[string]*v1alpha1.RoleBinding),
	}
}

// User operations
func (s *memoryStore) CreateUser(ctx context.Context, user *v1alpha1.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Name]; exists {
		return errors.ErrUserExists.WithReason(user.Name)
	}

	s.users[user.Name] = user.DeepCopy()
	return nil
}

func (s *memoryStore) GetUser(ctx context.Context, name string) (*v1alpha1.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[name]
	if !exists {
		return nil, errors.ErrUserNotFound.WithReason(name)
	}

	return user.DeepCopy(), nil
}

func (s *memoryStore) UpdateUser(ctx context.Context, user *v1alpha1.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Name]; !exists {
		return errors.ErrUserNotFound.WithReason(user.Name)
	}

	s.users[user.Name] = user.DeepCopy()
	return nil
}

func (s *memoryStore) DeleteUser(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; !exists {
		return errors.ErrUserNotFound.WithReason(name)
	}

	delete(s.users, name)
	return nil
}

func (s *memoryStore) ListUsers(ctx context.Context) (*v1alpha1.UserList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.UserList{
		Items: make([]v1alpha1.User, 0, len(s.users)),
	}

	for _, user := range s.users {
		list.Items = append(list.Items, *user.DeepCopy())
	}

	return list, nil
}

// Role operations
func (s *memoryStore) CreateRole(ctx context.Context, role *v1alpha1.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[role.Name]; exists {
		return errors.ErrRoleExists.WithReason(role.Name)
	}

	s.roles[role.Name] = role.DeepCopy()
	return nil
}

func (s *memoryStore) GetRole(ctx context.Context, name string) (*v1alpha1.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, exists := s.roles[name]
	if !exists {
		return nil, errors.ErrRoleNotFound.WithReason(name)
	}

	return role.DeepCopy(), nil
}

func (s *memoryStore) UpdateRole(ctx context.Context, role *v1alpha1.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[role.Name]; !exists {
		return errors.ErrRoleNotFound.WithReason(role.Name)
	}

	s.roles[role.Name] = role.DeepCopy()
	return nil
}

func (s *memoryStore) DeleteRole(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[name]; !exists {
		return errors.ErrRoleNotFound.WithReason(name)
	}

	delete(s.roles, name)
	return nil
}

func (s *memoryStore) ListRoles(ctx context.Context) ([]*v1alpha1.Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*v1alpha1.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role.DeepCopy())
	}

	return roles, nil
}

// RoleBinding operations
func (s *memoryStore) CreateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roleBindings[binding.Name]; exists {
		return errors.NewStatusError(409, "role binding already exists").WithReason(binding.Name)
	}

	s.roleBindings[binding.Name] = binding.DeepCopy()
	return nil
}

func (s *memoryStore) GetRoleBinding(ctx context.Context, name string) (*v1alpha1.RoleBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	binding, exists := s.roleBindings[name]
	if !exists {
		return nil, errors.NewStatusError(404, "role binding not found").WithReason(name)
	}

	return binding.DeepCopy(), nil
}

func (s *memoryStore) UpdateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roleBindings[binding.Name]; !exists {
		return errors.NewStatusError(404, "role binding not found").WithReason(binding.Name)
	}

	s.roleBindings[binding.Name] = binding.DeepCopy()
	return nil
}

func (s *memoryStore) DeleteRoleBinding(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roleBindings[name]; !exists {
		return errors.NewStatusError(404, "role binding not found").WithReason(name)
	}

	delete(s.roleBindings, name)
	return nil
}

func (s *memoryStore) ListRoleBindings(ctx context.Context) ([]*v1alpha1.RoleBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bindings := make([]*v1alpha1.RoleBinding, 0, len(s.roleBindings))
	for _, binding := range s.roleBindings {
		bindings = append(bindings, binding.DeepCopy())
	}

	return bindings, nil
}
