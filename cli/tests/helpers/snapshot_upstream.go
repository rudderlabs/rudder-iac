package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// UpstreamAdapter combines fetching resources and reading remote state.
// Implementations should be able to list all managed resources and fetch individual resources by ID.
type UpstreamAdapter interface {
	// RemoteIDs returns a map of URN -> resource ID for all managed resources
	RemoteIDs(ctx context.Context) (map[string]string, error)
	// FetchResource fetches a resource by type and ID from the remote API
	FetchResource(ctx context.Context, resourceType, resourceID string) (any, error)
}

// UpstreamSnapshotTester provides functionality to test upstream state snapshots
// by fetching actual data from the API and comparing with expected upstream states
type UpstreamSnapshotTester struct {
	adapter     UpstreamAdapter
	fileManager *SnapshotFileManager
	ignore      []string
}

// NewUpstreamSnapshotTester creates a new instance of UpstreamSnapshotTester
func NewUpstreamSnapshotTester(
	adapter UpstreamAdapter,
	fileManager *SnapshotFileManager,
	ignore []string,
) *UpstreamSnapshotTester {
	return &UpstreamSnapshotTester{
		adapter:     adapter,
		fileManager: fileManager,
		ignore:      ignore,
	}
}

// SnapshotTest performs upstream validation by extracting IDs from state and calling APIs
// to verify actual upstream data against expected upstream state files
func (u *UpstreamSnapshotTester) SnapshotTest(ctx context.Context) error {
	remoteIDs, err := u.adapter.RemoteIDs(ctx)
	if err != nil {
		return fmt.Errorf("reading actual state: %w", err)
	}

	expectedResources, err := u.fileManager.ListResources()
	if err != nil {
		return fmt.Errorf("listing upstream files: %w", err)
	}

	if len(remoteIDs) != len(expectedResources) {
		return fmt.Errorf(
			"resource count mismatch: got %d resources, want %d resources",
			len(remoteIDs),
			len(expectedResources),
		)
	}

	var errs Errors

	// For each entity ID, call appropriate API method and compare with expected upstream state
	for urn, resourceID := range remoteIDs {
		parts := strings.Split(urn, ":")
		resourceType := parts[0]

		actual, err := u.fetchAndMarshal(ctx, resourceID, resourceType)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: failed to fetch upstream data: %v", urn, err))
			continue
		}

		expected, err := u.fileManager.LoadExpectedState(urn)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: failed to load expected upstream state: %v", urn, err))
			continue
		}

		if err := CompareStates(actual, expected, u.ignore); err != nil {
			errs = append(errs, fmt.Errorf("resource %s failed comparison with upstream state: %v", urn, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

// fetchAndMarshal fetches the resource and converts it to map[string]any for comparison
func (u *UpstreamSnapshotTester) fetchAndMarshal(ctx context.Context, resourceID, resourceType string) (any, error) {
	v, err := u.adapter.FetchResource(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}

	byt, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling resource: %w", err)
	}

	var toReturn map[string]any
	if err := json.Unmarshal(byt, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling resource: %w", err)
	}

	return toReturn, nil
}
