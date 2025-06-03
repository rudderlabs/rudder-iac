package utils

import (
	"fmt"
	"log"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// NewAPIClient creates and returns a new API client.
func NewAPIClient(token string) (*client.Client, error) {
	log.Println("Creating API client")
	apiClient, err := client.New(token, client.WithBaseURL(client.BASE_URL_V2))
	if err != nil {
		log.Printf("Error creating API client: %v", err)
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	log.Println("Successfully created API client")
	return apiClient, nil
} 