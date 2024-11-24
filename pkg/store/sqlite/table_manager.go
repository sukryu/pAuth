package sqlite

import (
	"context"

	"gorm.io/gorm"
)

type TableManager struct {
	db *gorm.DB
}

func NewTableManager(db *gorm.DB) *TableManager {
	return &TableManager{db: db}
}

func (tm *TableManager) Initialize(ctx context.Context) error {
	// EntitySchema 테이블 생성
	if err := tm.db.AutoMigrate(&EntitySchema{}); err != nil {
		return err
	}

	// 코어 스키마 생성
	for _, schema := range CoreSchemas {
		if err := tm.CreateTable(ctx, &schema); err != nil {
			return err
		}
	}

	return nil
}
