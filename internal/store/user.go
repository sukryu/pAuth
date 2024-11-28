package store

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
	dynamicStore dynamic.DynamicStore
	config       Config
}

func NewStore(dynStore dynamic.DynamicStore, cfg Config) (interfaces.UserStore, error) {
	return &Store{
		dynamicStore: dynStore,
		config:       cfg,
	}, nil
}

func (s *Store) Create(ctx context.Context, user *v1alpha1.User) error {
	// 스키마 가져오기
	schema, err := s.dynamicStore.GetTableSchema(ctx, "users")
	if err != nil {
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

	// roles와 last_login은 별도 처리 (JSON/NULL 가능)
	if len(user.Spec.Roles) > 0 {
		rolesJSON, err := json.Marshal(user.Spec.Roles)
		if err != nil {
			return err
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

	// 추가 필드 처리 (annotations에서)
	for key, value := range user.Annotations {
		// 해당 필드가 스키마에 존재하는지 확인
		for _, field := range schema {
			if key == field {
				data[key] = value
				break
			}
		}
	}

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
	// 스키마 정보 가져오기
	schema, err := s.dynamicStore.GetTableSchema(ctx, "users")
	if err != nil {
		return err
	}

	data := make(map[string]interface{})

	// 기본 필드 업데이트
	baseFields := map[string]interface{}{
		"username":   user.Spec.Username,
		"email":      user.Spec.Email,
		"is_active":  user.Status.Active,
		"updated_at": time.Now(),
	}

	// roles와 last_login 처리
	if len(user.Spec.Roles) > 0 {
		rolesJSON, err := json.Marshal(user.Spec.Roles)
		if err != nil {
			return err
		}
		baseFields["roles"] = string(rolesJSON)
	}

	if user.Status.LastLogin != nil {
		baseFields["last_login"] = user.Status.LastLogin.Time
	}

	// 기본 필드 복사
	for k, v := range baseFields {
		data[k] = v
	}

	// 추가 필드 처리
	for key, value := range user.Annotations {
		for _, field := range schema {
			if key == field {
				data[key] = value
				break
			}
		}
	}

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
			return nil, fmt.Errorf("failed to unmarshal roles: %v", err)
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

	// 추가 필드 처리
	coreFields := map[string]bool{
		"id": true, "username": true, "email": true,
		"password_hash": true, "is_active": true, "roles": true,
		"last_login": true, "created_at": true, "updated_at": true,
	}

	for key, value := range data {
		if !coreFields[key] && value != nil {
			user.Annotations[key] = fmt.Sprint(value)
		}
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
