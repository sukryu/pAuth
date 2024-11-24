package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/mocks"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuthController_CreateUser(t *testing.T) {
	tests := []struct {
		name      string
		user      *v1alpha1.User
		setupMock func(*mocks.MockStore)
		wantErr   error
	}{
		{
			name: "successful creation",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "password123",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("CreateUser", mock.Anything, mock.MatchedBy(func(user *v1alpha1.User) bool {
					// 필수 필드만 검증
					return user.Name == "testuser" &&
						user.Spec.Username == "testuser" &&
						user.Spec.Email == "test@example.com" &&
						user.TypeMeta.Kind == "User" &&
						user.TypeMeta.APIVersion == "auth.service/v1alpha1" &&
						user.Status.Active == true
				})).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "user already exists",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existinguser",
				},
				Spec: v1alpha1.UserSpec{
					Username:     "existinguser",
					Email:        "existing@example.com",
					PasswordHash: "password123",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("CreateUser", mock.Anything, mock.MatchedBy(func(user *v1alpha1.User) bool {
					return user.Name == "existinguser"
				})).Return(errors.ErrUserExists)
			},
			wantErr: errors.ErrUserExists,
		},
		{
			name: "empty username",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   errors.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			_, err := controller.CreateUser(context.Background(), tt.user)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestAuthController_Login(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		setupMock func(*mocks.MockStore)
		wantErr   error
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: "password123",
			setupMock: func(ms *mocks.MockStore) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username:     "testuser",
						Email:        "test@example.com",
						PasswordHash: string(hashedPassword),
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(user, nil)
				ms.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *v1alpha1.User) bool {
					return u.Name == "testuser" && u.Status.LastLogin != nil
				})).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password123",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetUser", mock.Anything, "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
			wantErr: errors.ErrInvalidCredentials,
		},
		{
			name:      "empty credentials",
			username:  "",
			password:  "",
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   errors.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			_, err := controller.Login(context.Background(), tt.username, tt.password)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			mockStore.AssertExpectations(t)
		})
	}
}
func TestAuthController_GetUser(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name:     "successful get",
			username: "testuser",
			setupMock: func(ms *mocks.MockStore) {
				user := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "test@example.com",
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(user, nil)
			},
			wantErr: "",
		},
		{
			name:     "user not found",
			username: "nonexistent",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetUser", mock.Anything, "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
			wantErr: "failed to get user: status 404: user not found", // 실제 에러 메시지와 정확히 일치하도록 수정
		},
		{
			name:      "empty username",
			username:  "",
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "user name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			_, err := controller.GetUser(context.Background(), tt.username)

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

func TestAuthController_UpdateUser(t *testing.T) {
	tests := []struct {
		name      string
		user      *v1alpha1.User
		setupMock func(*mocks.MockStore)
		wantErr   string
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
			setupMock: func(ms *mocks.MockStore) {
				existingUser := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "old@example.com",
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
				ms.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *v1alpha1.User) bool {
					return u.Name == "testuser" && u.Spec.Email == "updated@example.com"
				})).Return(nil)
			},
			wantErr: "",
		},
		{
			name: "user not found during get",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nonexistent",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetUser", mock.Anything, "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
			wantErr: "failed to get existing user: status 404: user not found", // 에러 메시지 수정
		},
		{
			name: "update fails",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username: "testuser",
					Email:    "updated@example.com",
				},
			},
			setupMock: func(ms *mocks.MockStore) {
				existingUser := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "old@example.com",
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
				ms.On("UpdateUser", mock.Anything, mock.Anything).Return(errors.ErrInternal)
			},
			wantErr: "failed to update user: status 500: internal server error",
		},
		{
			name: "empty username",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "",
				},
			},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "user name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			_, err := controller.UpdateUser(context.Background(), tt.user)

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
func TestAuthController_DeleteUser(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name:     "successful deletion",
			username: "testuser",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("DeleteUser", mock.Anything, "testuser").Return(nil)
			},
			wantErr: "",
		},
		{
			name:     "user not found",
			username: "nonexistent",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("DeleteUser", mock.Anything, "nonexistent").Return(errors.ErrUserNotFound)
			},
			wantErr: "failed to delete user: status 404: user not found",
		},
		{
			name:      "empty username",
			username:  "",
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "user name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			err := controller.DeleteUser(context.Background(), tt.username)

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

func TestAuthController_ListUsers(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockStore)
		wantLen   int
		wantErr   string
	}{
		{
			name: "successful list",
			setupMock: func(ms *mocks.MockStore) {
				userList := &v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "user1"},
							Spec:       v1alpha1.UserSpec{Username: "user1"},
						},
						{
							ObjectMeta: metav1.ObjectMeta{Name: "user2"},
							Spec:       v1alpha1.UserSpec{Username: "user2"},
						},
					},
				}
				ms.On("ListUsers", mock.Anything).Return(userList, nil)
			},
			wantLen: 2,
			wantErr: "",
		},
		{
			name: "empty list",
			setupMock: func(ms *mocks.MockStore) {
				userList := &v1alpha1.UserList{
					Items: []v1alpha1.User{},
				}
				ms.On("ListUsers", mock.Anything).Return(userList, nil)
			},
			wantLen: 0,
			wantErr: "",
		},
		{
			name: "store error",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("ListUsers", mock.Anything).Return(nil, errors.ErrInternal)
			},
			wantLen: 0,
			wantErr: "failed to list users: status 500: internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			users, err := controller.ListUsers(context.Background())

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.Nil(t, users)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, users)
				assert.Equal(t, tt.wantLen, len(users.Items))
			}
			mockStore.AssertExpectations(t)
		})
	}
}

