package dynamic

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/db"
	"github.com/sukryu/pAuth/internal/store/dynamic/query"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/schema"
)

func setupTestDB(t *testing.T) (*sql.DB, *DynamicStore) {
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

	// DynamicStore 생성
	store, err := NewDynamicStore(manager)
	if err != nil {
		t.Fatalf("failed to create dynamic store: %v", err)
	}

	return dbConn, store
}

func TestDynamicStore_CreateAndGetSchema(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테스트 스키마 생성
	schema := db.CreateSchemaParams{
		ID:          "test-id",
		Name:        "test_table",
		Description: sql.NullString{String: "Test Table", Valid: true},
		Fields:      `[{"name": "title", "type": "string", "required": true}]`,
		Indexes:     sql.NullString{String: "[]", Valid: true},
	}

	err := store.queries.CreateSchema(ctx, schema)
	assert.NoError(t, err)

	// 스키마 조회
	result, err := store.queries.GetSchema(ctx, schema.Name)
	assert.NoError(t, err)
	assert.Equal(t, schema.Name, result.Name)
	assert.Equal(t, schema.Fields, result.Fields)
}

func TestDynamicStore_CreateAndQueryDynamicTable(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 동적 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{
				Name:     "title",
				Type:     schema.FieldTypeString,
				Nullable: false,
			},
			{
				Name:     "price",
				Type:     schema.FieldTypeInteger,
				Nullable: true,
			},
		},
	}

	err := store.CreateDynamicTable(ctx, "test_items", opts)
	assert.NoError(t, err)

	// 데이터 삽입
	data := map[string]interface{}{
		"id":    "item1",
		"title": "Test Item",
		"price": 1000,
	}
	err = store.DynamicInsert(ctx, "test_items", data)
	assert.NoError(t, err)

	// 데이터 조회 전에 삽입 확인
	var count int
	err = dbConn.QueryRow("SELECT COUNT(*) FROM test_items WHERE id = ?", "item1").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 데이터 조회
	results, err := store.DynamicSelect(ctx, "test_items", map[string]interface{}{
		"id": "item1",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	if len(results) > 0 {
		assert.Equal(t, "Test Item", results[0]["title"])
	}
}

func TestDynamicStore_UpdateAndDelete(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{
				Name:     "title",
				Type:     schema.FieldTypeString,
				Nullable: false,
			},
			{
				Name:     "price",
				Type:     schema.FieldTypeInteger,
				Nullable: true,
			},
		},
	}

	err := store.CreateDynamicTable(ctx, "test_items", opts)
	assert.NoError(t, err)

	// 초기 데이터 삽입
	data := map[string]interface{}{
		"id":    "item1",
		"title": "Original Title",
		"price": 1000,
	}
	err = store.DynamicInsert(ctx, "test_items", data)
	assert.NoError(t, err)

	// 데이터 존재 확인
	var count int
	err = dbConn.QueryRow("SELECT COUNT(*) FROM test_items WHERE id = ?", "item1").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 데이터 업데이트
	updateData := map[string]interface{}{
		"title": "Updated Title",
		"price": 2000,
	}
	err = store.DynamicUpdate(ctx, "test_items", "item1", updateData)
	assert.NoError(t, err)

	// 업데이트 확인
	results, err := store.DynamicSelect(ctx, "test_items", map[string]interface{}{
		"id": "item1",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", results[0]["title"])

	// 데이터 삭제
	err = store.DynamicDelete(ctx, "test_items", "item1")
	assert.NoError(t, err)

	// 삭제 확인
	results, err = store.DynamicSelect(ctx, "test_items", map[string]interface{}{
		"id": "item1",
	})
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestDynamicStore_ComplexQuery(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{
				Name:     "title",
				Type:     schema.FieldTypeString,
				Nullable: false,
			},
			{
				Name:     "price",
				Type:     schema.FieldTypeInteger,
				Nullable: true,
			},
			{
				Name:     "category",
				Type:     schema.FieldTypeString,
				Nullable: true,
			},
		},
		Indexes: []schema.IndexDef{
			{
				Name:    "idx_category",
				Columns: []string{"category"},
			},
		},
	}

	err := store.CreateDynamicTable(ctx, "test_products", opts)
	assert.NoError(t, err)

	// 테스트 데이터 삽입
	testProducts := []map[string]interface{}{
		{"id": "p1", "title": "Product 1", "price": 100, "category": "A"},
		{"id": "p2", "title": "Product 2", "price": 200, "category": "B"},
		{"id": "p3", "title": "Product 3", "price": 300, "category": "A"},
	}

	for _, product := range testProducts {
		err := store.DynamicInsert(ctx, "test_products", product)
		assert.NoError(t, err)
	}

	// 복잡한 쿼리 테스트
	queryParams := query.QueryParams{
		SelectColumns: []string{"id", "title", "price"},
		Where: []query.WhereCondition{
			{Column: "category", Operator: "=", Value: "A"},
			{Column: "price", Operator: ">", Value: 50},
		},
		OrderBy: []query.OrderByClause{
			{Column: "price", Desc: true},
		},
		Limit:  10,
		Offset: 0,
	}

	results, err := store.DynamicQuery(ctx, "test_products", queryParams)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Product 3", results[0]["title"])
}

