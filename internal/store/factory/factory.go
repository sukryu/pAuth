package factory

import (
	"fmt"
	"sync"

	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/interfaces"
	"github.com/sukryu/pAuth/internal/store/manager"
	"github.com/sukryu/pAuth/internal/store/role"
	rolebinding "github.com/sukryu/pAuth/internal/store/role_binding"
	"github.com/sukryu/pAuth/internal/store/user"
)

type StoreFactory interface {
	NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error)
	NewRoleStore(cfg *config.DatabaseConfig) (interfaces.RoleStore, error)
	NewRoleBindingStore(cfg *config.DatabaseConfig) (interfaces.RoleBindingStore, error)
	NewDynamicStore(cfg *config.DatabaseConfig) (*dynamic.DynamicStore, error)
	Close() error
	GetStats() map[string]interface{}
}

type storeFactory struct {
	mu             sync.RWMutex
	managers       map[string]manager.Manager
	managerFactory manager.ManagerFactory
}

func NewStoreFactory(managerFactory manager.ManagerFactory) StoreFactory {
	return &storeFactory{
		managers:       make(map[string]manager.Manager),
		managerFactory: managerFactory,
	}
}

func (f *storeFactory) getManager(cfg *config.DatabaseConfig) (manager.Manager, error) {
	f.mu.RLock()
	mgr, exists := f.managers[cfg.GetDSN()]
	f.mu.RUnlock()

	if exists {
		return mgr, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if mgr, exists = f.managers[cfg.GetDSN()]; exists {
		return mgr, nil
	}

	// Create new manager
	mgr, err := f.managerFactory.NewManager(manager.Config{
		Type: cfg.Type,
		DSN:  cfg.GetDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %v", err)
	}

	// Initialize the manager
	if err := mgr.Initialize(nil); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	f.managers[cfg.GetDSN()] = mgr
	return mgr, nil
}

func (f *storeFactory) NewDynamicStore(cfg *config.DatabaseConfig) (*dynamic.DynamicStore, error) {
	manager, err := f.getManager(cfg)
	if err != nil {
		return nil, err
	}

	return dynamic.NewDynamicStore(manager.GetDB()), nil
}

func (f *storeFactory) NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error) {
	dynStore, err := f.NewDynamicStore(cfg)
	if err != nil {
		return nil, err
	}

	return user.NewStore(dynStore, user.Config{
		DatabaseType: cfg.Type,
	})
}

func (f *storeFactory) NewRoleStore(cfg *config.DatabaseConfig) (interfaces.RoleStore, error) {
	dynStore, err := f.NewDynamicStore(cfg)
	if err != nil {
		return nil, err
	}

	return role.NewStore(dynStore, role.Config{
		DatabaseType: cfg.Type,
	})
}

func (f *storeFactory) NewRoleBindingStore(cfg *config.DatabaseConfig) (interfaces.RoleBindingStore, error) {
	dynStore, err := f.NewDynamicStore(cfg)
	if err != nil {
		return nil, err
	}

	return rolebinding.NewStore(dynStore, rolebinding.Config{
		DatabaseType: cfg.Type,
	})
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