func TestAuthController_ChangePassword(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		oldPassword string
		newPassword string
		setupMock   func(*mocks.MockStore)
		wantErr     string
	}{
		{
			name:        "successful password change",
			username:    "testuser",
			oldPassword: "oldpass123",
			newPassword: "newpass123",
			setupMock: func(ms *mocks.MockStore) {
				hashedOldPass, _ := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.DefaultCost)
				existingUser := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username:     "testuser",
						PasswordHash: string(hashedOldPass),
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
				ms.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *v1alpha1.User) bool {
					return u.Name == "testuser" && u.Spec.PasswordHash != string(hashedOldPass)
				})).Return(nil)
			},
			wantErr: "",
		},
		{
			name:        "user not found",
			username:    "nonexistent",
			oldPassword: "oldpass",
			newPassword: "newpass",
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetUser", mock.Anything, "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
			wantErr: "status 404: user not found",
		},
		{
			name:        "incorrect old password",
			username:    "testuser",
			oldPassword: "wrongpass",
			newPassword: "newpass123",
			setupMock: func(ms *mocks.MockStore) {
				hashedOldPass, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
				existingUser := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username:     "testuser",
						PasswordHash: string(hashedOldPass),
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
			},
			wantErr: "status 401: invalid credentials: invalid old password",
		},
		{
			name:        "empty username",
			username:    "",
			oldPassword: "oldpass",
			newPassword: "newpass",
			setupMock:   func(ms *mocks.MockStore) {},
			wantErr:     "status 400: invalid input: all fields are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			err := controller.ChangePassword(context.Background(), tt.username, tt.oldPassword, tt.newPassword)

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

func TestAuthController_AssignRoles(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		roles     []string
		setupMock func(*mocks.MockStore)
		wantErr   string
	}{
		{
			name:     "successful role assignment",
			username: "testuser",
			roles:    []string{"admin", "user"},
			setupMock: func(ms *mocks.MockStore) {
				existingUser := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Roles:    []string{"user"},
					},
				}
				ms.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
				ms.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *v1alpha1.User) bool {
					return u.Name == "testuser" &&
						len(u.Spec.Roles) == 2 &&
						containsString(u.Spec.Roles, "admin") &&
						containsString(u.Spec.Roles, "user")
				})).Return(nil)
			},
			wantErr: "",
		},
		{
			name:     "user not found",
			username: "nonexistent",
			roles:    []string{"admin"},
			setupMock: func(ms *mocks.MockStore) {
				ms.On("GetUser", mock.Anything, "nonexistent").Return(nil, errors.ErrUserNotFound)
			},
			wantErr: "status 404: user not found",
		},
		{
			name:      "empty username",
			username:  "",
			roles:     []string{"admin"},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: name and roles are required",
		},
		{
			name:      "empty roles",
			username:  "testuser",
			roles:     []string{},
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   "status 400: invalid input: name and roles are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			err := controller.AssignRoles(context.Background(), tt.username, tt.roles)

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

// 헬퍼 함수
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
