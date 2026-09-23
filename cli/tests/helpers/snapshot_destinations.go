package helpers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
)

// DestinationLister is the subset of the destinations API client the tester needs.
type DestinationLister interface {
	GetAll(ctx context.Context) ([]client.Destination, error)
}

// DestinationSnapshotTester compares the CLI-managed destinations fetched from the
// API against expected snapshot files, mirroring UpstreamSnapshotTester. Unlike
// the catalog client, the destinations API exposes no hasExternalId list filter,
// so managed destinations are selected client-side by ExternalID presence.
type DestinationSnapshotTester struct {
	client      DestinationLister
	fileManager *SnapshotFileManager
	ignore      []string
}

func NewDestinationSnapshotTester(
	c DestinationLister,
	fileManager *SnapshotFileManager,
	ignore []string,
) *DestinationSnapshotTester {
	return &DestinationSnapshotTester{
		client:      c,
		fileManager: fileManager,
		ignore:      ignore,
	}
}

// SnapshotTest fetches every managed destination and compares each against its
// expected snapshot file, keyed by URN (destination:<externalID>). It guards the
// managed count against the number of expected files so an unexpected create or
// delete upstream fails the test, matching UpstreamSnapshotTester.
func (d *DestinationSnapshotTester) SnapshotTest(ctx context.Context) error {
	all, err := d.client.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("listing destinations: %w", err)
	}

	// The destinations API has no hasExternalId filter, so drop UI/other-tool
	// resources here — only CLI-managed destinations carry an external ID.
	managed := make(map[string]client.Destination)
	for _, dest := range all {
		if dest.ExternalID != "" {
			managed[urn(destination.DestinationResourceType, dest.ExternalID)] = dest
		}
	}

	expectedResources, err := d.fileManager.ListResources()
	if err != nil {
		return fmt.Errorf("listing upstream files: %w", err)
	}

	if len(managed) != len(expectedResources) {
		return fmt.Errorf(
			"resource count mismatch: got %d managed destinations, want %d resources",
			len(managed),
			len(expectedResources),
		)
	}

	var errs Errors
	for resourceURN, dest := range managed {
		actual, err := toMap(dest)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: %w", resourceURN, err))
			continue
		}

		expected, err := d.fileManager.LoadExpectedState(resourceURN)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: loading expected state: %w", resourceURN, err))
			continue
		}

		if err := CompareStates(actual, expected, d.ignore); err != nil {
			errs = append(errs, fmt.Errorf("resource %s: %w", resourceURN, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}
