package factory

import (
	"fmt"
	"sync"

	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/pkg/store/database"
	"github.com/sukryu/pAuth/pkg/store/dynamic"
	"github.com/sukryu/pAuth/pkg/store/dynamic/sqlite"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
	"github.com/sukryu/pAuth/pkg/store/role"
	rolebinding "github.com/sukryu/pAuth/pkg/store/role_binding"
	"github.com/sukryu/pAuth/pkg/store/user"
)

type StoreFactory interface {
	NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error)
	NewRoleStore(cfg *config.DatabaseConfig) (interfaces.RoleStore, error)
	NewRoleBindingStore(cfg *config.DatabaseConfig) (interfaces.RoleBindingStore, error)
	NewDynamicStore(cfg *config.DatabaseConfig) (dynamic.DynamicStore, error)
	Close() error
	GetStats() map[string]interface{}
}

type storeFactory struct {
	mu             sync.RWMutex
	managers       map[string]database.Manager
	managerFactory database.ManagerFactory
}

func NewStoreFactory(managerFactory database.ManagerFactory) StoreFactory {
	return &storeFactory{
		managers:       make(map[string]database.Manager),
		managerFactory: managerFactory,
	}
}

func (f *storeFactory) getManager(cfg *config.DatabaseConfig) (database.Manager, error) {
	f.mu.RLock()
	manager, exists := f.managers[cfg.GetDSN()]
	f.mu.RUnlock()

	if exists {
		return manager, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if manager, exists = f.managers[cfg.GetDSN()]; exists {
		return manager, nil
	}

	// Create new manager
	manager, err := f.managerFactory.NewManager(database.Config{
		Type: cfg.Type,
		DSN:  cfg.GetDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %v", err)
	}

	// Initialize the manager
	if err := manager.Initialize(nil); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	f.managers[cfg.GetDSN()] = manager
	return manager, nil
}

func (f *storeFactory) NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error) {
	manager, err := f.getManager(cfg)
	if err != nil {
		return nil, err
	}

	return user.NewStore(user.Config{
		DatabaseType: cfg.Type,
		DB:           manager.GetDB(),
	})
}

func (f *storeFactory) NewRoleStore(cfg *config.DatabaseConfig) (interfaces.RoleStore, error) {
	manager, err := f.getManager(cfg)
	if err != nil {
		return nil, err
	}

	return role.NewStore(manager.GetDB(), cfg.Type)
}

func (f *storeFactory) NewRoleBindingStore(cfg *config.DatabaseConfig) (interfaces.RoleBindingStore, error) {
	manager, err := f.getManager(cfg)
	if err != nil {
		return nil, err
	}

	return rolebinding.NewStore(manager.GetDB(), cfg.Type)
}

func (f *storeFactory) NewDynamicStore(cfg *config.DatabaseConfig) (dynamic.DynamicStore, error) {
	manager, err := f.getManager(cfg)
	if err != nil {
		return nil, err
	}

	switch cfg.Type {
	case "sqlite":
		return sqlite.NewSQLiteDynamicStore(manager.GetDB())
	// case "postgres":
	// 	return postgres.NewPostgresDynamicStore(manager.GetDB())
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

func (f *storeFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	for dsn, manager := range f.managers {
		if err := manager.Close(); err != nil {
			errs = append(errs, err)
		}
		delete(f.managers, dsn)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing managers: %v", errs)
	}
	return nil
}

func (f *storeFactory) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := make(map[string]interface{})
	for dsn, manager := range f.managers {
		stats[dsn] = manager.GetStats()
	}
	return stats
}
