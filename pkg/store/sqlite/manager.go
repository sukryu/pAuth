package sqlite

import (
	"context"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sukryu/pAuth/pkg/store/database"
)

type Manager struct {
	db           *gorm.DB
	tableManager *TableManager
}

func NewManager(cfg database.Config) (database.Manager, error) {
	db, err := gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	return &Manager{
		db:           db,
		tableManager: NewTableManager(db),
	}, nil
}

func (m *Manager) Initialize(ctx context.Context) error {
	return m.tableManager.Initialize(ctx)
}

func (m *Manager) GetDB() *gorm.DB {
	return m.db
}

func (m *Manager) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (m *Manager) GetStats() map[string]interface{} {
	sqlDB, err := m.db.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
	}
}

type managerFactory struct{}

func NewManagerFactory() database.ManagerFactory {
	return &managerFactory{}
}

func (f *managerFactory) NewManager(cfg database.Config) (database.Manager, error) {
	return NewManager(cfg)
}
