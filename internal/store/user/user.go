package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/interfaces"
	"github.com/sukryu/pAuth/internal/store/schema"
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

func NewStore(dynStore *dynamic.DynamicStore, cfg Config) (interfaces.UserStore, error) {
	return &Store{
		dynamicStore: dynStore,
		config:       cfg,
	}, nil
}

func (s *Store) Create(ctx context.Context, user *v1alpha1.User) error {
	// 테이블이 존재하는지 확인 (schema 검증용)
	if _, err := s.dynamicStore.GetTableSchema(ctx, "users"); err != nil {
		return err
	}

	now := time.Now()
	if user.CreationTimestamp.IsZero() {
		user.CreationTimestamp = metav1.NewTime(now)
	}

	// 데이터 맵 생성
	data := make(map[string]interface{})

	// 기본 필드 설정
	coreFields := map[string]interface{}{
		"id":            user.Name,
		"username":      user.Spec.Username,
		"email":         user.Spec.Email,
		"password_hash": user.Spec.PasswordHash,
		"is_active":     user.Status.Active,
		"created_at":    user.CreationTimestamp.Time,
		"updated_at":    now,
	}

	// roles와 last_login 처리
	if len(user.Spec.Roles) > 0 {
		rolesJSON, err := json.Marshal(user.Spec.Roles)
		if err != nil {
			return fmt.Errorf("failed to marshal roles: %w", err)
		}
		coreFields["roles"] = string(rolesJSON)
	}

	if user.Status.LastLogin != nil {
		coreFields["last_login"] = user.Status.LastLogin.Time
	}

	// 기본 필드 복사
	for k, v := range coreFields {
		data[k] = v
	}

	// 사용자 정의 필드 (Annotations) 처리
	if len(user.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(user.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}
		data["annotations"] = string(annotationsJSON)
	}

	// 데이터 삽입
	return s.dynamicStore.DynamicInsert(ctx, "users", data)
}

func (s *Store) Get(ctx context.Context, name string) (*v1alpha1.User, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "users", map[string]interface{}{
		"id": name,
	})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.ErrUserNotFound
	}

	return mapToUser(results[0])
}

func (s *Store) Update(ctx context.Context, user *v1alpha1.User) error {
	// 데이터 업데이트 전에 기존 레코드 확인
	existing, err := s.Get(ctx, user.Name)
	if err != nil {
		return err
	}

	// UNIQUE 필드 충돌 방지: username, email 확인
	if existing.Spec.Username != user.Spec.Username {
		conflictCheck, err := s.FindByUsername(ctx, user.Spec.Username)
		if err == nil && conflictCheck.Name != user.Name {
			return fmt.Errorf("username '%s' already exists", user.Spec.Username)
		}
	}
	if existing.Spec.Email != user.Spec.Email {
		conflictCheck, err := s.FindByEmail(ctx, user.Spec.Email)
		if err == nil && conflictCheck.Name != user.Name {
			return fmt.Errorf("email '%s' already exists", user.Spec.Email)
		}
	}

	// 업데이트 데이터 생성
	data := map[string]interface{}{
		"username":   user.Spec.Username,
		"email":      user.Spec.Email,
		"updated_at": time.Now(),
	}

	// roles와 last_login 처리
	if len(user.Spec.Roles) > 0 {
		rolesJSON, err := json.Marshal(user.Spec.Roles)
		if err != nil {
			return err
		}
		data["roles"] = string(rolesJSON)
	}
	if user.Status.LastLogin != nil {
		data["last_login"] = user.Status.LastLogin.Time
	}

	// Annotations 처리
	annotationsJSON, err := json.Marshal(user.Annotations)
	if err != nil {
		return err
	}
	data["annotations"] = string(annotationsJSON)

	return s.dynamicStore.DynamicUpdate(ctx, "users", user.Name, data)
}

func (s *Store) Delete(ctx context.Context, name string) error {
	return s.dynamicStore.DynamicDelete(ctx, "users", name)
}

