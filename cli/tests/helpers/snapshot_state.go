package helpers

import (
	"context"
	"fmt"
)

// UpstreamStateReader provides an interface for reading state in a raw format.
type UpstreamStateReader interface {
	RawState(ctx context.Context) (map[string]any, error)
}

// SnapshotStateTester provides functionality to test state snapshots
// using the existing generic comparator.
type SnapshotStateTester struct {
	reader      UpstreamStateReader
	fileManager *StateFileManager
	ignore      []string
}

// NewSnapshotStateTester creates a new instance of SnapshotStateTester
func NewSnapshotStateTester(reader UpstreamStateReader, manager *StateFileManager, ignore []string) *SnapshotStateTester {
	return &SnapshotStateTester{
		reader:      reader,
		fileManager: manager,
		ignore:      ignore,
	}
}

// SnapshotTest performs a complete snapshot comparison between actual and expected state.
// It fetches the actual state, compares versions, resource counts, and individual resources.
func (s *SnapshotStateTester) SnapshotTest(ctx context.Context) error {
	actualState, err := s.reader.RawState(ctx)
	if err != nil {
		return fmt.Errorf("reading actual state: %w", err)
	}

	expectedVersion, ok := s.fileManager.LoadExpectedVersion()
	if !ok {
		return fmt.Errorf("no expected version found")
	}

	if expectedVersion != actualState["version"].(string) {
		return fmt.Errorf("version mismatch: got %q, want %q", actualState["version"], expectedVersion)
	}

	actualResources, ok := actualState["resources"].(map[string]any)
	if !ok {
		return fmt.Errorf("actual state resources is not a map: %T", actualState["resources"])
	}

	expectedResources, err := s.fileManager.ListResources()
	if err != nil {
		return fmt.Errorf("listing expected resources: %w", err)
	}

	if len(actualResources) != len(expectedResources) {
		return fmt.Errorf(
			"resource count mismatch: got %d resources, want %d resources",
			len(actualResources),
			len(expectedResources),
		)
	}

	var errs Errors

	for id, resourceData := range actualResources {
		actualResource, ok := resourceData.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Errorf("resource %s: actual resource is not a map: %T", id, resourceData))
			continue
		}

		expectedResource, err := s.fileManager.LoadExpectedState(id)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: failed to load expected state: %v", id, err))
			continue
		}

		if err := CompareStates(actualResource, expectedResource, s.ignore); err != nil {
			errs = append(errs, fmt.Errorf("resource %s: %v", id, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}
