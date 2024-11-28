package interfaces

import (
	"context"

	"github.com/sukryu/pAuth/internal/store/schema"
)

type DynamicStore interface {
	CreateDynamicTable(ctx context.Context, tableName string, opts schema.TableOptions) error
	AlterDynamicTable(ctx context.Context, tableName string, alterSQL string) error
	CreateDynamicIndex(ctx context.Context, indexName, tableName string, columns string) error
	DynamicInsert(ctx context.Context, tableName string, data map[string]interface{}) error
	DynamicSelect(ctx context.Context, tableName string, conditions map[string]interface{}) ([]map[string]interface{}, error)
	DynamicUpdate(ctx context.Context, tableName string, id string, data map[string]interface{}) error
	DynamicDelete(ctx context.Context, tableName string, id string) error
	GetTableSchema(ctx context.Context, tableName string) ([]string, error)
}
