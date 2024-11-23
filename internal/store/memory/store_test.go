package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMemoryStore_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		user    *v1alpha1.User
		setup   func(*memoryStore)
		wantErr error
	}{
		{
			name: "successful creation",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username: "testuser",
					Email:    "test@example.com",
				},
			},
			setup:   func(s *memoryStore) {},
			wantErr: nil,
		},
		{
			name: "duplicate user",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existinguser",
				},
			},
			setup: func(s *memoryStore) {
				s.users["existinguser"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existinguser",
					},
				}
			},
			wantErr: errors.ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			err := store.CreateUser(context.Background(), tt.user)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				storedUser, exists := store.users[tt.user.Name]
				assert.True(t, exists)
				assert.Equal(t, tt.user.Name, storedUser.Name)
			}
		})
	}
}

func TestMemoryStore_GetUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		setup    func(*memoryStore)
		want     *v1alpha1.User
		wantErr  error
	}{
		{
			name:     "existing user",
			username: "testuser",
			setup: func(s *memoryStore) {
				s.users["testuser"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "test@example.com",
					},
				}
			},
			want: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username: "testuser",
					Email:    "test@example.com",
				},
			},
			wantErr: nil,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			setup:    func(s *memoryStore) {},
			want:     nil,
			wantErr:  errors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			got, err := store.GetUser(context.Background(), tt.username)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Name, got.Name)
				assert.Equal(t, tt.want.Spec.Username, got.Spec.Username)
				assert.Equal(t, tt.want.Spec.Email, got.Spec.Email)
			}
		})
	}
}

func TestMemoryStore_CreateRole(t *testing.T) {
	tests := []struct {
		name    string
		role    *v1alpha1.Role
		setup   func(*memoryStore)
		wantErr error
	}{
		{
			name: "successful creation",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "admin",
				},
				Rules: []v1alpha1.PolicyRule{
					{
						Verbs:     []string{"*"},
						Resources: []string{"*"},
						APIGroups: []string{"*"},
					},
				},
			},
			setup:   func(s *memoryStore) {},
			wantErr: nil,
		},
		{
			name: "duplicate role",
			role: &v1alpha1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-role",
				},
			},
			setup: func(s *memoryStore) {
				s.roles["existing-role"] = &v1alpha1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-role",
					},
				}
			},
			wantErr: errors.ErrRoleExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			err := store.CreateRole(context.Background(), tt.role)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				storedRole, exists := store.roles[tt.role.Name]
				assert.True(t, exists)
				assert.Equal(t, tt.role.Name, storedRole.Name)
				assert.Equal(t, len(tt.role.Rules), len(storedRole.Rules))
			}
		})
	}
}

func TestMemoryStore_DeleteUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		setup    func(*memoryStore)
		wantErr  error
	}{
		{
			name:     "existing user",
			username: "testuser",
			setup: func(s *memoryStore) {
				s.users["testuser"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
				}
			},
			wantErr: nil,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			setup:    func(s *memoryStore) {},
			wantErr:  errors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			err := store.DeleteUser(context.Background(), tt.username)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				_, exists := store.users[tt.username]
				assert.False(t, exists)
			}
		})
	}
}

func TestMemoryStore_UpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		user    *v1alpha1.User
		setup   func(*memoryStore)
		wantErr error
	}{
		{
			name: "successful update",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username: "testuser",
					Email:    "updated@example.com",
				},
			},
			setup: func(s *memoryStore) {
				s.users["testuser"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "original@example.com",
					},
				}
			},
			wantErr: nil,
		},
		{
			name: "update non-existent user",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nonexistent",
				},
			},
			setup:   func(s *memoryStore) {},
			wantErr: errors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			err := store.UpdateUser(context.Background(), tt.user)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				updatedUser := store.users[tt.user.Name]
				assert.Equal(t, tt.user.Spec.Email, updatedUser.Spec.Email)
			}
		})
	}
}

func TestMemoryStore_ListUsers(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memoryStore)
		wantLen int
		wantErr error
	}{
		{
			name: "list multiple users",
			setup: func(s *memoryStore) {
				s.users["user1"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{Name: "user1"},
				}
				s.users["user2"] = &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{Name: "user2"},
				}
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:    "empty list",
			setup:   func(s *memoryStore) {},
			wantLen: 0,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			tt.setup(store)

			users, err := store.ListUsers(context.Background())

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantLen, len(users.Items))
			}
		})
	}
}

func TestMemoryStore_RoleOperations(t *testing.T) {
	t.Run("CRUD operations for Role", func(t *testing.T) {
		store := NewMemoryStore()
		ctx := context.Background()

		// Create
		role := &v1alpha1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			},
			Rules: []v1alpha1.PolicyRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				},
			},
		}

		err := store.CreateRole(ctx, role)
		assert.NoError(t, err)

		// Get
		retrieved, err := store.GetRole(ctx, "admin")
		assert.NoError(t, err)
		assert.Equal(t, role.Name, retrieved.Name)
		assert.Equal(t, len(role.Rules), len(retrieved.Rules))

		// Update
		role.Rules[0].Verbs = []string{"get", "list"}
		err = store.UpdateRole(ctx, role)
		assert.NoError(t, err)

		// Verify Update
		updated, err := store.GetRole(ctx, "admin")
		assert.NoError(t, err)
		assert.Equal(t, []string{"get", "list"}, updated.Rules[0].Verbs)

		// Delete
		err = store.DeleteRole(ctx, "admin")
		assert.NoError(t, err)

		// Verify Deletion
		_, err = store.GetRole(ctx, "admin")
		assert.Error(t, err)
	})
}

func TestMemoryStore_RoleBindingOperations(t *testing.T) {
	t.Run("CRUD operations for RoleBinding", func(t *testing.T) {
		store := NewMemoryStore()
		ctx := context.Background()

		// Create
		binding := &v1alpha1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin-binding",
			},
			Subjects: []v1alpha1.Subject{
				{
					Kind: "User",
					Name: "admin-user",
				},
			},
			RoleRef: v1alpha1.RoleRef{
				Kind: "Role",
				Name: "admin",
			},
		}

		err := store.CreateRoleBinding(ctx, binding)
		assert.NoError(t, err)

		// Get
		retrieved, err := store.GetRoleBinding(ctx, "admin-binding")
		assert.NoError(t, err)
		assert.Equal(t, binding.Name, retrieved.Name)
		assert.Equal(t, binding.Subjects[0].Name, retrieved.Subjects[0].Name)

		// Update
		binding.Subjects = append(binding.Subjects, v1alpha1.Subject{
			Kind: "User",
			Name: "second-admin",
		})
		err = store.UpdateRoleBinding(ctx, binding)
		assert.NoError(t, err)

		// Verify Update
		updated, err := store.GetRoleBinding(ctx, "admin-binding")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(updated.Subjects))

		// List
		bindings, err := store.ListRoleBindings(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(bindings))

		// Delete
		err = store.DeleteRoleBinding(ctx, "admin-binding")
		assert.NoError(t, err)

		// Verify Deletion
		_, err = store.GetRoleBinding(ctx, "admin-binding")
		assert.Error(t, err)
	})
}

func TestMemoryStore_ConcurrentOperations(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Concurrent user operations
	t.Run("concurrent user operations", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				user := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("user-%d", id),
					},
				}
				store.CreateUser(ctx, user)
				store.GetUser(ctx, user.Name)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		users, err := store.ListUsers(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 10, len(users.Items))
	})
}
