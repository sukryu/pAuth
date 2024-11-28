package manager

import (
	"context"

	"gorm.io/gorm"
)

// Manager defines the interface for database operations
type Manager interface {
	// Initialize performs any necessary database setup
	Initialize(ctx context.Context) error

	// GetDB returns the database connection
	GetDB() *gorm.DB

	// Close closes the database connection
	Close() error

	// GetStats returns database statistics
	GetStats() map[string]interface{}
}

// Config holds database configuration
type Config struct {
	Type     string
	DSN      string
	MaxConns int
	// 기타 필요한 설정들...
}

// ManagerFactory creates database managers
type ManagerFactory interface {
	NewManager(cfg Config) (Manager, error)
}
