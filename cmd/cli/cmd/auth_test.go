package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthCommandStructure(t *testing.T) {
	assert.Equal(t, "auth", authCmd.Use)
	assert.Equal(t, "Authentication commands", authCmd.Short)
}

func TestLoginCommandStructure(t *testing.T) {
	assert.Equal(t, "login", loginCmd.Use)
	assert.Equal(t, "Login to file vault", loginCmd.Short)
	assert.Contains(t, loginCmd.Aliases, "in")
}

func TestRegisterCommandStructure(t *testing.T) {
	assert.Equal(t, "register", registerCmd.Use)
	assert.Equal(t, "Register a new account", registerCmd.Short)
	assert.Contains(t, registerCmd.Aliases, "reg")
}

func TestLogoutCommandStructure(t *testing.T) {
	assert.Equal(t, "logout", logoutCmd.Use)
	assert.Equal(t, "log out from app", logoutCmd.Short)
	assert.Contains(t, logoutCmd.Aliases, "out")
}

func TestStatusCommandStructure(t *testing.T) {
	assert.Equal(t, "status", statusCmd.Use)
	assert.Equal(t, "Check login status", statusCmd.Short)
	assert.Contains(t, statusCmd.Aliases, "st")
}