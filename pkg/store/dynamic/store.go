package dynamic

import (
	"context"

	"github.com/sukryu/pAuth/pkg/store/schema"
)

// DynamicStore defines interface for dynamic table operations
type DynamicStore interface {
	// Basic CRUD operations
	Create(ctx context.Context, tableName string, data map[string]interface{}) error
	Get(ctx context.Context, tableName string, id string) (map[string]interface{}, error)
	Update(ctx context.Context, tableName string, id string, data map[string]interface{}) error
	Delete(ctx context.Context, tableName string, id string) error

	// Query operations
	List(ctx context.Context, tableName string, filter map[string]interface{}, limit, offset int) ([]map[string]interface{}, error)
	Count(ctx context.Context, tableName string, filter map[string]interface{}) (int64, error)
	Query(ctx context.Context, tableName string, query string, args ...interface{}) ([]map[string]interface{}, error)

	// Schema operations
	GetSchema(ctx context.Context, tableName string) (*schema.EntitySchema, error)
	ValidateData(data map[string]interface{}, schema *schema.EntitySchema) error

	// Transaction support
	TransactionWithOptions(ctx context.Context, opts TransactionOptions, fn func(tx DynamicStore) error) error

	Close() error
}

type TransactionOptions struct {
	IsolationLevel IsolationLevel
	ReadOnly       bool
}

type IsolationLevel int

const (
	ReadUncommitted IsolationLevel = iota
	ReadCommitted
	RepeatableRead
	Serializable
)

type DatabaseType string

const (
	SQLiteDB   DatabaseType = "sqlite"
	PostgresDB DatabaseType = "postgres"
	MySQLDB    DatabaseType = "mysql"
)
