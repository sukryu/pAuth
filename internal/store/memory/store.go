package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/sukryu/pAuth/pkg/apis/auth/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type memoryStore struct {
	mu    sync.RWMutex
	users map[string]*v1alpha1.User
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{
		users: make(map[string]*v1alpha1.User),
	}
}

func (s *memoryStore) Create(ctx context.Context, user *v1alpha1.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Name]; exists {
		return fmt.Errorf("user %s already exists", user.Name)
	}

	// Deep copy the user object
	s.users[user.Name] = user.DeepCopy()
	return nil
}

func (s *memoryStore) Get(ctx context.Context, name string) (*v1alpha1.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[name]
	if !exists {
		return nil, fmt.Errorf("user %s not found", name)
	}

	return user.DeepCopy(), nil
}

func (s *memoryStore) Update(ctx context.Context, user *v1alpha1.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Name]; !exists {
		return fmt.Errorf("user %s not found", user.Name)
	}

	s.users[user.Name] = user.DeepCopy()
	return nil
}

func (s *memoryStore) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; !exists {
		return fmt.Errorf("user %s not found", name)
	}

	delete(s.users, name)
	return nil
}

func (s *memoryStore) List(ctx context.Context) (*v1alpha1.UserList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &v1alpha1.UserList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "auth.service/v1alpha1",
			Kind:       "UserList",
		},
		Items: make([]v1alpha1.User, 0, len(s.users)),
	}

	for _, user := range s.users {
		list.Items = append(list.Items, *user.DeepCopy())
	}

	return list, nil
}
