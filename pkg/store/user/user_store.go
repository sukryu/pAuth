package user

import (
	"context"
	"fmt"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/store/base"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
	"github.com/sukryu/pAuth/pkg/store/sqlite"
	"gorm.io/gorm"
)

// Config defines the configuration for UserStore
type Config struct {
	DatabaseType string
	DB           *gorm.DB
}

// Store implements interfaces.UserStore
type Store struct {
	base base.Store[*v1alpha1.User]
}

// NewStore creates a new UserStore with the specified database type
func NewStore(cfg Config) (interfaces.UserStore, error) {
	var baseStore base.Store[*v1alpha1.User]

	switch cfg.DatabaseType {
	case "sqlite":
		baseStore = sqlite.NewSQLiteStore[*v1alpha1.User](cfg.DB, "users")
	case "postgresql":
		return nil, errors.ErrNotImplemented.WithReason("postgresql support not implemented yet")
	default:
		return nil, errors.ErrInvalidInput.WithReason(fmt.Sprintf("unsupported database type: %s", cfg.DatabaseType))
	}

	return &Store{
		base: baseStore,
	}, nil
}

// Basic CRUD operations
func (s *Store) Create(ctx context.Context, user *v1alpha1.User) error {
	return s.base.Create(ctx, user)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.User, error) {
	return s.base.Get(ctx, name)
}

func (s *Store) Update(ctx context.Context, user *v1alpha1.User) error {
	return s.base.Update(ctx, user)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.base.Delete(ctx, name)
}

func (s *Store) List(ctx context.Context) (*v1alpha1.UserList, error) {
	users, err := s.base.List(ctx)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.UserList{Items: users}, nil
}

// Additional user-specific operations
func (s *Store) FindByEmail(ctx context.Context, email string) (*v1alpha1.User, error) {
	var user v1alpha1.User
	result := s.base.GetDB().WithContext(ctx).
		Where("spec->>'email' = ?", email).
		First(&user)

	if result.Error != nil {
		if s.base.IsNotFound(result.Error) {
			return nil, errors.ErrUserNotFound.WithReason(fmt.Sprintf("email: %s", email))
		}
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return &user, nil
}

func (s *Store) FindByUsername(ctx context.Context, username string) (*v1alpha1.User, error) {
	var user v1alpha1.User
	result := s.base.GetDB().WithContext(ctx).
		Where("spec->>'username' = ?", username).
		First(&user)

	if result.Error != nil {
		if s.base.IsNotFound(result.Error) {
			return nil, errors.ErrUserNotFound.WithReason(fmt.Sprintf("username: %s", username))
		}
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return &user, nil
}

func (s *Store) UpdatePassword(ctx context.Context, name string, hashedPassword string) error {
	result := s.base.GetDB().WithContext(ctx).
		Model(&v1alpha1.User{}).
		Where("name = ?", name).
		Update("spec->>'passwordHash'", hashedPassword)

	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound.WithReason(name)
	}
	return nil
}

func (s *Store) UpdateStatus(ctx context.Context, name string, active bool) error {
	result := s.base.GetDB().WithContext(ctx).
		Model(&v1alpha1.User{}).
		Where("name = ?", name).
		Update("status->>'active'", active)

	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrUserNotFound.WithReason(name)
	}
	return nil
}

func (s *Store) ListByRole(ctx context.Context, roleName string) (*v1alpha1.UserList, error) {
	var users []*v1alpha1.User
	result := s.base.GetDB().WithContext(ctx).
		Joins("JOIN role_bindings ON users.name = role_bindings.subject_name").
		Where("role_bindings.role_ref = ? AND role_bindings.subject_kind = ?", roleName, "User").
		Find(&users)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}

	return &v1alpha1.UserList{Items: users}, nil
}

// Transaction executes operations within a transaction
func (s *Store) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.base.Transaction(ctx, fn)
}
