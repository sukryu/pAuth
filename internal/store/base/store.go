package base

import (
	"context"
	"database/sql"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Object defines the minimal interface that all stored objects must implement
type Object interface {
	GetName() string
	SetName(string)
	GetObjectMeta() v1.Object
}

// Store defines the interface for all database implementations
type Store[T Object] interface {
	// Basic CRUD operations
	Create(ctx context.Context, obj T) error
	Get(ctx context.Context, name string) (T, error)
	Update(ctx context.Context, obj T) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]T, error)

	// Transaction support
	Transaction(ctx context.Context, fn func(tx *sql.Tx) error) error

	// Database specific operations
	IsNotFound(err error) bool
	IsUniqueViolation(err error) bool
	GetDB() *sql.DB
	GetTableName() string
}
