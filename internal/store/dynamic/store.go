package dynamic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sukryu/pAuth/internal/db"
	"github.com/sukryu/pAuth/internal/store/dynamic/query"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/schema"
)

type DynamicStore struct {
	manager manager.Manager
	queries *db.Queries
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
		manager: mgr,
		queries: queries,
	}, nil
}

// 동적 테이블 생성
func (s *DynamicStore) CreateDynamicTable(ctx context.Context, tableName string, opts schema.TableOptions) error {
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

	if _, err := s.manager.GetDB().ExecContext(ctx, query); err != nil {
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
func (s *DynamicStore) AlterDynamicTable(ctx context.Context, tableName string, alterSQL string) error {
	query := fmt.Sprintf("ALTER TABLE %s %s", tableName, alterSQL)
	_, err := s.manager.GetDB().ExecContext(ctx, query)
	return err
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
func (s *DynamicStore) tableExists(ctx context.Context, tableName string) (bool, error) {
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
func (s *DynamicStore) DropColumn(ctx context.Context, tableName, columnName string) error {
	return fmt.Errorf("sqlite does not support dropping columns directly. Consider creating a new table.")
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
func (s *DynamicStore) DropTable(ctx context.Context, tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := s.manager.GetDB().ExecContext(ctx, query)
	return err
}
