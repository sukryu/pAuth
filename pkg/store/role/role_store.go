package role

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	"github.com/sukryu/pAuth/pkg/store/base"
	"github.com/sukryu/pAuth/pkg/store/interfaces"
	"github.com/sukryu/pAuth/pkg/store/sqlite"
)

type Store struct {
	base base.Store[*v1alpha1.Role]
}

func NewStore(db *gorm.DB, dbType string) (interfaces.RoleStore, error) {
	var baseStore base.Store[*v1alpha1.Role]

	switch dbType {
	case "sqlite":
		baseStore = sqlite.NewSQLiteStore[*v1alpha1.Role](db, "roles")
	case "postgresql":
		return nil, errors.ErrNotImplemented.WithReason("postgresql support not implemented yet")
	default:
		return nil, errors.ErrInvalidInput.WithReason(fmt.Sprintf("unsupported database type: %s", dbType))
	}

	return &Store{
		base: baseStore,
	}, nil
}

func (s *Store) Create(ctx context.Context, role *v1alpha1.Role) error {
	return s.base.Create(ctx, role)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.Role, error) {
	return s.base.Get(ctx, name)
}

func (s *Store) Update(ctx context.Context, role *v1alpha1.Role) error {
	return s.base.Update(ctx, role)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.base.Delete(ctx, name)
}

func (s *Store) List(ctx context.Context) ([]*v1alpha1.Role, error) {
	return s.base.List(ctx)
}

// Additional role-specific operations
func (s *Store) FindByVerb(ctx context.Context, verb string) ([]*v1alpha1.Role, error) {
	var roles []*v1alpha1.Role
	result := s.base.GetDB().WithContext(ctx).
		Where("rules->>'verbs' LIKE ?", fmt.Sprintf("%%\"%s\"%%", verb)).
		Find(&roles)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return roles, nil
}

func (s *Store) FindByResource(ctx context.Context, resource string) ([]*v1alpha1.Role, error) {
	var roles []*v1alpha1.Role
	result := s.base.GetDB().WithContext(ctx).
		Where("rules->>'resources' LIKE ?", fmt.Sprintf("%%\"%s\"%%", resource)).
		Find(&roles)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return roles, nil
}

func (s *Store) FindByAPIGroup(ctx context.Context, apiGroup string) ([]*v1alpha1.Role, error) {
	var roles []*v1alpha1.Role
	result := s.base.GetDB().WithContext(ctx).
		Where("rules->>'apiGroups' LIKE ?", fmt.Sprintf("%%\"%s\"%%", apiGroup)).
		Find(&roles)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return roles, nil
}

func (s *Store) UpdateRules(ctx context.Context, name string, rules []v1alpha1.PolicyRule) error {
	result := s.base.GetDB().WithContext(ctx).
		Model(&v1alpha1.Role{}).
		Where("name = ?", name).
		Update("rules", rules)

	if result.Error != nil {
		return errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return errors.ErrRoleNotFound.WithReason(name)
	}
	return nil
}

func (s *Store) ListBySubject(ctx context.Context, subjectKind, subjectName string) ([]*v1alpha1.Role, error) {
	var roles []*v1alpha1.Role
	result := s.base.GetDB().WithContext(ctx).
		// role_bindings 테이블과 조인하여 특정 subject에 할당된 role들을 찾음
		Joins("JOIN role_bindings ON roles.name = role_bindings.role_ref->>'name'").
		// subject 조건 확인
		Where("role_bindings.subjects @> ?", fmt.Sprintf(`[{"kind":"%s","name":"%s"}]`, subjectKind, subjectName)).
		Find(&roles)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return roles, nil
}
