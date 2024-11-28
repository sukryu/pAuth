package manager

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Manager defines the interface for database operations
type Manager interface {
	// Initialize performs any necessary database setup
	Initialize(ctx context.Context) error

	// GetDB returns the database connection
	GetDB() *sql.DB

	// Close closes the database connection
	Close() error

	// GetStats returns database statistics
	GetStats() map[string]interface{}

	// EnsureSchemaExists ensures the existence of a specific schema
	EnsureSchemaExists(ctx context.Context, schemaName string, schemaSQL string) error
}

// Config holds database configuration
type Config struct {
	Type     string
	DSN      string
	MaxConns int
}

// SQLManager implements the Manager interface using sql.DB
type SQLManager struct {
	db       *sql.DB
	dsn      string
	dbType   string
	maxConns int
}

// NewSQLManager creates a new SQLManager
func NewSQLManager(cfg Config) (*SQLManager, error) {
	db, err := sql.Open(cfg.Type, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if cfg.MaxConns > 0 {
		db.SetMaxOpenConns(cfg.MaxConns)
		db.SetMaxIdleConns(cfg.MaxConns / 2)
	}

	return &SQLManager{
		db:       db,
		dsn:      cfg.DSN,
		dbType:   cfg.Type,
		maxConns: cfg.MaxConns,
	}, nil
}

// Initialize performs any necessary database setup
func (m *SQLManager) Initialize(ctx context.Context) error {
	// Example: Run migrations or other setup tasks
	return nil
}

// GetDB returns the database connection
func (m *SQLManager) GetDB() *sql.DB {
	return m.db
}

// Close closes the database connection
func (m *SQLManager) Close() error {
	return m.db.Close()
}

// GetStats returns database statistics
func (m *SQLManager) GetStats() map[string]interface{} {
	stats := m.db.Stats()
	healthy := true
	if err := m.db.Ping(); err != nil {
		healthy = false
	}
	return map[string]interface{}{
		"max_open_conns": stats.MaxOpenConnections,
		"open_conns":     stats.OpenConnections,
		"in_use":         stats.InUse,
		"idle":           stats.Idle,
		"healthy":        healthy,
	}
}

// EnsureSchemaExists ensures the existence of a specific schema in the database
func (m *SQLManager) EnsureSchemaExists(ctx context.Context, schemaName string, schemaSQL string) error {
	_, err := m.db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to ensure schema exists: %w", err)
	}
	return nil
}

// ManagerFactory creates database managers
type ManagerFactory interface {
	NewManager(cfg Config) (Manager, error)
}

// SQLManagerFactory creates SQLManager instances
type SQLManagerFactory struct{}

// NewManager creates a new SQLManager
func (f *SQLManagerFactory) NewManager(cfg Config) (Manager, error) {
	switch cfg.Type {
	case "sqlite":
		return NewSQLManager(cfg)
	case "postgresql":
		// Add logic for PostgreSQL-specific manager initialization if needed
		return NewSQLManager(cfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
