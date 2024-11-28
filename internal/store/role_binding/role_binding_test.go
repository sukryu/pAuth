package rolebinding

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupTestDB(t *testing.T) (*sql.DB, *dynamic.DynamicStore) {
	// Manager 설정
	manager, err := manager.NewSQLManager(manager.Config{
		Type: "sqlite3",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("failed to create SQLManager: %v", err)
	}

	// 데이터베이스 연결 가져오기
	dbConn := manager.GetDB()

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

	// role_bindings 테이블 생성
	_, err = dbConn.Exec(`
       CREATE TABLE IF NOT EXISTS role_bindings (
           id TEXT PRIMARY KEY,
           name TEXT UNIQUE NOT NULL,
           role_ref TEXT NOT NULL,
           subjects TEXT NOT NULL,
           annotations TEXT,
           created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
           updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
           deleted_at TIMESTAMP
       )
   `)
	if err != nil {
		t.Fatalf("failed to create role_bindings table: %v", err)
	}

	// DynamicStore 생성
	store, err := dynamic.NewDynamicStore(manager)
	if err != nil {
		t.Fatalf("failed to create dynamic store: %v", err)
	}

	return dbConn, store
}
func setupTestStore(t *testing.T) (*Store, func()) {
	dbConn, dynStore := setupTestDB(t)
	store := &Store{
		dynamicStore: dynStore,
		config:       Config{DatabaseType: "sqlite"},
	}

	cleanup := func() {
		dbConn.Close()
	}

	return store, cleanup
}

func createTestRoleBinding(t *testing.T) *v1alpha1.RoleBinding {
	return &v1alpha1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-binding",
			Annotations: map[string]string{
				"description": "Test RoleBinding",
			},
		},
		RoleRef: v1alpha1.RoleRef{
			Kind: "Role",
			Name: "test-role",
		},
		Subjects: []v1alpha1.Subject{
			{
				Kind: "User",
				Name: "test-user",
			},
		},
	}
}

func TestRoleBindingStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Create basic binding", func(t *testing.T) {
		binding := createTestRoleBinding(t)
		err := store.Create(ctx, binding)
		assert.NoError(t, err)

		saved, err := store.Get(ctx, binding.Name)
		assert.NoError(t, err)
		assert.Equal(t, binding.Name, saved.Name)
		assert.Equal(t, binding.RoleRef.Name, saved.RoleRef.Name)
		assert.Equal(t, binding.Subjects[0].Name, saved.Subjects[0].Name)
	})

	t.Run("Create duplicate binding", func(t *testing.T) {
		binding := createTestRoleBinding(t)
		binding.Name = "duplicate-binding"
		err := store.Create(ctx, binding)
		assert.NoError(t, err)

		err = store.Create(ctx, binding)
		assert.Error(t, err)
	})
}

func TestRoleBindingStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update subjects", func(t *testing.T) {
		binding := createTestRoleBinding(t)
		err := store.Create(ctx, binding)
		assert.NoError(t, err)

		newSubject := v1alpha1.Subject{
			Kind: "User",
			Name: "new-user",
		}
		binding.Subjects = append(binding.Subjects, newSubject)

		err = store.Update(ctx, binding)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, binding.Name)
		assert.NoError(t, err)
		assert.Len(t, updated.Subjects, 2)
		assert.Equal(t, "new-user", updated.Subjects[1].Name)
	})
}

func TestRoleBindingStore_FindBySubject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	binding1 := createTestRoleBinding(t)
	binding1.Name = "binding1"
	binding1.Subjects = []v1alpha1.Subject{{Kind: "User", Name: "user1"}}
	err := store.Create(ctx, binding1)
	assert.NoError(t, err)

	binding2 := createTestRoleBinding(t)
	binding2.Name = "binding2"
	binding2.Subjects = []v1alpha1.Subject{{Kind: "User", Name: "user2"}}
	err = store.Create(ctx, binding2)
	assert.NoError(t, err)

	t.Run("Find bindings by subject", func(t *testing.T) {
		bindings, err := store.FindBySubject(ctx, "User", "user1")
		assert.NoError(t, err)
		assert.Len(t, bindings, 1)
		assert.Equal(t, "binding1", bindings[0].Name)
	})
}

func TestRoleBindingStore_FindByRole(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	binding1 := createTestRoleBinding(t)
	binding1.Name = "binding1"
	binding1.RoleRef.Name = "role1"
	err := store.Create(ctx, binding1)
	assert.NoError(t, err)

	binding2 := createTestRoleBinding(t)
	binding2.Name = "binding2"
	binding2.RoleRef.Name = "role2"
	err = store.Create(ctx, binding2)
	assert.NoError(t, err)

	t.Run("Find bindings by role", func(t *testing.T) {
		bindings, err := store.FindByRole(ctx, "role1")
		assert.NoError(t, err)
		assert.Len(t, bindings, 1)
		assert.Equal(t, "binding1", bindings[0].Name)
	})
}

func TestRoleBindingStore_AddRemoveSubject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Add and remove subject", func(t *testing.T) {
		binding := createTestRoleBinding(t)
		err := store.Create(ctx, binding)
		assert.NoError(t, err)

		newSubject := v1alpha1.Subject{
			Kind: "User",
			Name: "new-user",
		}

		// Add subject
		err = store.AddSubject(ctx, binding.Name, newSubject)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, binding.Name)
		assert.NoError(t, err)
		assert.Len(t, updated.Subjects, 2)

		// Remove subject
		err = store.RemoveSubject(ctx, binding.Name, newSubject)
		assert.NoError(t, err)

		updated, err = store.Get(ctx, binding.Name)
		assert.NoError(t, err)
		assert.Len(t, updated.Subjects, 1)
	})

	t.Run("Add duplicate subject", func(t *testing.T) {
		binding := createTestRoleBinding(t)
		binding.Name = "duplicate-subject-binding"
		err := store.Create(ctx, binding)
		assert.NoError(t, err)

		err = store.AddSubject(ctx, binding.Name, binding.Subjects[0])
		assert.Error(t, err)
	})
}

// func TestRoleBindingStore_ListByNamespace(t *testing.T) {
// 	store, cleanup := setupTestStore(t)
// 	defer cleanup()
// 	ctx := context.Background()

// 	binding1 := createTestRoleBinding(t)
// 	binding1.Name = "binding1"
// 	binding1.Namespace = "namespace1"
// 	err := store.Create(ctx, binding1)
// 	assert.NoError(t, err)

// 	binding2 := createTestRoleBinding(t)
// 	binding2.Name = "binding2"
// 	binding2.Namespace = "namespace2"
// 	err = store.Create(ctx, binding2)
// 	assert.NoError(t, err)

// 	t.Run("List bindings by namespace", func(t *testing.T) {
// 		bindings, err := store.ListByNamespace(ctx, "namespace1")
// 		assert.NoError(t, err)
// 		assert.Len(t, bindings, 1)
// 		assert.Equal(t, "binding1", bindings[0].Name)
// 	})
// }
