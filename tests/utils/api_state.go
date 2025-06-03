package utils

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

// FetchResourceState creates API and catalog clients and fetches the resource state.
func FetchResourceState(token string) (*catalog.State, error) {
	log.Println("Fetching resource state")
	// Create a new API client
	apiClient, err := NewAPIClient(token)
	if err != nil {
		log.Printf("Error creating API client: %v", err)
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create a new catalog client
	log.Println("Creating catalog client")
	catalogClient := catalog.NewRudderDataCatalog(apiClient)

	// Read the state of resources
	log.Println("Reading resource state from API")
	ctx := context.Background()
	state, err := catalogClient.ReadState(ctx)
	if err != nil {
		log.Printf("Error reading resource state from API: %v", err)
		return nil, fmt.Errorf("failed to read resource state: %w", err)
	}

	log.Printf("Fetched resource state: %+v", state)
	log.Println("Successfully fetched resource state")
	return state, nil
}

// CompareStates logs both states for now. Replace with real comparison logic as needed.
func CompareStates(t *testing.T, before, after *catalog.State) {
	t.Logf("State before: %+v", before)
	t.Logf("State after: %+v", after)
	// TODO: Implement actual comparison logic
}