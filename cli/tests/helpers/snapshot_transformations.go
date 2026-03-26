package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// VersionedResourceRef holds the IDs needed to fetch a specific version of a resource.
type VersionedResourceRef struct {
	ResourceID string
	VersionID  string
}

// TransformationSnapshotTester compares transformation and library versions
// fetched from the API against expected snapshot files using CompareStates.
type TransformationSnapshotTester struct {
	store       transformations.TransformationStore
	resources   map[string]VersionedResourceRef
	fileManager *SnapshotFileManager
	ignore      []string
}

func NewTransformationSnapshotTester(
	store transformations.TransformationStore,
	resources map[string]VersionedResourceRef,
	fileManager *SnapshotFileManager,
	ignore []string,
) *TransformationSnapshotTester {
	return &TransformationSnapshotTester{
		store:       store,
		resources:   resources,
		fileManager: fileManager,
		ignore:      ignore,
	}
}

// SnapshotTest fetches each versioned resource and compares it against
// expected snapshot file. Returns an aggregated error if any mismatches are found.
func (t *TransformationSnapshotTester) SnapshotTest(ctx context.Context) error {
	var errs Errors

	for urn, ref := range t.resources {
		parts := strings.Split(urn, ":")
		resourceType := parts[0]

		actual, err := t.upstreamVersion(ctx, ref, resourceType)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: fetching version: %w", urn, err))
			continue
		}

		expected, err := t.fileManager.LoadExpectedState(urn)
		if err != nil {
			errs = append(errs, fmt.Errorf("resource %s: loading expected state: %w", urn, err))
			continue
		}

		if err := CompareStates(actual, expected, t.ignore); err != nil {
			errs = append(errs, fmt.Errorf("resource %s: %w", urn, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (t *TransformationSnapshotTester) upstreamVersion(ctx context.Context, ref VersionedResourceRef, resourceType string) (map[string]any, error) {
	var v any

	switch resourceType {
	case transformation.HandlerMetadata.ResourceType:
		transformation, err := t.store.GetTransformationVersion(ctx, ref.ResourceID, ref.VersionID)
		if err != nil {
			return nil, fmt.Errorf("getting transformation version: %w", err)
		}
		v = transformation

	case library.HandlerMetadata.ResourceType:
		library, err := t.store.GetLibraryVersion(ctx, ref.ResourceID, ref.VersionID)
		if err != nil {
			return nil, fmt.Errorf("getting library version: %w", err)
		}
		v = library

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return marshalToMap(v)
}

func marshalToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling resource: %w", err)
	}

	var toReturn map[string]any
	if err := json.Unmarshal(data, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling resource: %w", err)
	}

	return toReturn, nil
}

func VersionRefs(results *testorchestrator.TestResults) map[string]VersionedResourceRef {
	refs := make(map[string]VersionedResourceRef, len(results.Transformations)+len(results.Libraries))

	for _, tr := range results.Transformations {
		refs[resources.URN(tr.Result.ExternalID, transformation.HandlerMetadata.ResourceType)] = VersionedResourceRef{
			ResourceID: tr.Result.ID,
			VersionID:  tr.Result.VersionID,
		}
	}

	for _, lib := range results.Libraries {
		refs[resources.URN(lib.ExternalID, library.HandlerMetadata.ResourceType)] = VersionedResourceRef{
			ResourceID: lib.ID,
			VersionID:  lib.VersionID,
		}
	}
	return refs
}