func TestDynamicStore_ComplexQueryVariations(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()
	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{
				Name:     "title",
				Type:     schema.FieldTypeString,
				Nullable: false,
			},
			{
				Name:     "price",
				Type:     schema.FieldTypeInteger,
				Nullable: true,
			},
			{
				Name:     "category",
				Type:     schema.FieldTypeString,
				Nullable: true,
			},
		},
		Indexes: []schema.IndexDef{
			{
				Name:    "idx_category",
				Columns: []string{"category"},
			},
			{
				Name:    "idx_price",
				Columns: []string{"price"},
			},
		},
	}

	err := store.CreateDynamicTable(ctx, "test_items", opts)
	assert.NoError(t, err)

	// 테스트 데이터 삽입
	testData := []map[string]interface{}{
		{"id": "1", "title": "Item 1", "price": 100, "category": "A"},
		{"id": "2", "title": "Item 2", "price": 200, "category": "B"},
		{"id": "3", "title": "Item 3", "price": 300, "category": "A"},
	}
	for _, item := range testData {
		err := store.DynamicInsert(ctx, "test_items", item)
		assert.NoError(t, err)
	}

	// 테스트 케이스
	tests := []struct {
		name          string
		queryParams   query.QueryParams
		expectedLen   int
		expectedFirst string
	}{
		{
			name: "Select specific columns",
			queryParams: query.QueryParams{
				SelectColumns: []string{"id", "title"},
			},
			expectedLen: 3,
		},
		{
			name: "With multiple conditions",
			queryParams: query.QueryParams{
				Where: []query.WhereCondition{
					{Column: "category", Operator: "=", Value: "A"},
					{Column: "price", Operator: "<", Value: 200},
				},
			},
			expectedLen: 1,
		},
		{
			name: "With ordering and limit",
			queryParams: query.QueryParams{
				OrderBy: []query.OrderByClause{{Column: "price", Desc: true}},
				Limit:   2,
			},
			expectedLen:   2,
			expectedFirst: "Item 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.DynamicQuery(ctx, "test_items", tt.queryParams)
			assert.NoError(t, err)
			assert.Len(t, results, tt.expectedLen)

			if tt.expectedFirst != "" && len(results) > 0 {
				assert.Equal(t, tt.expectedFirst, results[0]["title"])
			}
		})
	}
}

func TestDynamicStore_ErrorCases(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()
	ctx := context.Background()

	t.Run("Invalid table name", func(t *testing.T) {
		_, err := store.DynamicSelect(ctx, "non_existent_table", nil)
		assert.Error(t, err)
	})

	t.Run("Invalid column in update", func(t *testing.T) {
		opts := schema.TableOptions{
			Fields: []schema.FieldDef{
				{
					Name:     "name",
					Type:     schema.FieldTypeString,
					Nullable: false,
				},
			},
		}
		err := store.CreateDynamicTable(ctx, "test_table", opts)
		assert.NoError(t, err)

		err = store.DynamicUpdate(ctx, "test_table", "1", map[string]interface{}{
			"non_existent_column": "value",
		})
		assert.Error(t, err)
	})

	t.Run("Invalid SQL in dynamic query", func(t *testing.T) {
		params := query.QueryParams{
			Where: []query.WhereCondition{
				{Column: "id", Operator: "INVALID", Value: "1"},
			},
		}
		_, err := store.DynamicQuery(ctx, "test_table", params)
		assert.Error(t, err)
	})
}
