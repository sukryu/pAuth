// pkg/store/rolebinding/store.go

package rolebinding

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
	base base.Store[*v1alpha1.RoleBinding]
}

func NewStore(db *gorm.DB, dbType string) (interfaces.RoleBindingStore, error) {
	var baseStore base.Store[*v1alpha1.RoleBinding]

	switch dbType {
	case "sqlite":
		baseStore = sqlite.NewSQLiteStore[*v1alpha1.RoleBinding](db, "role_bindings")
	case "postgresql":
		return nil, errors.ErrNotImplemented.WithReason("postgresql support not implemented yet")
	default:
		return nil, errors.ErrInvalidInput.WithReason(fmt.Sprintf("unsupported database type: %s", dbType))
	}

	return &Store{
		base: baseStore,
	}, nil
}

func (s *Store) Create(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	return s.base.Create(ctx, binding)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.RoleBinding, error) {
	return s.base.Get(ctx, name)
}

func (s *Store) Update(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	return s.base.Update(ctx, binding)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.base.Delete(ctx, name)
}

func (s *Store) List(ctx context.Context) ([]*v1alpha1.RoleBinding, error) {
	return s.base.List(ctx)
}

// Additional rolebinding-specific operations
func (s *Store) FindBySubject(ctx context.Context, subjectKind, subjectName string) ([]*v1alpha1.RoleBinding, error) {
	var bindings []*v1alpha1.RoleBinding
	result := s.base.GetDB().WithContext(ctx).
		Where("subjects @> ?", fmt.Sprintf(`[{"kind":"%s","name":"%s"}]`, subjectKind, subjectName)).
		Find(&bindings)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return bindings, nil
}

func (s *Store) FindByRole(ctx context.Context, roleName string) ([]*v1alpha1.RoleBinding, error) {
	var bindings []*v1alpha1.RoleBinding
	result := s.base.GetDB().WithContext(ctx).
		Where("role_ref->>'name' = ?", roleName).
		Find(&bindings)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return bindings, nil
}

func (s *Store) AddSubject(ctx context.Context, name string, subject v1alpha1.Subject) error {
	binding, err := s.Get(ctx, name)
	if err != nil {
		return err
	}

	// Check if subject already exists
	for _, existingSubject := range binding.Subjects {
		if existingSubject.Kind == subject.Kind && existingSubject.Name == subject.Name {
			return nil // Subject already exists
		}
	}

	// Add new subject
	binding.Subjects = append(binding.Subjects, subject)
	return s.Update(ctx, binding)
}

func (s *Store) RemoveSubject(ctx context.Context, name string, subject v1alpha1.Subject) error {
	binding, err := s.Get(ctx, name)
	if err != nil {
		return err
	}

	// Filter out the subject to remove
	newSubjects := make([]v1alpha1.Subject, 0, len(binding.Subjects))
	for _, existingSubject := range binding.Subjects {
		if existingSubject.Kind != subject.Kind || existingSubject.Name != subject.Name {
			newSubjects = append(newSubjects, existingSubject)
		}
	}

	if len(newSubjects) == len(binding.Subjects) {
		return errors.ErrNotFound.WithReason(fmt.Sprintf("subject %s/%s not found in binding", subject.Kind, subject.Name))
	}

	binding.Subjects = newSubjects
	return s.Update(ctx, binding)
}

func (s *Store) ListByNamespace(ctx context.Context, namespace string) ([]*v1alpha1.RoleBinding, error) {
	var bindings []*v1alpha1.RoleBinding
	result := s.base.GetDB().WithContext(ctx).
		Where("metadata->>'namespace' = ?", namespace).
		Find(&bindings)

	if result.Error != nil {
		return nil, errors.ErrStorageOperation.WithReason(result.Error.Error())
	}
	return bindings, nil
}
