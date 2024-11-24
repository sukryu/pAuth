package sqlite

import (
	"context"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
	"gorm.io/gorm"
)

type userStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) interfaces.UserStore {
	return &userStore{db: db}
}

func (s *userStore) Create(ctx context.Context, user *v1alpha1.User) error {
	result := s.db.WithContext(ctx).Create(user)
	return result.Error
}
