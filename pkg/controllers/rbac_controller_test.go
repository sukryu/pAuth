package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRBACController_CheckAccess(t *testing.T) {
	tests := []struct {
		name      string
		user      *v1alpha1.User
		verb      string
		resource  string
		apiGroup  string
		setupMock func(*mocks.MockStore)
		want      bool
		wantErr   string
	}{
		{
			name: "admin has full access",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin-user",
				},
			},
			verb:     "create",
			resource: "users",
			apiGroup: "auth.service",
			setupMock: func(ms *mocks.MockStore) {
				binding := &v1alpha1.RoleBinding{
					Subjects: []v1alpha1.Subject{{
						Kind: "User",
						Name: "admin-user",
					}},
					RoleRef: v1alpha1.RoleRef{
						Kind: "Role",
						Name: "admin",
					},
				}
				role := &v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "admin"},
					Rules: []v1alpha1.PolicyRule{{
						Verbs:     []string{"*"},
						Resources: []string{"*"},
						APIGroups: []string{"*"},
					}},
				}
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{binding}, nil)
				ms.On("GetRole", mock.Anything, "admin").Return(role, nil)
			},
			want:    true,
			wantErr: "",
		},
		{
			name: "user with specific permissions",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
			},
			verb:     "get",
			resource: "users",
			apiGroup: "auth.service",
			setupMock: func(ms *mocks.MockStore) {
				binding := &v1alpha1.RoleBinding{
					Subjects: []v1alpha1.Subject{{
						Kind: "User",
						Name: "test-user",
					}},
					RoleRef: v1alpha1.RoleRef{
						Kind: "Role",
						Name: "reader",
					},
				}
				role := &v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "reader"},
					Rules: []v1alpha1.PolicyRule{{
						Verbs:     []string{"get", "list"},
						Resources: []string{"users"},
						APIGroups: []string{"auth.service"},
					}},
				}
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{binding}, nil)
				ms.On("GetRole", mock.Anything, "reader").Return(role, nil)
			},
			want:    true,
			wantErr: "",
		},
		{
			name: "unauthorized access",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
			},
			verb:     "delete",
			resource: "users",
			apiGroup: "auth.service",
			setupMock: func(ms *mocks.MockStore) {
				binding := &v1alpha1.RoleBinding{
					Subjects: []v1alpha1.Subject{{
						Kind: "User",
						Name: "test-user",
					}},
					RoleRef: v1alpha1.RoleRef{
						Kind: "Role",
						Name: "reader",
					},
				}
				role := &v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "reader"},
					Rules: []v1alpha1.PolicyRule{{
						Verbs:     []string{"get", "list"},
						Resources: []string{"users"},
						APIGroups: []string{"auth.service"},
					}},
				}
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{binding}, nil)
				ms.On("GetRole", mock.Anything, "reader").Return(role, nil)
			},
			want:    false,
			wantErr: "",
		},
		{
			name: "error listing role bindings",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
			},
			verb:     "get",
			resource: "users",
			apiGroup: "auth.service",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListRoleBindings", mock.Anything).Return(nil, errors.ErrInternal)
			},
			want:    false,
			wantErr: "status 500: internal server error: failed to list role bindings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			got, err := controller.CheckAccess(context.Background(), tt.user, tt.verb, tt.resource, tt.apiGroup)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_CreateRole(t *testing.T) {
	tests := []struct {
		name      string
		role      *v1alpha1.Role
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name: "successful role creation",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				Rules: []v1alpha1.PolicyRule{{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				}},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("CreateRole", mock.Anything, mock.MatchedBy(func(r *v1alpha1.Role) bool {
					return r.Name == "admin"
				})).Return(nil)
			},
			wantErr: "",
		},
		{
			name:      "nil role",
			role:      nil,
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: role cannot be nil",
		},
		{
			name: "empty role name",
			role: &v1alpha1.Role{
				Rules: []v1alpha1.PolicyRule{{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				}},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: role name is required",
		},
		{
			name: "no rules",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: at least one rule is required",
		},
		{
			name: "empty verbs in rule",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				Rules: []v1alpha1.PolicyRule{{
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				}},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: verbs are required in rule 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			err := controller.CreateRole(context.Background(), tt.role)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_DeleteRole(t *testing.T) {
	tests := []struct {
		name      string
		roleName  string
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name:     "successful deletion",
			roleName: "test-role",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "test-role").Return(&v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				}, nil)
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{}, nil)
				ms.On("DeleteRole", mock.Anything, "test-role").Return(nil)
			},
			wantErr: "",
		},
		{
			name:     "role is referenced by binding",
			roleName: "test-role",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "test-role").Return(&v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "test-role"},
				}, nil)
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
						RoleRef:    v1alpha1.RoleRef{Name: "test-role"},
					},
				}, nil)
			},
			wantErr: "status 400: invalid input: role test-role is still referenced by role binding test-binding",
		},
		{
			name:      "empty role name",
			roleName:  "",
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: role name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			err := controller.DeleteRole(context.Background(), tt.roleName)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_GetRole(t *testing.T) {
	tests := []struct {
		name      string
		roleName  string
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name:     "successful get role",
			roleName: "admin",
			setupMock: func(ms *mocks.MockStore) {
				role := &v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name: "admin",
					},
					Rules: []v1alpha1.PolicyRule{{
						Verbs:     []string{"*"},
						Resources: []string{"*"},
						APIGroups: []string{"*"},
					}},
				}
				ms.On("GetRole", mock.Anything, "admin").Return(role, nil)
			},
			wantErr: "",
		},
		{
			name:     "role not found",
			roleName: "nonexistent",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "nonexistent").Return(nil, errors.ErrRoleNotFound)
			},
			wantErr: "status 404: role not found",
		},
		{
			name:     "empty role name",
			roleName: "",
			setupMock: func(ms *mocks.MockStore) {
				// empty role name이므로 store.GetRole이 호출되지 않음
			},
			wantErr: "status 400: invalid input: role name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			role, err := controller.GetRole(context.Background(), tt.roleName)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.Nil(t, role)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, tt.roleName, role.Name)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_ListRoles(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockStore)
		wantLen   int
		wantErr   string
	}{
		{
			name: "successful list roles",
			setupMock: func(ms *mocks.MockStore) {
				roles := []*v1alpha1.Role{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "admin"},
						Rules: []v1alpha1.PolicyRule{{
							Verbs:     []string{"*"},
							Resources: []string{"*"},
							APIGroups: []string{"*"},
						}},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "user"},
						Rules: []v1alpha1.PolicyRule{{
							Verbs:     []string{"get", "list"},
							Resources: []string{"users"},
							APIGroups: []string{"auth.service"},
						}},
					},
				}
				ms.On("ListRoles", mock.Anything).Return(roles, nil)
			},
			wantLen: 2,
			wantErr: "",
		},
		{
			name: "empty list",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListRoles", mock.Anything).Return([]*v1alpha1.Role{}, nil)
			},
			wantLen: 0,
			wantErr: "",
		},
		{
			name: "internal error",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListRoles", mock.Anything).Return(nil, errors.ErrInternal)
			},
			wantLen: 0,
			wantErr: "status 500: internal server error: failed to list role bindings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			roles, err := controller.ListRoles(context.Background())

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantLen, len(roles))
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_CreateRoleBinding(t *testing.T) {
	tests := []struct {
		name        string
		roleBinding *v1alpha1.RoleBinding
		setupMock   func(*mocks.MockStore)
		wantErr     string
	}{
		{
			name: "successful role binding creation",
			roleBinding: &v1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin-binding",
				},
				Subjects: []v1alpha1.Subject{{
					Kind: "User",
					Name: "admin-user",
				}},
				RoleRef: v1alpha1.RoleRef{
					Kind: "Role",
					Name: "admin",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "admin").Return(&v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "admin"},
				}, nil)
				ms.On("CreateRoleBinding", mock.Anything, mock.MatchedBy(func(rb *v1alpha1.RoleBinding) bool {
					return rb.Name == "admin-binding" && rb.RoleRef.Name == "admin"
				})).Return(nil)
			},
			wantErr: "",
		},
		{
			name: "role not found",
			roleBinding: &v1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Subjects: []v1alpha1.Subject{{
					Kind: "User",
					Name: "test-user",
				}},
				RoleRef: v1alpha1.RoleRef{
					Kind: "Role",
					Name: "nonexistent",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "nonexistent").Return(nil, errors.ErrRoleNotFound)
			},
			wantErr: "status 404: role not found",
		},
		{
			name: "role binding already exists",
			roleBinding: &v1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-binding",
				},
				Subjects: []v1alpha1.Subject{{
					Kind: "User",
					Name: "test-user",
				}},
				RoleRef: v1alpha1.RoleRef{
					Kind: "Role",
					Name: "admin",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRole", mock.Anything, "admin").Return(&v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "admin"},
				}, nil)
				ms.On("CreateRoleBinding", mock.Anything, mock.Anything).Return(errors.ErrRoleBindingExists)
			},
			wantErr: "status 409: role binding already exists",
		},
		{
			name:        "nil role binding",
			roleBinding: nil,
			setupMock:   func(ms *mocks.MockStore) {},
			wantErr:     "status 400: invalid input: role binding cannot be nil",
		},
		{
			name: "empty role binding name",
			roleBinding: &v1alpha1.RoleBinding{
				Subjects: []v1alpha1.Subject{{
					Kind: "User",
					Name: "test-user",
				}},
				RoleRef: v1alpha1.RoleRef{
					Kind: "Role",
					Name: "admin",
				},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: role binding name is required",
		},
		{
			name: "empty role ref name",
			roleBinding: &v1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Subjects: []v1alpha1.Subject{{
					Kind: "User",
					Name: "test-user",
				}},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: role reference name is required",
		},
		{
			name: "no subjects",
			roleBinding: &v1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				RoleRef: v1alpha1.RoleRef{
					Kind: "Role",
					Name: "admin",
				},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: at least one subject is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			err := controller.CreateRoleBinding(context.Background(), tt.roleBinding)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_ListRoleBindings(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockStore)
		wantLen   int
		wantErr   string
	}{
		{
			name: "successful list",
			setupMock: func(ms *mocks.MockStore) {
				bindings := []*v1alpha1.RoleBinding{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "binding1"},
						Subjects: []v1alpha1.Subject{{
							Kind: "User",
							Name: "user1",
						}},
						RoleRef: v1alpha1.RoleRef{
							Kind: "Role",
							Name: "role1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "binding2"},
						Subjects: []v1alpha1.Subject{{
							Kind: "User",
							Name: "user2",
						}},
						RoleRef: v1alpha1.RoleRef{
							Kind: "Role",
							Name: "role2",
						},
					},
				}
				ms.On("ListRoleBindings", mock.Anything).Return(bindings, nil)
			},
			wantLen: 2,
			wantErr: "",
		},
		{
			name: "empty list",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListRoleBindings", mock.Anything).Return([]*v1alpha1.RoleBinding{}, nil)
			},
			wantLen: 0,
			wantErr: "",
		},
		{
			name: "store error",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListRoleBindings", mock.Anything).Return(nil, errors.ErrInternal)
			},
			wantLen: 0,
			wantErr: "status 500: internal server error: failed to list role bindings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			bindings, err := controller.ListRoleBindings(context.Background())

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.Nil(t, bindings)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bindings)
				assert.Equal(t, tt.wantLen, len(bindings))
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestRBACController_DeleteRoleBinding(t *testing.T) {
	tests := []struct {
		name        string
		bindingName string
		setupMock   func(*mocks.MockStore)
		wantErr     string
	}{
		{
			name:        "successful deletion",
			bindingName: "test-binding",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRoleBinding", mock.Anything, "test-binding").Return(&v1alpha1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
				}, nil)
				ms.On("DeleteRoleBinding", mock.Anything, "test-binding").Return(nil)
			},
			wantErr: "",
		},
		{
			name:        "empty binding name",
			bindingName: "",
			setupMock:   func(ms *mocks.MockStore) {},
			wantErr:     "status 400: invalid input: role binding name is required",
		},
		{
			name:        "binding not found",
			bindingName: "nonexistent",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRoleBinding", mock.Anything, "nonexistent").Return(nil, errors.ErrRoleBindingNotFound)
			},
			wantErr: "status 404: role binding not found",
		},
		{
			name:        "deletion error",
			bindingName: "test-binding",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetRoleBinding", mock.Anything, "test-binding").Return(&v1alpha1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
				}, nil)
				ms.On("DeleteRoleBinding", mock.Anything, "test-binding").Return(errors.ErrInternal)
			},
			wantErr: "status 500: internal server error: failed to list role bindings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewRBACController(mockStore)
			err := controller.DeleteRoleBinding(context.Background(), tt.bindingName)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}
