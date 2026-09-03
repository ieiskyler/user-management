package config

import (
	"errors"
	"os"
)

var ErrJWTSecretMissing = errors.New("JWT_SECRET is not configured")

func JWTSecret() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", ErrJWTSecretMissing
	}

	return secret, nil
}
