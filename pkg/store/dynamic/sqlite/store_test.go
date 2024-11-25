package sqlite

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sukryu/pAuth/pkg/store/dynamic"
	"github.com/sukryu/pAuth/pkg/store/schema"
)

func setupTestDB(t *testing.T) (*gorm.DB, dynamic.DynamicStore) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	store, err := NewSQLiteDynamicStore(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return db, store
}

func createTestTable(t *testing.T, db *gorm.DB) *schema.EntitySchema {
	testSchema := &schema.EntitySchema{
		Name: "test_table",
		Fields: append(getCoreEntityFields(), []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Required: true},
			{Name: "age", Type: schema.FieldTypeNumber},
			{Name: "active", Type: schema.FieldTypeBoolean},
		}...),
	}

	// EntitySchema 테이블 생성
	err := db.AutoMigrate(&schema.EntitySchemaModel{})
	if err != nil {
		t.Fatalf("failed to migrate schema table: %v", err)
	}

	// 스키마를 모델로 변환
	model, err := toModel(testSchema)
	if err != nil {
		t.Fatalf("failed to convert schema to model: %v", err)
	}

	// 모델 저장
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	// 실제 테이블 생성
	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
        id TEXT PRIMARY KEY,
		created_at DATE
		updated_at DATE
        name TEXT NOT NULL,
        age NUMERIC,
        active BOOLEAN
		deleted_at DATE
    )`, testSchema.Name)

	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	return testSchema
}
func TestCRUD(t *testing.T) {
	db, store := setupTestDB(t)
	testSchema := createTestTable(t, db)
	ctx := context.Background()

	var createdID string

	t.Run("Create", func(t *testing.T) {
		data := map[string]interface{}{
			"name":   "test",
			"age":    25,
			"active": true,
		}

		err := store.Create(ctx, testSchema.Name, data)
		assert.NoError(t, err)

		id, ok := data["id"].(string)
		assert.True(t, ok, "id should be a string")
		createdID = id
	})

	t.Run("Get", func(t *testing.T) {
		result, err := store.Get(ctx, testSchema.Name, createdID)
		assert.NoError(t, err)
		assert.Equal(t, "test", result["name"])
		assert.Equal(t, 25.0, result["age"]) // SQLite converts numbers to float64
	})

	t.Run("Update", func(t *testing.T) {
		update := map[string]interface{}{
			"age": 26,
		}
		err := store.Update(ctx, testSchema.Name, createdID, update)
		assert.NoError(t, err)

		result, err := store.Get(ctx, testSchema.Name, createdID)
		assert.NoError(t, err)
		assert.Equal(t, 26.0, result["age"])
	})

	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, testSchema.Name, createdID)
		assert.NoError(t, err)

		_, err = store.Get(ctx, testSchema.Name, createdID)
		assert.Error(t, err)
	})
}

func TestTransactionWithOptions(t *testing.T) {
	db, store := setupTestDB(t)
	testSchema := createTestTable(t, db)
	ctx := context.Background()

	var createdID string

	t.Run("Successful Transaction", func(t *testing.T) {
		err := store.TransactionWithOptions(ctx, dynamic.TransactionOptions{
			IsolationLevel: dynamic.Serializable,
			ReadOnly:       false,
		}, func(tx dynamic.DynamicStore) error {
			data := map[string]interface{}{
				"name":   "transaction_test",
				"age":    30,
				"active": true,
			}
			if err := tx.Create(ctx, testSchema.Name, data); err != nil {
				return err
			}

			id, ok := data["id"].(string)
			assert.True(t, ok, "id should be a string")
			createdID = id

			return nil
		})

		assert.NoError(t, err)

		result, err := store.Get(ctx, testSchema.Name, createdID)
		assert.NoError(t, err)
		assert.Equal(t, "transaction_test", result["name"])
	})
}

func TestValidation(t *testing.T) {
	db, store := setupTestDB(t)
	testSchema := createTestTable(t, db)
	ctx := context.Background()

	t.Run("Invalid Type", func(t *testing.T) {
		data := map[string]interface{}{
			"id":   "4",
			"name": 123, // Should be string
		}
		err := store.Create(ctx, testSchema.Name, data)
		assert.Error(t, err)
	})

	t.Run("Missing Required Field", func(t *testing.T) {
		data := map[string]interface{}{
			"id":     "4",
			"active": true,
		}
		err := store.Create(ctx, testSchema.Name, data)
		assert.Error(t, err)
	})
}
