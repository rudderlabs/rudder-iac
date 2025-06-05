package config

import (
	"fmt"
	"os"
)

// Config holds the configuration for the schema operations
type Config struct {
	APIToken string
	APIURL   string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	apiToken := os.Getenv("RUDDERSTACK_ACCESS_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("access token is required. Please run 'rudder-cli auth login' or set RUDDERSTACK_ACCESS_TOKEN environment variable")
	}

	apiURL := os.Getenv("RUDDERSTACK_API_URL")
	if apiURL == "" {
		// Use default URL if not provided
		apiURL = "https://api.rudderstack.com"
	}

	return &Config{
		APIToken: apiToken,
		APIURL:   apiURL,
	}, nil
}
