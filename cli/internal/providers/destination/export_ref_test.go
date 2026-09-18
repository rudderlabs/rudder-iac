package destination_test

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Import references a transformation the project already manages, and the
// reference it writes loads back.
//
// The export half alone would duplicate "resolves transformation reference" in
// handler_test.go, which pins the same key through a stub resolver. What this
// adds is the round trip: the exported "#transformation:my-transform" is fed
// back through ExtractResourcesFromSpec, so the string is proven loadable
// rather than merely proven equal to a literal. A reference that exports and
// does not load is the failure mode worth a test this wide, and nothing else
// covers it — the retl equivalent round-trips its account ref the same way.
func TestHandlerImpl_FormatForExport_ManagedTransformation(t *testing.T) {
	t.Parallel()

	const (
		localID  = "my-transform"
		remoteID = "trans-remote"
	)

	h := destination.NewHandler(nil, testRegistry(t))

	entities, _, err := h.Impl.FormatForExport(
		map[string]*destination.RemoteDestination{
			"webhook-1": {Destination: &client.Destination{
				ID:             "dst-1",
				Name:           "My Webhook",
				Type:           "WEBHOOK",
				Version:        1,
				IsEnabled:      true,
				Config:         []byte(`{"webhookUrl":"https://example.com/hook"}`),
				Transformation: &client.DestinationTransformationLink{ID: remoteID},
			}},
		},
		nil,
		managedTransformationResolver(t, remoteID, localID),
	)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	assert.Equal(t, map[string]any{
		"id":                 "webhook-1",
		"display_name":       "My Webhook",
		"type":               "WEBHOOK",
		"enabled":            true,
		"definition_version": int64(1),
		"config": map[string]any{
			"webhook_url": "https://example.com/hook",
		},
		"transformation": "#transformation:" + localID,
	}, spec.Spec)

	// The round trip. ExtractResourcesFromSpec is the load path that calls
	// parseTransformationRef, so this fails if export ever writes a form the
	// loader rejects — the one gap the assertion above cannot see.
	var loaded destination.DestinationSpec
	require.NoError(t, mapstructure.Decode(spec.Spec, &loaded))

	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/webhook-1.yaml", &loaded)
	require.NoError(t, err)
	require.Len(t, extracted, 1)

	// Field asserts rather than the whole-struct compare CLAUDE.md asks for:
	// PropertyRef carries a Resolve func, so two refs to the same resource are
	// never Equal. URN and Property are what identify the referent.
	ref := extracted["webhook-1"].Transformation
	require.NotNil(t, ref)
	assert.Equal(t, resources.URN(localID, ttypes.TransformationResourceType), ref.URN)
	assert.Equal(t, "id", ref.Property)
}

// managedTransformationResolver is the resolver `import workspace` builds when a
// transformation is already managed by the project: remote state and the project
// graph both hold it, and the importable set is empty because the destination is
// the only thing being imported.
func managedTransformationResolver(t *testing.T, remoteID, localID string) *resolver.ImportRefResolver {
	t.Helper()

	tp := transformations.NewProviderWithStore(nil)
	require.NoError(t, tp.LoadSpec("transformations/"+localID+".yaml", &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    ttypes.TransformationSpecKind,
		Spec: map[string]any{
			"id":       localID,
			"name":     "My Transform",
			"language": "javascript",
			"code":     "export function transformEvent(event, metadata) { return event; }",
		},
	}))
	graph, err := tp.ResourceGraph()
	require.NoError(t, err)

	remote := resources.NewRemoteResources()
	remote.Set(ttypes.TransformationResourceType, map[string]*resources.RemoteResource{
		remoteID: {ID: remoteID, ExternalID: localID},
	})

	return &resolver.ImportRefResolver{
		Remote:     remote,
		Graph:      graph,
		Importable: resources.NewRemoteResources(),
	}
}
