package sqlite

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/store/base"
)

type SQLiteStore[T base.Object] struct {
	db        *gorm.DB
	tableName string
}

func NewSQLiteStore[T base.Object](db *gorm.DB, tableName string) base.Store[T] {
	return &SQLiteStore[T]{
		db:        db,
		tableName: tableName,
	}
}

func (s *SQLiteStore[T]) Create(ctx context.Context, obj T) error {
	if obj.GetName() == "" {
		return errors.ErrInvalidInput.WithReason("name cannot be empty")
	}

	result := s.db.WithContext(ctx).Create(obj)
	if result.Error != nil {
		if s.IsUniqueViolation(result.Error) {
			return errors.ErrAlreadyExists.WithReason(fmt.Sprintf("%s/%s", s.tableName, obj.GetName()))
		}
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return nil
}

func (s *SQLiteStore[T]) Get(ctx context.Context, name string) (T, error) {
	var obj T
	result := s.db.WithContext(ctx).Where("name = ?", name).First(&obj)
	if result.Error != nil {
		var zero T
		if s.IsNotFound(result.Error) {
			return zero, errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", s.tableName, name))
		}
		return zero, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return obj, nil
}

func (s *SQLiteStore[T]) Update(ctx context.Context, obj T) error {
	if obj.GetName() == "" {
		return errors.ErrInvalidInput.WithReason("name cannot be empty")
	}

	result := s.db.WithContext(ctx).Where("name = ?", obj.GetName()).Updates(obj)
	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", s.tableName, obj.GetName()))
	}
	return nil
}

func (s *SQLiteStore[T]) Delete(ctx context.Context, name string) error {
	result := s.db.WithContext(ctx).Where("name = ?", name).Delete(new(T))
	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("%s/%s", s.tableName, name))
	}
	return nil
}

func (s *SQLiteStore[T]) List(ctx context.Context) ([]T, error) {
	var objects []T
	result := s.db.WithContext(ctx).Find(&objects)
	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return objects, nil
}

func (s *SQLiteStore[T]) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

func (s *SQLiteStore[T]) IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func (s *SQLiteStore[T]) IsUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *SQLiteStore[T]) GetDB() *gorm.DB {
	return s.db
}

func (s *SQLiteStore[T]) GetTableName() string {
	return s.tableName
}
