package store_test

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/sukryu/pAuth/internal/store"
	"github.com/sukryu/pAuth/internal/store/dynamic"
	"github.com/sukryu/pAuth/internal/store/schema"
	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupTestUserStore(t *testing.T) (*dynamic.DynamicStore, *store.Store) {
	dbConn, err := SetupTestDB()
	assert.NoError(t, err)

	dynStore := dynamic.NewDynamicStore(dbConn)

	// 스키마 생성
	userSchema := schema.CoreSchemas[0] // users 스키마
	err = dynStore.CreateDynamicTable(context.Background(), userSchema.Name, schema.TableOptions{
		Fields:  userSchema.Fields,
		Indexes: userSchema.Indexes,
	})
	assert.NoError(t, err)

	userStore := store.NewStore(dynStore, store.Config{DatabaseType: "sqlite"})
	return dynStore, userStore
}

func TestUserStore_CreateAndGet(t *testing.T) {
	_, userStore := setupTestUserStore(t)

	ctx := context.Background()

	// 새로운 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user1",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: "hashedpassword",
			Roles:        []string{"admin", "user"},
		},
		Status: v1alpha1.UserStatus{
			Active: true,
		},
	}

	err := userStore.Create(ctx, user)
	assert.NoError(t, err)

	// 사용자 조회
	retrievedUser, err := userStore.Get(ctx, "user1")
	assert.NoError(t, err)
	assert.Equal(t, "testuser", retrievedUser.Spec.Username)
	assert.Equal(t, "testuser@example.com", retrievedUser.Spec.Email)
	assert.Equal(t, []string{"admin", "user"}, retrievedUser.Spec.Roles)
	assert.True(t, retrievedUser.Status.Active)
}

func TestUserStore_UpdateAndDelete(t *testing.T) {
	_, userStore := setupTestUserStore(t)

	ctx := context.Background()

	// 새로운 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user1",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: "hashedpassword",
			Roles:        []string{"admin", "user"},
		},
		Status: v1alpha1.UserStatus{
			Active: true,
		},
	}

	err := userStore.Create(ctx, user)
	assert.NoError(t, err)

	// 사용자 업데이트
	user.Spec.Email = "updated@example.com"
	user.Status.Active = false
	err = userStore.Update(ctx, user)
	assert.NoError(t, err)

	// 업데이트된 사용자 확인
	updatedUser, err := userStore.Get(ctx, "user1")
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", updatedUser.Spec.Email)
	assert.False(t, updatedUser.Status.Active)

	// 사용자 삭제
	err = userStore.Delete(ctx, "user1")
	assert.NoError(t, err)

	// 삭제된 사용자 확인
	_, err = userStore.Get(ctx, "user1")
	assert.Error(t, err)
}

func TestUserStore_FindByEmailAndUsername(t *testing.T) {
	_, userStore := setupTestUserStore(t)

	ctx := context.Background()

	// 사용자 생성
	user := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user1",
		},
		Spec: v1alpha1.UserSpec{
			Username:     "testuser",
			Email:        "testuser@example.com",
			PasswordHash: "hashedpassword",
			Roles:        []string{"admin"},
		},
		Status: v1alpha1.UserStatus{
			Active: true,
		},
	}

	err := userStore.Create(ctx, user)
	assert.NoError(t, err)

	// 이메일로 사용자 조회
	retrievedByEmail, err := userStore.FindByEmail(ctx, "testuser@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "testuser", retrievedByEmail.Spec.Username)

	// 사용자 이름으로 사용자 조회
	retrievedByUsername, err := userStore.FindByUsername(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, "testuser@example.com", retrievedByUsername.Spec.Email)
}

func TestUserStore_ListByRole(t *testing.T) {
	_, userStore := setupTestUserStore(t)

	ctx := context.Background()

	// 사용자 생성
	users := []*v1alpha1.User{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user1",
			},
			Spec: v1alpha1.UserSpec{
				Username:     "adminuser",
				Email:        "admin@example.com",
				PasswordHash: "adminhash",
				Roles:        []string{"admin"},
			},
			Status: v1alpha1.UserStatus{Active: true},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user2",
			},
			Spec: v1alpha1.UserSpec{
				Username:     "normaluser",
				Email:        "user@example.com",
				PasswordHash: "userhash",
				Roles:        []string{"user"},
			},
			Status: v1alpha1.UserStatus{Active: true},
		},
	}

	for _, user := range users {
		err := userStore.Create(ctx, user)
		assert.NoError(t, err)
	}

	// 특정 역할의 사용자 목록 조회
	adminUsers, err := userStore.ListByRole(ctx, "admin")
	assert.NoError(t, err)
	assert.Len(t, adminUsers.Items, 1)
	assert.Equal(t, "adminuser", adminUsers.Items[0].Spec.Username)
}
