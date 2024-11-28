package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetupTestDB(t *testing.T) (*sql.DB, *dynamic.DynamicStore) {
	// SQLite in-memory 데이터베이스 생성
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// 스키마 테이블 생성
	_, err = dbConn.Exec(`
        CREATE TABLE IF NOT EXISTS entity_schemas (
            id TEXT PRIMARY KEY,
            name TEXT UNIQUE NOT NULL,
            description TEXT,
            fields TEXT NOT NULL,
            indexes TEXT,
			annotations TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP
        )
    `)
	if err != nil {
		t.Fatalf("failed to create schema table: %v", err)
	}

	// users 테이블 스키마 생성
	_, err = dbConn.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            roles TEXT,
            is_active BOOLEAN DEFAULT true,
            last_login TIMESTAMP,
			annotations TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            deleted_at TIMESTAMP
        )
    `)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	// DynamicStore 생성
	dynStore := dynamic.NewDynamicStore(dbConn)

	return dbConn, dynStore
}

func setupTestStore(t *testing.T) (*Store, func()) {
	dbConn, dynStore := SetupTestDB(t)
	store := &Store{
		dynamicStore: dynStore,
		config:       Config{DatabaseType: "sqlite"},
	}

	cleanup := func() {
		dbConn.Close()
	}

	return store, cleanup
}

func createTestUser(t *testing.T) *v1alpha1.User {
	return &v1alpha1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-user",
			CreationTimestamp: metav1.Now(),
			Annotations: map[string]string{
				"custom-field": "custom-value",
			},
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Roles:        []string{"role1", "role2"},
		},
		Status: v1alpha1.UserStatus{
			Active:    true,
			LastLogin: &metav1.Time{Time: time.Now()},
		},
	}
}

func TestUserStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Create basic user", func(t *testing.T) {
		user := createTestUser(t)
		user.Name = "basic-user" // 고유 Name 설정
		user.Spec.Username = "basicuser"
		user.Spec.Email = "basic@example.com"

		err := store.Create(ctx, user)
		assert.NoError(t, err)

		// Verify creation
		saved, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, user.Spec.Username, saved.Spec.Username)
		assert.Equal(t, user.Spec.Email, saved.Spec.Email)
	})

	t.Run("Create duplicate user", func(t *testing.T) {
		user := createTestUser(t)
		user.Name = "duplicate-user"
		user.Spec.Username = "duplicateuser"
		user.Spec.Email = "duplicate@example.com" // 고유 Email 설정

		err := store.Create(ctx, user)
		assert.NoError(t, err)

		// Try to create duplicate (same username)
		user.Spec.Email = "another@example.com" // 이메일만 변경
		err = store.Create(ctx, user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "UNIQUE constraint failed")
	})

	t.Run("Create user with custom fields", func(t *testing.T) {
		user := createTestUser(t)
		user.Name = "custom-user"
		user.Spec.Username = "customuser"
		user.Spec.Email = "custom@example.com" // 고유 Email 설정
		user.Annotations["department"] = "IT"

		err := store.Create(ctx, user)
		assert.NoError(t, err)

		// Verify creation
		saved, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, "IT", saved.Annotations["department"])
	})
}

func TestUserStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Get existing user", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		found, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, user.Spec.Username, found.Spec.Username)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent")
		assert.Error(t, err)
	})
}

func TestUserStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update basic info", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		user.Spec.Email = "updated@example.com"
		err = store.Update(ctx, user)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, "updated@example.com", updated.Spec.Email)
	})

	t.Run("Update custom fields", func(t *testing.T) {
		user := createTestUser(t)
		user.Name = "custom-update-user"
		user.Spec.Username = "customupdateuser"
		user.Spec.Email = "customupdate@example.com" // 고유한 이메일 설정
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		user.Annotations["department"] = "HR"
		err = store.Update(ctx, user)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, "HR", updated.Annotations["department"])
	})
}

func TestUserStore_FindByEmail(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Find existing email", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		found, err := store.FindByEmail(ctx, user.Spec.Email)
		assert.NoError(t, err)
		assert.Equal(t, user.Name, found.Name)
	})

	t.Run("Find non-existent email", func(t *testing.T) {
		_, err := store.FindByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
	})
}

func TestUserStore_FindByUsername(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Find existing username", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		found, err := store.FindByUsername(ctx, user.Spec.Username)
		assert.NoError(t, err)
		assert.Equal(t, user.Name, found.Name)
	})

	t.Run("Find non-existent username", func(t *testing.T) {
		_, err := store.FindByUsername(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

func TestUserStore_UpdatePassword(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update password", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		newHash := "new_hashed_password"
		err = store.UpdatePassword(ctx, user.Name, newHash)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.Equal(t, newHash, updated.Spec.PasswordHash)
	})
}

func TestUserStore_UpdateStatus(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update status", func(t *testing.T) {
		user := createTestUser(t)
		err := store.Create(ctx, user)
		assert.NoError(t, err)

		err = store.UpdateStatus(ctx, user.Name, false)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, user.Name)
		assert.NoError(t, err)
		assert.False(t, updated.Status.Active)
	})
}

func TestUserStore_ListByRole(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("List users by role", func(t *testing.T) {
		// Create users with different roles
		user1 := createTestUser(t)
		user1.Name = "user1"
		user1.Spec.Username = "user1"
		user1.Spec.Email = "user1@example.com" // 고유 Email 설정
		user1.Spec.Roles = []string{"admin"}
		err := store.Create(ctx, user1)
		assert.NoError(t, err)

		user2 := createTestUser(t)
		user2.Name = "user2"
		user2.Spec.Username = "user2"
		user2.Spec.Email = "user2@example.com" // 고유 Email 설정
		user2.Spec.Roles = []string{"user"}
		err = store.Create(ctx, user2)
		assert.NoError(t, err)

		// List users with admin role
		users, err := store.ListByRole(ctx, "admin")
		assert.NoError(t, err)
		assert.Len(t, users.Items, 1)
		assert.Equal(t, "user1", users.Items[0].Name)
	})

	t.Run("List users with non-existent role", func(t *testing.T) {
		users, err := store.ListByRole(ctx, "non-existent-role")
		assert.NoError(t, err)
		assert.Len(t, users.Items, 0)
	})
}
