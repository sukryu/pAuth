package dynamic

import (
	"context"
	"database/sql"
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
func createIndex(ctx context.Context, db db.DBTX, tableName string, index schema.IndexDef) error {
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
