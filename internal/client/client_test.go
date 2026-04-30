package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveAndLoadTokens(t *testing.T) {
	// Create a temp directory to simulate home directory
	tmpDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Override the path used by the client
	// We'll test by calling the functions that use it
	testDir := filepath.Join(tmpDir, ".file-vault")
	os.MkdirAll(testDir, 0o700)

	// Test saving tokens
	err := saveTokens("test-access-token", "test-refresh-token")
	assert.NoError(t, err)

	// Test loading config
	cfg := loadConfig()
	assert.Equal(t, "test-access-token", cfg.AccessToken)
	assert.Equal(t, "test-refresh-token", cfg.RefreshToken)
}

func TestLoadTokenWhenNoConfig(t *testing.T) {
	// Create a temp directory with no config
	tmpDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Load token when no config exists
	token := LoadToken()
	assert.Empty(t, token)
}

func TestConfigStruct(t *testing.T) {
	// Test that config struct has correct fields
	cfg := config{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}

	assert.Equal(t, "access", cfg.AccessToken)
	assert.Equal(t, "refresh", cfg.RefreshToken)
}

func TestLoadConfigWithInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	testDir := filepath.Join(tmpDir, ".file-vault")
	os.MkdirAll(testDir, 0o700)

	// Write invalid JSON
	invalidConfigPath := filepath.Join(testDir, "config.json")
	os.WriteFile(invalidConfigPath, []byte("{invalid json"), 0o600)

	// Should return empty config
	cfg := loadConfig()
	assert.Empty(t, cfg.AccessToken)
	assert.Empty(t, cfg.RefreshToken)
}

func TestLoadConfigWithEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	testDir := filepath.Join(tmpDir, ".file-vault")
	os.MkdirAll(testDir, 0o700)

	// Write empty file
	emptyConfigPath := filepath.Join(testDir, "config.json")
	os.WriteFile(emptyConfigPath, []byte(""), 0o600)

	// Should return empty config (empty JSON unmarshal)
	cfg := loadConfig()
	assert.Empty(t, cfg.AccessToken)
	assert.Empty(t, cfg.RefreshToken)
}