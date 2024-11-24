package user

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	sqliteManager "github.com/sukryu/pAuth/pkg/store/sqlite"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// 임시 데이터베이스 파일 생성
	tempFile, err := os.CreateTemp("", "test-*.db")
	require.NoError(t, err)

	// 데이터베이스 연결
	db, err := gorm.Open(sqlite.Open(tempFile.Name()), &gorm.Config{})
	require.NoError(t, err)

	// 테이블 매니저 초기화
	tm := sqliteManager.NewTableManager(db)
	err = tm.Initialize(context.Background())
	require.NoError(t, err)

	// 클린업 함수 반환
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		os.Remove(tempFile.Name())
	}

	return db, cleanup
}

func TestUserStore_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		user    *v1alpha1.User
		wantErr bool
	}{
		{
			name: "valid user",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserSpec{
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: "hashed_password",
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate user",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			user: &v1alpha1.User{
				Spec: v1alpha1.UserSpec{
					Username: "testuser2",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Create(context.Background(), tt.user)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserStore_Get(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	// 테스트 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		},
	}
	err = store.Create(context.Background(), user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		userID  string
		want    *v1alpha1.User
		wantErr bool
	}{
		{
			name:    "existing user",
			userID:  "test-user",
			want:    user,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			userID:  "non-existent",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Get(context.Background(), tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Spec.Username, got.Spec.Username)
			assert.Equal(t, tt.want.Spec.Email, got.Spec.Email)
		})
	}
}

func TestUserStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	// 테스트 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		},
	}
	err = store.Create(context.Background(), user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		user    *v1alpha1.User
		wantErr bool
	}{
		{
			name: "valid update",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserSpec{
					Username:     "updated-user",
					Email:        "updated@example.com",
					PasswordHash: "updated_password",
				},
			},
			wantErr: false,
		},
		{
			name: "non-existent user",
			user: &v1alpha1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "non-existent",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Update(context.Background(), tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify update
			if !tt.wantErr {
				updated, err := store.Get(context.Background(), tt.user.Name)
				assert.NoError(t, err)
				assert.Equal(t, tt.user.Spec.Username, updated.Spec.Username)
				assert.Equal(t, tt.user.Spec.Email, updated.Spec.Email)
			}
		})
	}
}

func TestUserStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	// 테스트 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1alpha1.UserSpec{
			Username: "testuser",
		},
	}
	err = store.Create(context.Background(), user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "existing user",
			userID:  "test-user",
			wantErr: false,
		},
		{
			name:    "non-existent user",
			userID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Delete(context.Background(), tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify deletion
				_, err := store.Get(context.Background(), tt.userID)
				assert.Error(t, err)
			}
		})
	}
}

func TestUserStore_FindByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	// 테스트 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
		},
	}
	err = store.Create(context.Background(), user)
	require.NoError(t, err)

	tests := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{
			name:    "existing email",
			email:   "test@example.com",
			want:    "test-user",
			wantErr: false,
		},
		{
			name:    "non-existent email",
			email:   "nonexistent@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.FindByEmail(context.Background(), tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got.Name)
		})
	}
}

func TestUserStore_ListByRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store, err := NewStore(Config{
		DatabaseType: "sqlite",
		DB:           db,
	})
	require.NoError(t, err)

	// Create test users and role bindings
	users := []*v1alpha1.User{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user1",
			},
			Spec: v1alpha1.UserSpec{
				Username: "user1",
				Email:    "user1@example.com",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user2",
			},
			Spec: v1alpha1.UserSpec{
				Username: "user2",
				Email:    "user2@example.com",
			},
		},
	}

	for _, u := range users {
		err := store.Create(context.Background(), u)
		require.NoError(t, err)
	}

	// Create role bindings directly in the database
	roleBindings := []struct {
		Name        string
		RoleRef     string
		SubjectName string
	}{
		{
			Name:        "binding1",
			RoleRef:     "admin",
			SubjectName: "user1",
		},
		{
			Name:        "binding2",
			RoleRef:     "viewer",
			SubjectName: "user2",
		},
	}

	for _, rb := range roleBindings {
		err := db.Exec(`
			INSERT INTO role_bindings (name, role_ref, subject_name, subject_kind)
			VALUES (?, ?, ?, ?)
		`, rb.Name, rb.RoleRef, rb.SubjectName, "User").Error
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		roleName string
		want     int
		wantErr  bool
	}{
		{
			name:     "admin role",
			roleName: "admin",
			want:     1,
			wantErr:  false,
		},
		{
			name:     "viewer role",
			roleName: "viewer",
			want:     1,
			wantErr:  false,
		},
		{
			name:     "non-existent role",
			roleName: "non-existent",
			want:     0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.ListByRole(context.Background(), tt.roleName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, len(got.Items))
		})
	}
}
