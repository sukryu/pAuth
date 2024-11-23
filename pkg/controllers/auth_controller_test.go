package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/mocks"
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
				ms.ExpectCreateUser(&v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "test@example.com",
					},
				}, nil)
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
				ms.ExpectCreateUser(&v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existinguser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "existinguser",
						Email:    "existing@example.com",
					},
				}, errors.ErrUserExists)
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
		{
			name: "empty password",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testuser",
				},
				Spec: v1alpha1.UserSpec{
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "",
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
				user := &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testuser",
					},
					Spec: v1alpha1.UserSpec{
						Username: "testuser",
						Email:    "test@example.com",
					},
				}
				ms.ExpectGetUser("testuser", user, nil)
				ms.ExpectUpdateUser(user, nil)
			},
			wantErr: nil,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password123",
			setupMock: func(ms *mocks.MockStore) {
				ms.ExpectGetUser("nonexistent", nil, errors.ErrUserNotFound)
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
		wantErr   error
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
				ms.ExpectGetUser("testuser", user, nil)
			},
			wantErr: nil,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			setupMock: func(ms *mocks.MockStore) {
				ms.ExpectGetUser("nonexistent", nil, errors.ErrUserNotFound)
			},
			wantErr: errors.ErrUserNotFound,
		},
		{
			name:      "empty username",
			username:  "",
			setupMock: func(ms *mocks.MockStore) {},
			wantErr:   errors.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := mocks.NewMockStore()
			tt.setupMock(mockStore)

			controller := NewAuthController(mockStore)
			_, err := controller.GetUser(context.Background(), tt.username)

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
