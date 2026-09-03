package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTSecret(t *testing.T) {
	t.Run("returns configured secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-secret")

		secret, err := JWTSecret()

		require.NoError(t, err)
		assert.Equal(t, "test-secret", secret)
	})

	t.Run("returns error when secret is missing", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "")

		secret, err := JWTSecret()

		assert.ErrorIs(t, err, ErrJWTSecretMissing)
		assert.Equal(t, "", secret)
	})
}
