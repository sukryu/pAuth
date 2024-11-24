package sqlite

import (
	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/pkg/store/factory"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type sqliteStoreFactory struct{}

func NewSQLiteStoreFactory() factory.StoreFactory {
	return &sqliteStoreFactory{}
}

func (f *sqliteStoreFactory) NewUserStore(cfg *config.DatabaseConfig) (interfaces.UserStore, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Database), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return NewUserStore(db), nil
}
