package base

import (
	"context"

	"gorm.io/gorm"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Object defines the minimal interface that all stored objects must implement
type Object interface {
	GetName() string
	SetName(string)
	GetObjectMeta() v1.Object // metav1.ObjectMeta로 변경
}

// Store defines the interface for all database implementations
type Store[T Object] interface {
	// Basic CRUD operations
	Create(ctx context.Context, obj T) error
	Get(ctx context.Context, name string) (T, error)
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]T, error) // 포인터 슬라이스로 변경

	// Transaction support
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

	// Database specific operations
	IsNotFound(err error) bool
	IsUniqueViolation(err error) bool
	GetDB() *gorm.DB
	GetTableName() string
}
