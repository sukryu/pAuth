package dynamic

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sukryu/pAuth/internal/db"
	"github.com/sukryu/pAuth/internal/store/dynamic/query"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/schema"
)

type DynamicStore struct {
	manager      manager.Manager
	queries      *db.Queries
	versionCache *cache.Cache
}

// NewDynamicStore initializes a new DynamicStore instance
func NewDynamicStore(mgr manager.Manager) (*DynamicStore, error) {
	// Get DB connection from manager
	dbConn := mgr.GetDB()
	if dbConn == nil {
		return nil, fmt.Errorf("failed to initialize DynamicStore: no valid database connection")
	}

	// Create queries instance using the db.New function
	queries := db.New(dbConn)

	return &DynamicStore{
		manager:      mgr,
		queries:      queries,
		versionCache: cache.New(5*time.Minute, 10*time.Minute),
	}, nil
}

// 동적 테이블 생성
func (s *DynamicStore) CreateDynamicTable(ctx context.Context, tableName string, opts schema.TableOptions) error {
	// Validate table name
	if !isValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// 테이블 기본 컬럼과 추가 필드 설정
	baseColumns := `
        id TEXT PRIMARY KEY,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP`
	columnDefs := []string{baseColumns}

	for _, field := range opts.Fields {
		columnDefs = append(columnDefs, field.GenerateColumnDef())
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		tableName, strings.Join(columnDefs, ", "))
	_, err := s.manager.GetDB().ExecContext(ctx, query)
	if err != nil {
		return err
	}

	// 인덱스 생성
	for _, idx := range opts.Indexes {
		if err := s.CreateDynamicIndex(ctx, idx.Name, tableName, strings.Join(idx.Columns, ", ")); err != nil {
			return err
		}
	}
	return nil
}

// AlterDynamicTable 테이블 수정
func (s *DynamicStore) AlterDynamicTable(ctx context.Context, tableName string, changes map[string]string) error {
	// 테이블 이름 검증
	if !isValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// 변경 사항(action) 검증 및 처리
	for column, action := range changes {
		if err := validateChangeAction(action); err != nil {
			return err
		}

		switch action {
		case "ADD":
			// 컬럼 추가 (데이터 타입 추출)
			parts := strings.Split(column, " ")
			if len(parts) != 2 {
				return fmt.Errorf("invalid column definition for ADD: %s", column)
			}
			columnName := parts[0]
			columnType := parts[1]
			columnDef := fmt.Sprintf("%s %s", columnName, columnType)

			if err := s.AddColumn(ctx, tableName, columnDef); err != nil {
				return fmt.Errorf("failed to add column %s: %w", columnName, err)
			}
		case "DROP":
			// 컬럼 삭제
			if err := s.DropColumn(ctx, tableName, column, 1000); err != nil { // 배치 크기 1000
				return fmt.Errorf("failed to drop column %s: %w", column, err)
			}
		case "MODIFY":
			// SQLite에서는 MODIFY 지원 불가 -> 명확한 예외 처리
			return fmt.Errorf("SQLite does not support MODIFY COLUMN directly for %s", column)
		default:
			return fmt.Errorf("unsupported action: %s", action)
		}
	}

	return nil
}

// CreateDynamicIndex 인덱스 생성
func (s *DynamicStore) CreateDynamicIndex(ctx context.Context, indexName, tableName string, columns string) error {
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
		indexName, tableName, columns)
	_, err := s.manager.GetDB().ExecContext(ctx, query)
	return err
}

// 스키마 유효성 검증
func (s *DynamicStore) ValidateSchema(ctx context.Context, tableName string, fields []schema.FieldDef) error {
	existingSchema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return err
	}

	existingColumns := map[string]bool{}
	for _, col := range existingSchema {
		parts := strings.Split(col, " ")
		if len(parts) > 0 {
			existingColumns[parts[0]] = true
		}
	}

	for _, field := range fields {
		if !existingColumns[field.Name] {
			return fmt.Errorf("missing column: %s", field.Name)
		}
	}

	return nil
}

