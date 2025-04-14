package jwt

import (
	"testing"

	"avito/internal/oapi"

	"github.com/stretchr/testify/assert"
)

const (
	testSecret = "test-secret-1234567890"
)

func TestNewToken(t *testing.T) {
	h := New(testSecret)

	t.Run("valid token generation", func(t *testing.T) {
		expected := oapi.Employee
		token, err := h.NewToken(oapi.UserRole(expected))
		assert.NoError(t, err)

		role, err := h.GetRoleClaim(token)
		assert.NoError(t, err)
		assert.Equal(t, string(expected), string(role))
	})

	t.Run("invalid token", func(t *testing.T) {
		token := "cringe"
		_, err := h.GetRoleClaim(token)
		assert.Error(t, err)
	})
}