func (s *Store) List(ctx context.Context) (*v1alpha1.UserList, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "users", nil)
	if err != nil {
		return nil, err
	}

	userList := &v1alpha1.UserList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserList",
			APIVersion: "auth.service/v1alpha1",
		},
	}

	for _, result := range results {
		user, err := mapToUser(result)
		if err != nil {
			return nil, err
		}
		userList.Items = append(userList.Items, user)
	}

	return userList, nil
}

func mapToUser(data map[string]interface{}) (*v1alpha1.User, error) {
	user := &v1alpha1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       "User",
			APIVersion: "auth.service/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              data["id"].(string),
			CreationTimestamp: metav1.Time{Time: data["created_at"].(time.Time)},
			Annotations:       make(map[string]string),
		},
		Spec: v1alpha1.UserSpec{
			Username:     data["username"].(string),
			Email:        data["email"].(string),
			PasswordHash: data["password_hash"].(string),
		},
		Status: v1alpha1.UserStatus{
			Active: data["is_active"].(bool),
		},
	}

	// Roles 처리
	if roles, ok := data["roles"]; ok && roles != nil {
		var rolesList []string
		if err := json.Unmarshal([]byte(roles.(string)), &rolesList); err != nil {
			return nil, fmt.Errorf("failed to unmarshal roles: %w", err)
		}
		user.Spec.Roles = rolesList
	}

	// LastLogin 처리
	if lastLogin, ok := data["last_login"]; ok && lastLogin != nil {
		lastLoginTime, ok := lastLogin.(time.Time)
		if ok {
			user.Status.LastLogin = &metav1.Time{Time: lastLoginTime}
		}
	}

	// 사용자 정의 필드 (Annotations) 처리
	if annotations, ok := data["annotations"]; ok && annotations != nil {
		var parsedAnnotations map[string]string
		if err := json.Unmarshal([]byte(annotations.(string)), &parsedAnnotations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
		}
		user.Annotations = parsedAnnotations
	}

	return user, nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (*v1alpha1.User, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "users", map[string]interface{}{
		"email": email,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.ErrUserNotFound
	}

	return mapToUser(results[0])
}

func (s *Store) FindByUsername(ctx context.Context, username string) (*v1alpha1.User, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "users", map[string]interface{}{
		"username": username,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.ErrUserNotFound
	}

	return mapToUser(results[0])
}

func (s *Store) UpdatePassword(ctx context.Context, name string, hashedPassword string) error {
	data := map[string]interface{}{
		"password_hash": hashedPassword,
		"updated_at":    time.Now(),
	}

	return s.dynamicStore.DynamicUpdate(ctx, "users", name, data)
}

func (s *Store) UpdateStatus(ctx context.Context, name string, active bool) error {
	data := map[string]interface{}{
		"is_active":  active,
		"updated_at": time.Now(),
	}

	return s.dynamicStore.DynamicUpdate(ctx, "users", name, data)
}

func (s *Store) ListByRole(ctx context.Context, roleName string) (*v1alpha1.UserList, error) {
	results, err := s.dynamicStore.DynamicSelect(ctx, "users", nil)
	if err != nil {
		return nil, err
	}

	userList := &v1alpha1.UserList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UserList",
			APIVersion: "auth.service/v1alpha1",
		},
	}

	for _, result := range results {
		user, err := mapToUser(result)
		if err != nil {
			return nil, err
		}

		// roles 필드 확인
		if rolesStr, ok := result["roles"].(string); ok {
			var roles []string
			if err := json.Unmarshal([]byte(rolesStr), &roles); err != nil {
				return nil, fmt.Errorf("failed to unmarshal roles: %v", err)
			}

			// roleName이 roles 배열에 포함되어 있는지 확인
			for _, role := range roles {
				if role == roleName {
					userList.Items = append(userList.Items, user)
					break
				}
			}
		}
	}

	return userList, nil
}

func parseSchemaFields(schemaFields []string) map[string]schema.FieldType {
	parsedFields := make(map[string]schema.FieldType)

	for _, field := range schemaFields {
		parts := strings.SplitN(field, " ", 2)
		if len(parts) < 2 {
			continue
		}
		fieldName := parts[0]
		fieldType := schema.FieldType(parts[1])
		parsedFields[fieldName] = fieldType
	}

	return parsedFields
}
