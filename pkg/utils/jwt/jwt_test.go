package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTManager(t *testing.T) {
	secretKey := "test-secret-key"
	expiry := 1 * time.Hour
	manager := NewJWTManager(secretKey, expiry)

	t.Run("Generate and Validate Token", func(t *testing.T) {
		// Test data
		userID := "test-user"
		roles := []string{"admin", "user"}

		// Generate token
		token, err := manager.GenerateToken(userID, roles)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate token
		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, roles, claims.Roles)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		invalidToken := "invalid.token.string"
		claims, err := manager.ValidateToken(invalidToken)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Expired Token", func(t *testing.T) {
		// Create manager with very short expiry
		shortManager := NewJWTManager(secretKey, 1*time.Nanosecond)

		token, err := shortManager.GenerateToken("user", []string{"role"})
		assert.NoError(t, err)

		// Wait for token to expire
		time.Sleep(1 * time.Millisecond)

		claims, err := shortManager.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "token is expired")
	})

	t.Run("Token with Different Signing Method", func(t *testing.T) {
		// This test attempts to validate a token that was signed with a different method
		invalidToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."
		claims, err := manager.ValidateToken(invalidToken)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Generate Token with Empty User ID", func(t *testing.T) {
		token, err := manager.GenerateToken("", []string{"role"})
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Empty(t, claims.UserID)
	})

	t.Run("Generate Token with Nil Roles", func(t *testing.T) {
		token, err := manager.GenerateToken("user", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Empty(t, claims.Roles)
	})

	t.Run("Generate Token with Empty Roles", func(t *testing.T) {
		token, err := manager.GenerateToken("user", []string{})
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.Empty(t, claims.Roles)
	})

	t.Run("Validate Token with Different Secret", func(t *testing.T) {
		// Generate token with original manager
		token, err := manager.GenerateToken("user", []string{"role"})
		assert.NoError(t, err)

		// Create new manager with different secret
		differentManager := NewJWTManager("different-secret", expiry)
		claims, err := differentManager.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestClaimsType(t *testing.T) {
	claims := &Claims{
		UserID: "test-user",
		Roles:  []string{"admin"},
	}

	t.Run("Claims Fields", func(t *testing.T) {
		assert.Equal(t, "test-user", claims.UserID)
		assert.Equal(t, []string{"admin"}, claims.Roles)
	})
}