// DynamicInsert 동적 테이블에 데이터 삽입
func (s *DynamicStore) DynamicInsert(ctx context.Context, tableName string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	placeholders := make([]string, 0, len(data))

	for col, val := range data {
		columns = append(columns, col)
		values = append(values, val)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	_, err := s.manager.GetDB().ExecContext(ctx, query, values...)
	return err
}

// DynamicSelect 동적 테이블에서 데이터 조회
func (s *DynamicStore) DynamicSelect(ctx context.Context, tableName string, conditions map[string]interface{}) ([]map[string]interface{}, error) {
	clauses := []string{"deleted_at IS NULL"} // 기본 조건
	values := make([]interface{}, 0)

	// 추가 조건이 있는 경우
	if len(conditions) > 0 {
		for col, val := range conditions {
			clauses = append(clauses, fmt.Sprintf("%s = ?", col))
			values = append(values, val)
		}
	}

	// WHERE 절 구성
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s",
		tableName,
		strings.Join(clauses, " AND "))

	rows, err := s.manager.GetDB().QueryContext(ctx, query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 결과 스캔 로직은 동일
	results := make([]map[string]interface{}, 0)
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// DynamicUpdate 동적 테이블의 데이터 업데이트
func (s *DynamicStore) DynamicUpdate(ctx context.Context, tableName string, id string, data map[string]interface{}) error {
	setParts := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data)+1)

	for col, val := range data {
		setParts = append(setParts, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}
	values = append(values, id) // WHERE id = ? 조건을 위한 값

	query := fmt.Sprintf("UPDATE %s SET %s, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		tableName,
		strings.Join(setParts, ", "))

	result, err := s.manager.GetDB().ExecContext(ctx, query, values...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("no record found with id: %s", id)
	}

	return nil
}

// DynamicDelete 동적 테이블의 데이터 삭제 (소프트 삭제)
func (s *DynamicStore) DynamicDelete(ctx context.Context, tableName string, id string) error {
	query := fmt.Sprintf("UPDATE %s SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL",
		tableName)

	result, err := s.manager.GetDB().ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("no record found with id: %s", id)
	}

	return nil
}

// DynamicQuery 동적 테이블의 복잡한 쿼리 실행
func (s *DynamicStore) DynamicQuery(ctx context.Context, tableName string, queryParams query.QueryParams) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT %s FROM %s",
		queryParams.GetSelectClause(),
		tableName)

	if whereClause := queryParams.GetWhereClause(); whereClause != "" {
		query += " WHERE " + whereClause
	}

	if orderBy := queryParams.GetOrderByClause(); orderBy != "" {
		query += " ORDER BY " + orderBy
	}

	if limit := queryParams.GetLimitClause(); limit != "" {
		query += " " + limit
	}

	rows, err := s.manager.GetDB().QueryContext(ctx, query, queryParams.GetArgs()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// 테이블 존재 여부 확인
func (s *DynamicStore) TableExists(ctx context.Context, tableName string) (bool, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	row := s.manager.GetDB().QueryRowContext(ctx, query, tableName)

	var name string
	err := row.Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// 테이블 컬럼 추가
func (s *DynamicStore) AddColumn(ctx context.Context, tableName, columnDef string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef)
	_, err := s.manager.GetDB().ExecContext(ctx, query)
	return err
}

// 테이블 컬럼 삭제 (SQLite는 지원하지 않으므로 우회 방법 필요)
func (s *DynamicStore) DropColumn(ctx context.Context, tableName, columnName string, batchSize int) error {
	// 1. 기존 테이블의 스키마 조회
	columns, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
	}

	// 2. 삭제할 컬럼 존재 여부 확인
	columnExists := false
	newColumns := []string{}
	for _, col := range columns {
		if strings.HasPrefix(col, columnName+" ") {
			columnExists = true
			continue
		}
		newColumns = append(newColumns, col)
	}
	if !columnExists {
		return fmt.Errorf("column %s does not exist in table %s", columnName, tableName)
	}

	// 3. 새 테이블 이름 정의
	tempTable := tableName + "_temp"

	// 4. 새 테이블 생성
	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tempTable, strings.Join(newColumns, ", "))
	if _, err := s.manager.GetDB().ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("failed to create temp table: %w", err)
	}

	// 5. 데이터 배치 복사
	offset := 0
	for {
		// 데이터 복사 쿼리: 배치 단위로 처리
		copySQL := fmt.Sprintf(
			"INSERT INTO %s SELECT %s FROM %s LIMIT %d OFFSET %d",
			tempTable,
			strings.Join(getColumnNames(newColumns), ", "),
			tableName,
			batchSize,
			offset,
		)

		// 복사 실행
		result, err := s.manager.GetDB().ExecContext(ctx, copySQL)
		if err != nil {
			return fmt.Errorf("failed to copy data in batches: %w", err)
		}

		// 처리된 행 수 확인
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			break // 더 이상 복사할 데이터 없음
		}

		offset += batchSize
	}

	// 6. 기존 테이블 삭제 및 교체
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	if _, err := s.manager.GetDB().ExecContext(ctx, dropSQL); err != nil {
		return fmt.Errorf("failed to drop original table: %w", err)
	}

	renameSQL := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTable, tableName)
	if _, err := s.manager.GetDB().ExecContext(ctx, renameSQL); err != nil {
		return fmt.Errorf("failed to rename temp table: %w", err)
	}

	return nil
}

