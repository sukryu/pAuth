package dynamic

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/db"
	"github.com/sukryu/pAuth/internal/store/dynamic/query"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/schema"
)

func setupTestDB(t *testing.T) (*sql.DB, *DynamicStore) {
	manager, err := manager.NewSQLManager(manager.Config{
		Type: "sqlite3",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("failed to create SQLManager: %v", err)
	}

	dbConn := manager.GetDB()

	// 필요한 테이블 생성
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
       );
       CREATE TABLE IF NOT EXISTS schema_versions (
           id INTEGER PRIMARY KEY AUTOINCREMENT,
           schema_name TEXT NOT NULL,
           version INTEGER NOT NULL,
           changes TEXT,
           created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
       );
       CREATE TABLE IF NOT EXISTS schema_dependencies (
           id INTEGER PRIMARY KEY AUTOINCREMENT,
           parent_schema TEXT NOT NULL,
           child_schema TEXT NOT NULL,
           dependency_type TEXT NOT NULL,
           created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
       );
    `)
	if err != nil {
		t.Fatalf("failed to create required tables: %v", err)
	}

	store, err := NewDynamicStore(manager)
	if err != nil {
		t.Fatalf("failed to create dynamic store: %v", err)
	}

	return dbConn, store
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid table name", "valid_table_name", true},
		{"starts with number", "1_invalid", false},
		{"contains special char", "invalid-table", false},
		{"too long", "a_very_long_table_name_that_exceeds_the_sixty_four_character_limit", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidIdentifier(tt.input))
		})
	}
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

func TestDynamicStore_Caching(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()
	ctx := context.Background()

	// 캐시 만료 시간을 테스트에 맞게 짧게 설정
	store.versionCache = cache.New(50*time.Millisecond, 100*time.Millisecond)

	// 스키마 버전 추가
	err := store.TrackSchemaVersion(ctx, "test_schema", "Initial version")
	assert.NoError(t, err)

	// 첫 번째 호출 (DB에서 조회)
	versions, err := store.GetSchemaVersions(ctx, "test_schema")
	assert.NoError(t, err)
	assert.Len(t, versions, 1)

	// 캐시 적중 테스트
	versionsCached, err := store.GetSchemaVersions(ctx, "test_schema")
	assert.NoError(t, err)
	assert.Len(t, versionsCached, 1)
	assert.Equal(t, versions, versionsCached)

	// 캐시 강제 만료
	time.Sleep(60 * time.Millisecond) // 테스트용 짧은 만료 시간 사용
	_, found := store.versionCache.Get("test_schema")
	assert.False(t, found, "Cache should be expired")

	// 캐시 만료 후 재조회
	versionsAfterExpiry, err := store.GetSchemaVersions(ctx, "test_schema")
	assert.NoError(t, err)
	assert.Len(t, versionsAfterExpiry, 1)
}

func TestDynamicStore_DropColumnWithBatchCopy(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
		},
	}
	err := store.CreateDynamicTable(ctx, "test_users", opts)
	assert.NoError(t, err)

	// 데이터 삽입
	testData := []map[string]interface{}{
		{"id": "1", "name": "Alice", "age": 25},
		{"id": "2", "name": "Bob", "age": 30},
	}
	for _, data := range testData {
		err := store.DynamicInsert(ctx, "test_users", data)
		assert.NoError(t, err)
	}

	// 컬럼 삭제
	err = store.DropColumn(ctx, "test_users", "age", 1) // 배치 크기: 1
	assert.NoError(t, err)

	// 컬럼 삭제 후 데이터 확인
	results, err := store.DynamicSelect(ctx, "test_users", nil)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotContains(t, results[0], "age") // age 컬럼이 삭제되었는지 확인
}

func TestDynamicStore_SchemaDependencies(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()
	ctx := context.Background()

	// 의존성 추가
	err := store.AddSchemaDependency(ctx, "parent_schema", "child_schema", "ONE_TO_MANY")
	assert.NoError(t, err)

	// 의존성 조회
	dependencies, err := store.GetSchemaDependencies(ctx, "parent_schema")
	assert.NoError(t, err)
	assert.NotEmpty(t, dependencies) // 배열이 비어있지 않은지 확인

	if len(dependencies) > 0 {
		assert.Equal(t, "child_schema", dependencies[0].ChildSchema)
	}
}

func TestDynamicStore_AlterDynamicTable(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 초기 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
		},
	}
	err := store.CreateDynamicTable(ctx, "test_users", opts)
	assert.NoError(t, err)

	// 1. 컬럼 추가
	changes := map[string]string{"age INTEGER": "ADD"}
	err = store.AlterDynamicTable(ctx, "test_users", changes)
	assert.NoError(t, err)

	// 컬럼 추가 확인
	schema, err := store.GetTableSchema(ctx, "test_users")
	assert.NoError(t, err)
	assert.Contains(t, schema, "age INTEGER")

	// 2. 컬럼 삭제
	changes = map[string]string{"name": "DROP"}
	err = store.AlterDynamicTable(ctx, "test_users", changes)
	assert.NoError(t, err)

	// 컬럼 삭제 확인
	schema, err = store.GetTableSchema(ctx, "test_users")
	assert.NoError(t, err)
	assert.NotContains(t, schema, "name TEXT")

	// 3. 컬럼 수정 (예외 처리 확인 - SQLite 미지원)
	changes = map[string]string{"age": "MODIFY"}
	err = store.AlterDynamicTable(ctx, "test_users", changes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SQLite does not support MODIFY COLUMN")
}

func TestDynamicStore_TableExists(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
		},
	}
	err := store.CreateDynamicTable(ctx, "test_users", opts)
	assert.NoError(t, err)

	// 존재하는 테이블 확인
	exists, err := store.TableExists(ctx, "test_users")
	assert.NoError(t, err)
	assert.True(t, exists)

	// 존재하지 않는 테이블 확인
	exists, err = store.TableExists(ctx, "non_existent_table")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDynamicStore_AddColumn(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
		},
	}
	err := store.CreateDynamicTable(ctx, "test_users", opts)
	assert.NoError(t, err)

	// 새 컬럼 추가
	err = store.AddColumn(ctx, "test_users", "age INTEGER")
	assert.NoError(t, err)

	// 새 컬럼 확인
	schema, err := store.GetTableSchema(ctx, "test_users")
	assert.NoError(t, err)
	assert.Contains(t, schema, "age INTEGER")
}

func TestDynamicStore_DropDynamicTable(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
		},
	}
	err := store.CreateDynamicTable(ctx, "test_users", opts)
	assert.NoError(t, err)

	// 테이블 삭제
	err = store.DropDynamicTable("test_users")
	assert.NoError(t, err)

	// 삭제된 테이블 확인
	exists, err := store.TableExists(ctx, "test_users")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDynamicStore_Concurrency(t *testing.T) {
	dbConn, store := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// 다수의 테이블 생성
	for i := 0; i < 10; i++ {
		tableName := fmt.Sprintf("test_table_%d", i)
		opts := schema.TableOptions{
			Fields: []schema.FieldDef{
				{Name: "name", Type: schema.FieldTypeString, Nullable: true},
				{Name: "value", Type: schema.FieldTypeInteger, Nullable: true},
			},
		}
		_ = store.CreateDynamicTable(ctx, tableName, opts)
	}

	// 동시 작업 실행
	t.Run("ConcurrentInsertAndUpdate", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(tableIdx int) {
				defer wg.Done()
				tableName := fmt.Sprintf("test_table_%d", tableIdx)
				for j := 0; j < 1000; j++ {
					_ = store.DynamicInsert(ctx, tableName, map[string]interface{}{
						"id":    fmt.Sprintf("record%d", j),
						"name":  fmt.Sprintf("Record %d", j),
						"value": j,
					})
				}
			}(i)
		}
		wg.Wait()
	})
}

// func TestDynamicStore_DropColumnWithConcurrency(t *testing.T) {
// 	dbConn, store := setupTestDB(t)
// 	defer dbConn.Close()

// 	ctx := context.Background()

// 	// 테스트 테이블 생성
// 	opts := schema.TableOptions{
// 		Fields: []schema.FieldDef{
// 			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
// 			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
// 			{Name: "email", Type: schema.FieldTypeString, Nullable: true},
// 		},
// 	}
// 	tableName := "test_users"
// 	err := store.CreateDynamicTable(ctx, tableName, opts)
// 	assert.NoError(t, err)

// 	// 데이터 삽입
// 	totalRows := 10000
// 	for i := 0; i < totalRows; i++ {
// 		err := store.DynamicInsert(ctx, tableName, map[string]interface{}{
// 			"id":    fmt.Sprintf("user%d", i),
// 			"name":  fmt.Sprintf("User %d", i),
// 			"age":   i % 100,
// 			"email": fmt.Sprintf("user%d@example.com", i),
// 		})
// 		assert.NoError(t, err)
// 	}

// 	// 멀티스레드 DropColumn 호출
// 	batchSize := 500
// 	workers := 4
// 	err = store.DropColumnWithConcurrency(ctx, tableName, "email", batchSize, workers)
// 	assert.NoError(t, err)

// 	// 결과 검증
// 	// 1. 컬럼이 삭제되었는지 확인
// 	schemaAfter, err := store.GetTableSchema(ctx, tableName)
// 	assert.NoError(t, err)
// 	for _, col := range schemaAfter {
// 		assert.NotContains(t, col, "email", "Column 'email' should be removed")
// 	}

// 	// 2. 데이터 무결성 확인
// 	results, err := store.DynamicSelect(ctx, tableName, nil)
// 	assert.NoError(t, err)
// 	assert.Equal(t, totalRows, len(results), "Row count should match after column removal")
// 	for _, row := range results {
// 		assert.Contains(t, row, "name", "Column 'name' should exist")
// 		assert.Contains(t, row, "age", "Column 'age' should exist")
// 		assert.NotContains(t, row, "email", "Column 'email' should not exist")
// 	}
// }
