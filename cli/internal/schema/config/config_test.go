package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Success(t *testing.T) {
	// Store original values to restore later
	originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	originalURL := os.Getenv("RUDDERSTACK_API_URL")

	// Set test environment
	os.Setenv("RUDDERSTACK_ACCESS_TOKEN", "test-token")
	os.Setenv("RUDDERSTACK_API_URL", "https://api.test.com")

	defer func() {
		if originalToken == "" {
			os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
		} else {
			os.Setenv("RUDDERSTACK_ACCESS_TOKEN", originalToken)
		}
		if originalURL == "" {
			os.Unsetenv("RUDDERSTACK_API_URL")
		} else {
			os.Setenv("RUDDERSTACK_API_URL", originalURL)
		}
	}()

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "test-token", cfg.APIToken)
	assert.Equal(t, "https://api.test.com", cfg.APIURL)
}

func TestLoadConfig_MissingToken(t *testing.T) {
	// Store original values to restore later
	originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	originalURL := os.Getenv("RUDDERSTACK_API_URL")

	// Set test environment
	os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
	os.Setenv("RUDDERSTACK_API_URL", "https://api.test.com")

	defer func() {
		if originalToken == "" {
			os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
		} else {
			os.Setenv("RUDDERSTACK_ACCESS_TOKEN", originalToken)
		}
		if originalURL == "" {
			os.Unsetenv("RUDDERSTACK_API_URL")
		} else {
			os.Setenv("RUDDERSTACK_API_URL", originalURL)
		}
	}()

	cfg, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "access token is required")
}

func TestLoadConfig_WithDefaultURL(t *testing.T) {
	// Store original values to restore later
	originalToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	originalURL := os.Getenv("RUDDERSTACK_API_URL")

	// Set test environment
	os.Setenv("RUDDERSTACK_ACCESS_TOKEN", "test-token")
	os.Unsetenv("RUDDERSTACK_API_URL")

	defer func() {
		if originalToken == "" {
			os.Unsetenv("RUDDERSTACK_ACCESS_TOKEN")
		} else {
			os.Setenv("RUDDERSTACK_ACCESS_TOKEN", originalToken)
		}
		if originalURL == "" {
			os.Unsetenv("RUDDERSTACK_API_URL")
		} else {
			os.Setenv("RUDDERSTACK_API_URL", originalURL)
		}
	}()

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "test-token", cfg.APIToken)
	assert.Equal(t, "https://api.rudderstack.com", cfg.APIURL) // Default URL
}
