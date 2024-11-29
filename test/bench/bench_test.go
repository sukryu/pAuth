package bench

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/schema"
)

func setupTestDB(t *testing.B) (*sql.DB, *dynamic.DynamicStore) {
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

	store, err := dynamic.NewDynamicStore(manager)
	if err != nil {
		t.Fatalf("failed to create dynamic store: %v", err)
	}

	return dbConn, store
}
func BenchmarkDynamicStore_DropColumn(b *testing.B) {
	dbConn, store := setupTestDB(b)
	defer dbConn.Close()

	ctx := context.Background()

	// 초기 테이블 생성 및 데이터 준비
	opts := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
			{Name: "email", Type: schema.FieldTypeString, Nullable: true},
		},
	}
	tableName := "test_users"
	_ = store.CreateDynamicTable(ctx, tableName, opts)

	for i := 0; i < 1000000; i++ {
		_ = store.DynamicInsert(ctx, tableName, map[string]interface{}{
			"id":    fmt.Sprintf("user%d", i),
			"name":  fmt.Sprintf("User %d", i),
			"age":   i % 100,
			"email": fmt.Sprintf("user%d@example.com", i),
		})
	}

	// 벤치마크 실행
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 새로운 테이블 복사
		testTable := fmt.Sprintf("%s_bench_%d", tableName, i)
		_ = store.CreateDynamicTable(ctx, testTable, opts)

		// 데이터 복사
		_, _ = dbConn.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", testTable, tableName))

		// 컬럼 삭제 성능 테스트
		err := store.DropColumn(ctx, testTable, "email", 500) // 배치 크기 조정
		if err != nil {
			b.Fatalf("DropColumn failed: %v", err)
		}
	}
}

func BenchmarkDynamicStore_IndexVsNoIndex(b *testing.B) {
	dbConn, store := setupTestDB(b)
	defer dbConn.Close()

	ctx := context.Background()

	// 테이블 생성 (인덱스 적용)
	optsWithIndex := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
		},
		Indexes: []schema.IndexDef{
			{Name: "idx_age", Columns: []string{"age"}},
		},
	}
	_ = store.CreateDynamicTable(ctx, "users_with_index", optsWithIndex)

	// 테이블 생성 (인덱스 없음)
	optsNoIndex := schema.TableOptions{
		Fields: []schema.FieldDef{
			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
		},
	}
	_ = store.CreateDynamicTable(ctx, "users_no_index", optsNoIndex)

	// 데이터 삽입
	for i := 0; i < 100000; i++ {
		data := map[string]interface{}{
			"id":   fmt.Sprintf("user%d", i),
			"name": fmt.Sprintf("User %d", i),
			"age":  i % 100,
		}
		_ = store.DynamicInsert(ctx, "users_with_index", data)
		_ = store.DynamicInsert(ctx, "users_no_index", data)
	}

	b.ResetTimer()
	b.Run("QueryWithIndex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = store.DynamicSelect(ctx, "users_with_index", map[string]interface{}{
				"age": 50,
			})
		}
	})

	b.Run("QueryWithoutIndex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = store.DynamicSelect(ctx, "users_no_index", map[string]interface{}{
				"age": 50,
			})
		}
	})
}

// func BenchmarkDynamicStore_DropColumnWithConcurrency(b *testing.B) {
// 	dbConn, store := setupTestDB(b)
// 	defer dbConn.Close()

// 	ctx := context.Background()

// 	// 데이터가 포함된 테이블 생성
// 	opts := schema.TableOptions{
// 		Fields: []schema.FieldDef{
// 			{Name: "name", Type: schema.FieldTypeString, Nullable: true},
// 			{Name: "age", Type: schema.FieldTypeInteger, Nullable: true},
// 			{Name: "email", Type: schema.FieldTypeString, Nullable: true},
// 		},
// 	}
// 	baseTableName := "test_users"

// 	// 베이스 테이블이 존재하면 삭제
// 	_, _ = dbConn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", baseTableName))

// 	err := store.CreateDynamicTable(ctx, baseTableName, opts)
// 	if err != nil {
// 		b.Fatalf("Failed to create base table: %v", err)
// 	}

// 	// 데이터 삽입 (한 번만 수행)
// 	totalRows := 100000
// 	for i := 0; i < totalRows; i++ {
// 		err := store.DynamicInsert(ctx, baseTableName, map[string]interface{}{
// 			"id":    fmt.Sprintf("user%d", i),
// 			"name":  fmt.Sprintf("User %d", i),
// 			"age":   i % 100,
// 			"email": fmt.Sprintf("user%d@example.com", i),
// 		})
// 		if err != nil {
// 			b.Fatalf("Failed to insert data: %v", err)
// 		}
// 	}

// 	b.ResetTimer()

// 	batchSizes := []int{100, 500, 1000}
// 	workerCounts := []int{2, 4, 8}

// 	for _, batchSize := range batchSizes {
// 		for _, workers := range workerCounts {
// 			b.Run(fmt.Sprintf("BatchSize_%d_Workers_%d", batchSize, workers), func(b *testing.B) {
// 				for i := 0; i < b.N; i++ {
// 					// 각 반복마다 새로운 테이블 생성
// 					testTableName := fmt.Sprintf("%s_bench_%d_%d", baseTableName, i, time.Now().UnixNano())

// 					// 테스트 테이블이 존재하면 삭제
// 					_, _ = dbConn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))

// 					// 테스트 테이블 생성 (데이터 포함)
// 					_, err := dbConn.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", testTableName, baseTableName))
// 					if err != nil {
// 						b.Fatalf("Failed to create test table: %v", err)
// 					}

// 					// DropColumnWithConcurrency 실행
// 					err = store.DropColumnWithConcurrency(ctx, testTableName, "email", batchSize, workers)
// 					if err != nil {
// 						b.Fatalf("DropColumnWithConcurrency failed: %v", err)
// 					}

// 					// 테스트 테이블 삭제
// 					_, _ = dbConn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))
// 				}
// 			})
// 		}
// 	}
// }
