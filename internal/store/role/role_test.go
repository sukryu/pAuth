package role

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetupTestDB(t *testing.T) (*sql.DB, *dynamic.DynamicStore) {
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

	// roles 테이블 생성
	_, err = dbConn.Exec(`
       CREATE TABLE IF NOT EXISTS roles (
           id TEXT PRIMARY KEY,
           name TEXT UNIQUE NOT NULL,
           description TEXT,
           rules TEXT NOT NULL,
           annotations TEXT,
           created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
           updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
           deleted_at TIMESTAMP
       )
   `)
	if err != nil {
		t.Fatalf("failed to create roles table: %v", err)
	}

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

func createTestRole(t *testing.T) *v1alpha1.Role {
	return &v1alpha1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-role",
			Annotations: map[string]string{
				"description": "Test Role Description",
			},
		},
		Rules: []v1alpha1.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				Resources: []string{"users"},
				APIGroups: []string{"auth.service"},
			},
		},
	}
}

func TestRoleStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Create basic role", func(t *testing.T) {
		role := createTestRole(t)
		err := store.Create(ctx, role)
		assert.NoError(t, err)

		saved, err := store.Get(ctx, role.Name)
		assert.NoError(t, err)
		assert.Equal(t, role.Name, saved.Name)
		assert.Equal(t, role.Rules[0].Verbs, saved.Rules[0].Verbs)
	})

	t.Run("Create duplicate role", func(t *testing.T) {
		role := createTestRole(t)
		role.Name = "duplicate-role"
		err := store.Create(ctx, role)
		assert.NoError(t, err)

		err = store.Create(ctx, role)
		assert.Error(t, err)
	})
}

func TestRoleStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Get existing role", func(t *testing.T) {
		role := createTestRole(t)
		err := store.Create(ctx, role)
		assert.NoError(t, err)

		found, err := store.Get(ctx, role.Name)
		assert.NoError(t, err)
		assert.Equal(t, role.Name, found.Name)
		assert.Equal(t, role.Rules[0].Verbs, found.Rules[0].Verbs)
	})

	t.Run("Get non-existent role", func(t *testing.T) {
		_, err := store.Get(ctx, "non-existent")
		assert.Error(t, err)
	})
}

func TestRoleStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update rules", func(t *testing.T) {
		role := createTestRole(t)
		err := store.Create(ctx, role)
		assert.NoError(t, err)

		role.Rules[0].Verbs = append(role.Rules[0].Verbs, "create")
		err = store.Update(ctx, role)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, role.Name)
		assert.NoError(t, err)
		assert.Contains(t, updated.Rules[0].Verbs, "create")
	})
}

func TestRoleStore_FindByVerb(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	role1 := createTestRole(t)
	role1.Name = "role1"
	role1.Rules[0].Verbs = []string{"get", "list"}
	err := store.Create(ctx, role1)
	assert.NoError(t, err)

	role2 := createTestRole(t)
	role2.Name = "role2"
	role2.Rules[0].Verbs = []string{"create", "update"}
	err = store.Create(ctx, role2)
	assert.NoError(t, err)

	t.Run("Find roles by verb", func(t *testing.T) {
		roles, err := store.FindByVerb(ctx, "get")
		assert.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "role1", roles[0].Name)
	})

	t.Run("Find roles by non-existent verb", func(t *testing.T) {
		roles, err := store.FindByVerb(ctx, "delete")
		assert.NoError(t, err)
		assert.Len(t, roles, 0)
	})
}

func TestRoleStore_FindByResource(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	role1 := createTestRole(t)
	role1.Name = "role1"
	role1.Rules[0].Resources = []string{"users"}
	err := store.Create(ctx, role1)
	assert.NoError(t, err)

	role2 := createTestRole(t)
	role2.Name = "role2"
	role2.Rules[0].Resources = []string{"roles"}
	err = store.Create(ctx, role2)
	assert.NoError(t, err)

	t.Run("Find roles by resource", func(t *testing.T) {
		roles, err := store.FindByResource(ctx, "users")
		assert.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "role1", roles[0].Name)
	})
}

func TestRoleStore_FindByAPIGroup(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	role1 := createTestRole(t)
	role1.Name = "role1"
	role1.Rules[0].APIGroups = []string{"auth.service"}
	err := store.Create(ctx, role1)
	assert.NoError(t, err)

	role2 := createTestRole(t)
	role2.Name = "role2"
	role2.Rules[0].APIGroups = []string{"core.service"}
	err = store.Create(ctx, role2)
	assert.NoError(t, err)

	t.Run("Find roles by API group", func(t *testing.T) {
		roles, err := store.FindByAPIGroup(ctx, "auth.service")
		assert.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "role1", roles[0].Name)
	})
}

func TestRoleStore_UpdateRules(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Update rules only", func(t *testing.T) {
		role := createTestRole(t)
		err := store.Create(ctx, role)
		assert.NoError(t, err)

		newRules := []v1alpha1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "create"},
				Resources: []string{"users", "roles"},
				APIGroups: []string{"auth.service"},
			},
		}

		err = store.UpdateRules(ctx, role.Name, newRules)
		assert.NoError(t, err)

		updated, err := store.Get(ctx, role.Name)
		assert.NoError(t, err)
		assert.Len(t, updated.Rules[0].Verbs, 3)
		assert.Len(t, updated.Rules[0].Resources, 2)
	})
}
