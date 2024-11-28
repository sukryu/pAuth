package role

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

func NewStore(dynStore *dynamic.DynamicStore, cfg Config) (interfaces.RoleStore, error) {
	return &Store{
		dynamicStore: dynStore,
		config:       cfg,
	}, nil
}

func (s *Store) Create(ctx context.Context, role *v1alpha1.Role) error {
	if _, err := s.dynamicStore.GetTableSchema(ctx, "roles"); err != nil {
		return err
	}

	now := time.Now()
	if role.CreationTimestamp.IsZero() {
		role.CreationTimestamp = metav1.NewTime(now)
	}

	rulesJSON, err := json.Marshal(role.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	data := map[string]interface{}{
		"id":          role.Name,
		"name":        role.Name,
		"description": role.Annotations["description"],
		"rules":       string(rulesJSON),
		"created_at":  role.CreationTimestamp.Time,
		"updated_at":  now,
	}

	if len(role.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(role.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		data["annotations"] = string(annotationsJSON)
	}

	return s.dynamicStore.DynamicInsert(ctx, "roles", data)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.Role, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "roles", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.ErrRoleNotFound
	}

	return mapToRole(results[0])
}

func (s *Store) Update(ctx context.Context, role *v1alpha1.Role) error {
	_, err := s.Get(ctx, role.Name)
	if err != nil {
		return err
	}

	rulesJSON, err := json.Marshal(role.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	data := map[string]interface{}{
		"description": role.Annotations["description"],
		"rules":       string(rulesJSON),
		"updated_at":  time.Now(),
	}

	if len(role.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(role.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		data["annotations"] = string(annotationsJSON)
	}

	return s.dynamicStore.DynamicUpdate(ctx, "roles", role.Name, data)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.dynamicStore.DynamicDelete(ctx, "roles", name)
}

func (s *Store) List(ctx context.Context) ([]*v1alpha1.Role, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "roles", nil)
	if err != nil {
		return nil, err
	}

	roles := make([]*v1alpha1.Role, 0, len(results))
	for _, result := range results {
		role, err := mapToRole(result)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (s *Store) FindByVerb(ctx context.Context, verb string) ([]*v1alpha1.Role, error) {
	roles, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*v1alpha1.Role
	for _, role := range roles {
		for _, rule := range role.Rules {
			for _, v := range rule.Verbs {
				if v == verb {
					filtered = append(filtered, role)
					break
				}
			}
		}
	}

	return filtered, nil
}

func (s *Store) FindByResource(ctx context.Context, resource string) ([]*v1alpha1.Role, error) {
	roles, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*v1alpha1.Role
	for _, role := range roles {
		for _, rule := range role.Rules {
			for _, r := range rule.Resources {
				if r == resource {
					filtered = append(filtered, role)
					break
				}
			}
		}
	}

	return filtered, nil
}

func (s *Store) FindByAPIGroup(ctx context.Context, apiGroup string) ([]*v1alpha1.Role, error) {
	roles, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*v1alpha1.Role
	for _, role := range roles {
		for _, rule := range role.Rules {
			for _, g := range rule.APIGroups {
				if g == apiGroup {
					filtered = append(filtered, role)
					break
				}
			}
		}
	}

	return filtered, nil
}

func (s *Store) UpdateRules(ctx context.Context, name string, rules []v1alpha1.PolicyRule) error {
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	data := map[string]interface{}{
		"rules":      string(rulesJSON),
		"updated_at": time.Now(),
	}

	return s.dynamicStore.DynamicUpdate(ctx, "roles", name, data)
}

func (s *Store) ListBySubject(ctx context.Context, subjectKind, subjectName string) ([]*v1alpha1.Role, error) {
	// 이 메서드는 RoleBindingStore와 함께 구현되어야 합니다
	// 현재는 미구현 상태입니다
	return nil, errors.ErrNotImplemented
}

func mapToRole(data map[string]interface{}) (*v1alpha1.Role, error) {
	role := &v1alpha1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              data["name"].(string),
			CreationTimestamp: metav1.Time{Time: data["created_at"].(time.Time)},
			Annotations:       make(map[string]string),
		},
	}

	if description, ok := data["description"]; ok && description != nil {
		role.Annotations["description"] = description.(string)
	}

	if rulesJSON, ok := data["rules"].(string); ok {
		var rules []v1alpha1.PolicyRule
		if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
		}
		role.Rules = rules
	}

	if annotations, ok := data["annotations"].(string); ok && annotations != "" {
		var parsedAnnotations map[string]string
		if err := json.Unmarshal([]byte(annotations), &parsedAnnotations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
		}
		role.Annotations = parsedAnnotations
	}

	return role, nil
}