// 테이블의 현재 스키마 조회
func (s *DynamicStore) GetTableSchema(ctx context.Context, tableName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := s.manager.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []string{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString

		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			return nil, err
		}
		columns = append(columns, fmt.Sprintf("%s %s", name, ctype))
	}
	return columns, nil
}

// 테이블 삭제
func (s *DynamicStore) DropDynamicTable(tableName string) error {
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)

	_, err := s.manager.GetDB().ExecContext(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	return nil
}

func (s *DynamicStore) TrackSchemaVersion(ctx context.Context, schemaName string, changes string) error {
	query := `INSERT INTO schema_versions (schema_name, version, changes, created_at)
              VALUES (?, (SELECT COALESCE(MAX(version), 0) + 1 FROM schema_versions WHERE schema_name = ?), ?, CURRENT_TIMESTAMP)`
	_, err := s.manager.GetDB().ExecContext(ctx, query, schemaName, schemaName, changes)
	return err
}

func (s *DynamicStore) GetSchemaVersions(ctx context.Context, schemaName string) ([]db.SchemaVersion, error) {
	// 캐시 확인.
	if cached, found := s.versionCache.Get(schemaName); found {
		return cached.([]db.SchemaVersion), nil
	}

	// DB에서 조회.
	query := `SELECT id, schema_name, version, changes, created_at FROM schema_versions WHERE schema_name = ? ORDER BY version DESC`
	rows, err := s.manager.GetDB().QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []db.SchemaVersion
	for rows.Next() {
		var version db.SchemaVersion
		if err := rows.Scan(&version.ID, &version.SchemaName, &version.Version, &version.Changes, &version.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	// 캐시에 저장
	s.versionCache.Set(schemaName, versions, cache.DefaultExpiration)
	return versions, nil
}

func (s *DynamicStore) AddSchemaDependency(ctx context.Context, parent, child, dependencyType string) error {
	query := `INSERT INTO schema_dependencies (parent_schema, child_schema, dependency_type, created_at)
              VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := s.manager.GetDB().ExecContext(ctx, query, parent, child, dependencyType)
	return err
}

func (s *DynamicStore) GetSchemaDependencies(ctx context.Context, schemaName string) ([]db.SchemaDependency, error) {
	query := `SELECT id, parent_schema, child_schema, dependency_type, created_at FROM schema_dependencies
              WHERE parent_schema = ? OR child_schema = ?`
	rows, err := s.manager.GetDB().QueryContext(ctx, query, schemaName, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dependencies []db.SchemaDependency
	for rows.Next() {
		var dependency db.SchemaDependency
		if err := rows.Scan(&dependency.ID, &dependency.ParentSchema, &dependency.ChildSchema, &dependency.DependencyType, &dependency.CreatedAt); err != nil {
			return nil, err
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

// isValidIdentifier checks if the given identifier (e.g., table name) is valid
func isValidIdentifier(identifier string) bool {
	// Define regex for valid identifiers
	// Must start with a letter, can contain letters, numbers, and underscores, and be 1-64 characters long
	regex := `^[a-zA-Z][a-zA-Z0-9_]{0,63}$`
	matched, err := regexp.MatchString(regex, identifier)
	if err != nil {
		return false // Return false if regex compilation fails
	}
	return matched
}

var allowedActions = map[string]bool{
	"ADD":    true,
	"DROP":   true,
	"MODIFY": true,
}

// validateChangeAction checks if the provided action is valid
func validateChangeAction(action string) error {
	if !allowedActions[action] {
		return fmt.Errorf("unsupported action: %s", action)
	}
	return nil
}

// batchCopy performs batch data copy using the provided SQL
func (s *DynamicStore) batchCopy(ctx context.Context, copySQL string, sourceTable string, batchSize int) (int, error) {
	offset := 0
	totalRows := 0

	for {
		// Add LIMIT and OFFSET for batch processing
		query := fmt.Sprintf("%s LIMIT %d OFFSET %d", copySQL, batchSize, offset)

		// Execute batch copy
		_, err := s.manager.GetDB().ExecContext(ctx, query)
		if err != nil {
			return totalRows, err
		}

		// Check how many rows were processed
		processedRows := s.getRowCount(ctx, sourceTable, batchSize, offset)
		if processedRows == 0 {
			break
		}

		totalRows += processedRows
		offset += batchSize
	}

	return totalRows, nil
}

// getRowCount gets the count of rows processed in the batch
func (s *DynamicStore) getRowCount(ctx context.Context, tableName string, batchSize, offset int) int {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s LIMIT %d OFFSET %d", tableName, batchSize, offset)
	row := s.manager.GetDB().QueryRowContext(ctx, query)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

// getColumnNames extracts column names without data types
func getColumnNames(columns []string) []string {
	names := []string{}
	for _, col := range columns {
		parts := strings.Split(col, " ")
		if len(parts) > 0 {
			names = append(names, parts[0])
		}
	}
	return names
}

// TEST
// func (s *DynamicStore) DropColumnWithConcurrency(ctx context.Context, originalTableName, columnName string, batchSize, workers int) error {
// 	columns, err := s.GetTableSchema(ctx, originalTableName)
// 	if err != nil {
// 		return fmt.Errorf("failed to get schema for table %s: %w", originalTableName, err)
// 	}

// 	// 삭제할 컬럼을 제외한 새로운 컬럼 리스트 생성
// 	newColumns := []string{}
// 	for _, col := range columns {
// 		if !strings.HasPrefix(col, columnName+" ") {
// 			newColumns = append(newColumns, col)
// 		}
// 	}

// 	if len(newColumns) == len(columns) {
// 		return fmt.Errorf("column %s does not exist in table %s", columnName, originalTableName)
// 	}

// 	// 고유한 임시 테이블 이름 생성
// 	tempTableName := fmt.Sprintf("%s_temp_%d", originalTableName, time.Now().UnixNano())

// 	// 임시 테이블이 존재하면 삭제
// 	_, _ = s.manager.GetDB().ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tempTableName))

// 	// 임시 테이블 생성 (DDL은 트랜잭션 밖에서 실행)
// 	createTableSQL := fmt.Sprintf("CREATE TABLE %s AS SELECT %s FROM %s WHERE 0", tempTableName, strings.Join(getColumnNames(newColumns), ", "), originalTableName)
// 	if _, err := s.manager.GetDB().ExecContext(ctx, createTableSQL); err != nil {
// 		return fmt.Errorf("failed to create temp table: %w", err)
// 	}

// 	// 전체 행 수 계산
// 	totalRows, err := s.testgetRowCount(ctx, originalTableName)
// 	if err != nil {
// 		return fmt.Errorf("failed to get row count for table %s: %w", originalTableName, err)
// 	}
// 	if totalRows == 0 {
// 		return fmt.Errorf("no rows found in table %s", originalTableName)
// 	}

// 	// 데이터 복사 (멀티스레드로 수행)
// 	batchCh := make(chan int, workers)
// 	//errCh := make(chan error, workers)
// 	var wg sync.WaitGroup
// 	var copyErr error
// 	copyErrMutex := &sync.Mutex{}

// 	// 워커 실행
// 	for i := 0; i < workers; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for offset := range batchCh {
// 				copySQL := fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s LIMIT %d OFFSET %d",
// 					tempTableName, strings.Join(getColumnNames(newColumns), ", "), originalTableName, batchSize, offset)

// 				if _, err := s.manager.GetDB().ExecContext(ctx, copySQL); err != nil {
// 					copyErrMutex.Lock()
// 					if copyErr == nil {
// 						copyErr = fmt.Errorf("failed to copy data in batch: %w", err)
// 					}
// 					copyErrMutex.Unlock()
// 					return
// 				}
// 			}
// 		}()
// 	}

// 	// 배치 작업 분배
// 	for offset := 0; offset < totalRows; offset += batchSize {
// 		batchCh <- offset
// 	}
// 	close(batchCh)

// 	// 워커 완료 대기
// 	wg.Wait()

// 	// 에러 확인
// 	if copyErr != nil {
// 		return copyErr
// 	}

// 	// 원본 테이블 삭제 및 임시 테이블로 교체 (DDL은 트랜잭션 밖에서 실행)
// 	_, err = s.manager.GetDB().ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", originalTableName))
// 	if err != nil {
// 		return fmt.Errorf("failed to drop original table: %w", err)
// 	}

// 	_, err = s.manager.GetDB().ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTableName, originalTableName))
// 	if err != nil {
// 		return fmt.Errorf("failed to rename temp table: %w", err)
// 	}

// 	return nil
// }

// func (s *DynamicStore) copyDataWithConcurrency(ctx context.Context, sourceTable, destTable string, columns []string, batchSize, workers int) error {
// 	jobs := make(chan int, workers)
// 	errors := make(chan error, workers)
// 	var wg sync.WaitGroup

// 	// 워커 생성
// 	for i := 0; i < workers; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for offset := range jobs {
// 				// 기존 getRowCount 메서드를 활용하여 복사
// 				count := s.getRowCount(ctx, sourceTable, batchSize, offset)
// 				if count == 0 {
// 					continue
// 				}
// 				copySQL := fmt.Sprintf("INSERT INTO %s SELECT %s FROM %s LIMIT %d OFFSET %d",
// 					destTable, strings.Join(getColumnNames(columns), ", "), sourceTable, batchSize, offset)
// 				if _, err := s.manager.GetDB().ExecContext(ctx, copySQL); err != nil {
// 					errors <- fmt.Errorf("failed to copy batch at offset %d: %w", offset, err)
// 					return
// 				}
// 			}
// 		}()
// 	}

// 	// 작업 생성
// 	for offset := 0; offset < s.getRowCount(ctx, sourceTable, 0, 0); offset += batchSize {
// 		jobs <- offset
// 	}
// 	close(jobs)

// 	// 워커 종료 대기
// 	wg.Wait()
// 	close(errors)

// 	// 에러 확인
// 	if len(errors) > 0 {
// 		return <-errors
// 	}

// 	return nil
// }

// func (s *DynamicStore) testgetRowCount(ctx context.Context, tableName string) (int, error) {
// 	var count int
// 	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
// 	err := s.manager.GetDB().QueryRowContext(ctx, query).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return count, nil
// }
