package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
)

// MockStore는 Store 인터페이스를 구현하는 mock 객체입니다.
type MockStore struct {
	mock.Mock
}

// NewMockStore는 MockStore의 새 인스턴스를 생성합니다.
func NewMockStore() *MockStore {
	return &MockStore{}
}

// User 관련 메서드
func (m *MockStore) CreateUser(ctx context.Context, user *v1alpha1.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockStore) GetUser(ctx context.Context, name string) (*v1alpha1.User, error) {
	args := m.Called(ctx, name)
	if user, ok := args.Get(0).(*v1alpha1.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStore) UpdateUser(ctx context.Context, user *v1alpha1.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockStore) DeleteUser(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockStore) ListUsers(ctx context.Context) (*v1alpha1.UserList, error) {
	args := m.Called(ctx)
	if list, ok := args.Get(0).(*v1alpha1.UserList); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

// Role 관련 메서드
func (m *MockStore) CreateRole(ctx context.Context, role *v1alpha1.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockStore) GetRole(ctx context.Context, name string) (*v1alpha1.Role, error) {
	args := m.Called(ctx, name)
	if role, ok := args.Get(0).(*v1alpha1.Role); ok {
		return role, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStore) UpdateRole(ctx context.Context, role *v1alpha1.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockStore) DeleteRole(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockStore) ListRoles(ctx context.Context) ([]*v1alpha1.Role, error) {
	args := m.Called(ctx)
	if roles, ok := args.Get(0).([]*v1alpha1.Role); ok {
		return roles, args.Error(1)
	}
	return nil, args.Error(1)
}

// RoleBinding 관련 메서드
func (m *MockStore) CreateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockStore) GetRoleBinding(ctx context.Context, name string) (*v1alpha1.RoleBinding, error) {
	args := m.Called(ctx, name)
	if binding, ok := args.Get(0).(*v1alpha1.RoleBinding); ok {
		return binding, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStore) UpdateRoleBinding(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockStore) DeleteRoleBinding(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockStore) ListRoleBindings(ctx context.Context) ([]*v1alpha1.RoleBinding, error) {
	args := m.Called(ctx)
	if bindings, ok := args.Get(0).([]*v1alpha1.RoleBinding); ok {
		return bindings, args.Error(1)
	}
	return nil, args.Error(1)
}

// Helper 메서드들
func (m *MockStore) ExpectCreateUser(user *v1alpha1.User, err error) *mock.Call {
	return m.On("CreateUser", mock.Anything, user).Return(err)
}

func (m *MockStore) ExpectGetUser(name string, user *v1alpha1.User, err error) *mock.Call {
	return m.On("GetUser", mock.Anything, name).Return(user, err)
}

func (m *MockStore) ExpectUpdateUser(user *v1alpha1.User, err error) *mock.Call {
	return m.On("UpdateUser", mock.Anything, user).Return(err)
}

func (m *MockStore) ExpectDeleteUser(name string, err error) *mock.Call {
	return m.On("DeleteUser", mock.Anything, name).Return(err)
}

func (m *MockStore) ExpectListUsers(list *v1alpha1.UserList, err error) *mock.Call {
	return m.On("ListUsers", mock.Anything).Return(list, err)
}

func (m *MockStore) ExpectCreateRole(role *v1alpha1.Role, err error) *mock.Call {
	return m.On("CreateRole", mock.Anything, role).Return(err)
}

func (m *MockStore) ExpectGetRole(name string, role *v1alpha1.Role, err error) *mock.Call {
	return m.On("GetRole", mock.Anything, name).Return(role, err)
}

func (m *MockStore) ExpectUpdateRole(role *v1alpha1.Role, err error) *mock.Call {
	return m.On("UpdateRole", mock.Anything, role).Return(err)
}

func (m *MockStore) ExpectDeleteRole(name string, err error) *mock.Call {
	return m.On("DeleteRole", mock.Anything, name).Return(err)
}

func (m *MockStore) ExpectListRoles(roles []*v1alpha1.Role, err error) *mock.Call {
	return m.On("ListRoles", mock.Anything).Return(roles, err)
}

func (m *MockStore) ExpectCreateRoleBinding(binding *v1alpha1.RoleBinding, err error) *mock.Call {
	return m.On("CreateRoleBinding", mock.Anything, binding).Return(err)
}

func (m *MockStore) ExpectGetRoleBinding(name string, binding *v1alpha1.RoleBinding, err error) *mock.Call {
	return m.On("GetRoleBinding", mock.Anything, name).Return(binding, err)
}

func (m *MockStore) ExpectUpdateRoleBinding(binding *v1alpha1.RoleBinding, err error) *mock.Call {
	return m.On("UpdateRoleBinding", mock.Anything, binding).Return(err)
}

func (m *MockStore) ExpectDeleteRoleBinding(name string, err error) *mock.Call {
	return m.On("DeleteRoleBinding", mock.Anything, name).Return(err)
}

func (m *MockStore) ExpectListRoleBindings(bindings []*v1alpha1.RoleBinding, err error) *mock.Call {
	return m.On("ListRoleBindings", mock.Anything).Return(bindings, err)
}
