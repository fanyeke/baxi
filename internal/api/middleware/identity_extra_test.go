package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrNotJWT_Error(t *testing.T) {
	err := ErrNotJWT
	assert.Equal(t, "token is not in JWT format", err.Error())
}

func TestErrNotJWT_IsError(t *testing.T) {
	var err error = ErrNotJWT
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT")
}
