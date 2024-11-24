package factory

import (
	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
)

type StoreFactory interface {
	NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error)
	NewRoleStore(cfg *config.DatabaseConfig) (interfaces.RoleStore, error)
	NewRoleBindingStore(cfg *config.DatabaseConfig) (interfaces.RoleBindingStore, error)
}
