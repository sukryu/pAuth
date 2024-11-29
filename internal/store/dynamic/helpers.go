package dynamic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sukryu/pAuth/internal/db"
	"github.com/sukryu/pAuth/internal/store/schema"
)

// scanRows converts sql.Rows to []map[string]interface{}
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var results []map[string]interface{}
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// createIndex creates a new index on the specified table
func CreateIndex(ctx context.Context, db db.DBTX, tableName string, index schema.IndexDef) error {
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE"
	}

	query := fmt.Sprintf("CREATE %s INDEX IF NOT EXISTS %s ON %s (%s)",
		uniqueStr,
		index.Name,
		tableName,
		strings.Join(index.Columns, ", "))

	_, err := db.ExecContext(ctx, query)
	return err
}

func (s *DynamicStore) ValidateSchemaDefinition(fields map[string]string) error {
	for name, fieldType := range fields {
		if name == "" || fieldType == "" {
			return fmt.Errorf("invalid field definition: field name or type is empty")
		}

		// 허용된 데이터 타입 확인
		validTypes := []string{"TEXT", "INTEGER", "REAL", "BLOB", "BOOLEAN", "DATETIME"}
		if !contains(validTypes, fieldType) {
			return fmt.Errorf("invalid field type: %s is not supported", fieldType)
		}
	}
	return nil
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func GenerateCreateTableSQL(schema1 db.EntitySchema) (string, error) {
	// JSON에서 필드 정의 파싱
	var fields []schema.FieldDef
	if err := json.Unmarshal([]byte(schema1.Fields), &fields); err != nil {
		return "", fmt.Errorf("invalid fields format: %w", err)
	}

	// 필드 정의 기반으로 SQL 생성
	var columns []string
	for _, field := range fields {
		columnDef := fmt.Sprintf("%s %s", field.Name, field.Type)

		if field.PrimaryKey {
			columnDef += " PRIMARY KEY"
		}
		if field.NotNull {
			columnDef += " NOT NULL"
		}
		if field.AutoIncrement {
			columnDef += " AUTOINCREMENT"
		}
		if field.DefaultValue != nil {
			columnDef += fmt.Sprintf(" DEFAULT %v", field.DefaultValue)
		}

		columns = append(columns, columnDef)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", schema1.Name, strings.Join(columns, ", ")), nil
}

func GenerateAlterTableSQL(tableName string, changes map[string]string) (string, error) {
	// 변경 사항(action)에 대한 유효성 검증
	var alters []string
	for column, action := range changes {
		parts := strings.Split(column, " ")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid column definition for %s: %s", action, column)
		}
		columnName := parts[0]
		columnType := parts[1]

		switch action {
		case "ADD":
			alters = append(alters, fmt.Sprintf("ADD COLUMN %s %s", columnName, columnType))
		case "DROP":
			// SQLite는 DROP COLUMN 미지원 -> 예외 처리
			return "", fmt.Errorf("SQLite does not support DROP COLUMN directly for %s", columnName)
		case "MODIFY":
			// SQLite는 MODIFY COLUMN 미지원 -> 예외 처리
			return "", fmt.Errorf("SQLite does not support MODIFY COLUMN for %s", columnName)
		default:
			return "", fmt.Errorf("unsupported action: %s", action)
		}
	}

	return fmt.Sprintf("ALTER TABLE %s %s", tableName, strings.Join(alters, ", ")), nil
}
