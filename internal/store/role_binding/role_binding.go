package rolebinding

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/interfaces"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	"github.com/sukryu/pAuth/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	DatabaseType string
}

type Store struct {
	dynamicStore *dynamic.DynamicStore
	config       Config
}

func NewStore(dynStore *dynamic.DynamicStore, cfg Config) (interfaces.RoleBindingStore, error) {
	return &Store{
		dynamicStore: dynStore,
		config:       cfg,
	}, nil
}

func (s *Store) Create(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	if _, err := s.dynamicStore.GetTableSchema(ctx, "role_bindings"); err != nil {
		return err
	}

	now := time.Now()
	if binding.CreationTimestamp.IsZero() {
		binding.CreationTimestamp = metav1.NewTime(now)
	}

	subjectsJSON, err := json.Marshal(binding.Subjects)
	if err != nil {
		return fmt.Errorf("failed to marshal subjects: %w", err)
	}

	data := map[string]interface{}{
		"id":         binding.Name,
		"name":       binding.Name,
		"role_ref":   binding.RoleRef.Name,
		"subjects":   string(subjectsJSON),
		"created_at": binding.CreationTimestamp.Time,
		"updated_at": now,
	}

	if len(binding.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(binding.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		data["annotations"] = string(annotationsJSON)
	}

	return s.dynamicStore.DynamicInsert(ctx, "role_bindings", data)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.RoleBinding, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "role_bindings", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.ErrRoleBindingNotFound
	}

	return mapToRoleBinding(results[0])
}

func (s *Store) Update(ctx context.Context, binding *v1alpha1.RoleBinding) error {
	_, err := s.Get(ctx, binding.Name)
	if err != nil {
		return err
	}

	subjectsJSON, err := json.Marshal(binding.Subjects)
	if err != nil {
		return fmt.Errorf("failed to marshal subjects: %w", err)
	}

	data := map[string]interface{}{
		"role_ref":   binding.RoleRef.Name,
		"subjects":   string(subjectsJSON),
		"updated_at": time.Now(),
	}

	if len(binding.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(binding.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		data["annotations"] = string(annotationsJSON)
	}

	return s.dynamicStore.DynamicUpdate(ctx, "role_bindings", binding.Name, data)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.dynamicStore.DynamicDelete(ctx, "role_bindings", name)
}

func (s *Store) List(ctx context.Context) ([]*v1alpha1.RoleBinding, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "role_bindings", nil)
	if err != nil {
		return nil, err
	}

	bindings := make([]*v1alpha1.RoleBinding, 0, len(results))
	for _, result := range results {
		binding, err := mapToRoleBinding(result)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

func (s *Store) FindBySubject(ctx context.Context, subjectKind, subjectName string) ([]*v1alpha1.RoleBinding, error) {
	bindings, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*v1alpha1.RoleBinding
	for _, binding := range bindings {
		for _, subject := range binding.Subjects {
			if subject.Kind == subjectKind && subject.Name == subjectName {
				filtered = append(filtered, binding)
				break
			}
		}
	}

	return filtered, nil
}

func (s *Store) FindByRole(ctx context.Context, roleName string) ([]*v1alpha1.RoleBinding, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "role_bindings", map[string]interface{}{
		"role_ref": roleName,
	})
	if err != nil {
		return nil, err
	}

	bindings := make([]*v1alpha1.RoleBinding, 0, len(results))
	for _, result := range results {
		binding, err := mapToRoleBinding(result)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

func (s *Store) AddSubject(ctx context.Context, name string, subject v1alpha1.Subject) error {
	binding, err := s.Get(ctx, name)
	if err != nil {
		return err
	}

	// 이미 존재하는 subject인지 확인
	for _, existing := range binding.Subjects {
		if existing.Kind == subject.Kind && existing.Name == subject.Name {
			return fmt.Errorf("subject already exists in binding")
		}
	}

	binding.Subjects = append(binding.Subjects, subject)
	return s.Update(ctx, binding)
}

func (s *Store) RemoveSubject(ctx context.Context, name string, subject v1alpha1.Subject) error {
	binding, err := s.Get(ctx, name)
	if err != nil {
		return err
	}

	var newSubjects []v1alpha1.Subject
	found := false
	for _, existing := range binding.Subjects {
		if existing.Kind == subject.Kind && existing.Name == subject.Name {
			found = true
			continue
		}
		newSubjects = append(newSubjects, existing)
	}

	if !found {
		return fmt.Errorf("subject not found in binding")
	}

	binding.Subjects = newSubjects
	return s.Update(ctx, binding)
}

// func (s *Store) ListByNamespace(ctx context.Context, namespace string) ([]*v1alpha1.RoleBinding, error) {
// 	bindings, err := s.List(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var filtered []*v1alpha1.RoleBinding
// 	for _, binding := range bindings {
// 		if binding.Namespace == namespace {
// 			filtered = append(filtered, binding)
// 		}
// 	}

// 	return filtered, nil
// }

func mapToRoleBinding(data map[string]interface{}) (*v1alpha1.RoleBinding, error) {
	binding := &v1alpha1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              data["name"].(string),
			CreationTimestamp: metav1.Time{Time: data["created_at"].(time.Time)},
			Annotations:       make(map[string]string),
		},
		RoleRef: v1alpha1.RoleRef{
			Kind: "Role",
			Name: data["role_ref"].(string),
		},
	}

	if subjectsJSON, ok := data["subjects"].(string); ok && subjectsJSON != "" {
		var subjects []v1alpha1.Subject
		if err := json.Unmarshal([]byte(subjectsJSON), &subjects); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subjects: %w", err)
		}
		binding.Subjects = subjects
	}

	if annotations, ok := data["annotations"].(string); ok && annotations != "" {
		var parsedAnnotations map[string]string
		if err := json.Unmarshal([]byte(annotations), &parsedAnnotations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
		}
		binding.Annotations = parsedAnnotations
	}

	return binding, nil
}
