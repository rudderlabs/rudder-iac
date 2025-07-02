package helpers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
)

// UpstreamSnapshotTester provides functionality to test upstream state snapshots
// by fetching actual data from the API and comparing with expected upstream states
type UpstreamSnapshotTester struct {
	dataCatalog catalog.DataCatalog
	fileManager *SnapshotFileManager
	stateReader UpstreamStateReader
	ignore      []string
}

// NewUpstreamSnapshotTester creates a new instance of UpstreamSnapshotTester
// with the required dependencies: DataCatalogClient, FileManager, and UpstreamRawStateFetcher
func NewUpstreamSnapshotTester(
	dataCatalog catalog.DataCatalog,
	stateReader UpstreamStateReader,
	fileManager *SnapshotFileManager,
	ignore []string,
) *UpstreamSnapshotTester {
	return &UpstreamSnapshotTester{
		dataCatalog: dataCatalog,
		fileManager: fileManager,
		stateReader: stateReader,
		ignore:      ignore,
	}
}

// SnapshotTest performs upstream validation by extracting IDs from state and calling catalog APIs
// to verify actual upstream data against expected upstream state files
func (u *UpstreamSnapshotTester) SnapshotTest(ctx context.Context) error {
	actualState, err := u.stateReader.RawState(ctx)
	if err != nil {
		return fmt.Errorf("reading actual state: %w", err)
	}

	actualResources, ok := actualState["resources"].(map[string]any)
	if !ok {
		return fmt.Errorf("actual state resources is not a map: %T", actualState["outputs"])
	}

	expectedResources, err := u.fileManager.ListResources()
	if err != nil {
		return fmt.Errorf("listing upstream files: %w", err)
	}

	if len(actualResources) != len(expectedResources) {
		return fmt.Errorf(
			"resource count mismatch: got %d resources, want %d resources",
			len(actualResources),
			len(expectedResources),
		)
	}

	var errs Errors

	// For each entity ID, call appropriate API method and compare with expected upstream state
	for urn, resourceData := range actualResources {
		resource, ok := resourceData.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Errorf("resource %s: is not a map: %T", urn, resourceData))
			continue
		}

		output, ok := resource["output"].(map[string]any)
		if !ok {
			errs = append(errs, fmt.Errorf("resource %s: output is not a map: %T", urn, resource))
			continue
		}

		entityID, ok := output["id"].(string)
		if !ok {
			errs = append(errs, fmt.Errorf("resource %s: output.id is not a string: %T", urn, output["id"]))
			continue
		}

		actual, err := u.upstreamEntity(ctx, entityID, resource["type"].(string))
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

// fetchUpstreamData calls the appropriate API method based on resource type
func (u *UpstreamSnapshotTester) upstreamEntity(ctx context.Context, entityID, entityType string) (any, error) {
	var v any

	switch entityType {
	case datacatalog.EventResourceType:
		event, err := u.dataCatalog.GetEvent(ctx, entityID)
		if err != nil {
			return nil, fmt.Errorf("calling GetEvent: %w", err)
		}
		v = event

	case datacatalog.PropertyResourceType:
		property, err := u.dataCatalog.GetProperty(ctx, entityID)
		if err != nil {
			return nil, fmt.Errorf("calling GetProperty: %w", err)
		}
		v = property

	case datacatalog.TrackingPlanResourceType:
		trackingPlan, err := u.dataCatalog.GetTrackingPlan(ctx, entityID)
		if err != nil {
			return nil, fmt.Errorf("calling GetTrackingPlan: %w", err)
		}
		v = trackingPlan

	case datacatalog.CustomTypeResourceType:
		customType, err := u.dataCatalog.GetCustomType(ctx, entityID)
		if err != nil {
			return nil, fmt.Errorf("calling GetCustomType: %w", err)
		}
		v = customType

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", entityType)
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
