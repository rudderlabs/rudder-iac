package destination_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	tc "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Import references a transformation the project already manages the same way
// it references one imported alongside, rather than failing the export.
func TestHandlerImpl_FormatForExport_ManagedTransformation(t *testing.T) {
	t.Parallel()

	store := &testutil.MockTransformationStore{
		ListTransformationsFunc: func(context.Context) ([]*tc.Transformation, error) {
			return []*tc.Transformation{{ID: "trans-remote", ExternalID: "my-transform", Name: "My Transform", Language: "javascript"}}, nil
		},
		ListLibrariesFunc: func(context.Context) ([]*tc.TransformationLibrary, error) {
			return nil, nil
		},
	}
	tp := transformations.NewProviderWithStore(store)
	require.NoError(t, tp.LoadSpec("transformations/my-transform.yaml", &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    "transformation",
		Spec: map[string]any{
			"id":       "my-transform",
			"name":     "My Transform",
			"language": "javascript",
			"code":     "export function transformEvent(event, metadata) { return event; }",
		},
	}))

	// The resolver import workspace builds: remote state and the project graph
	// hold the managed transformation, the import set holds only the destination.
	graph, err := tp.ResourceGraph()
	require.NoError(t, err)
	remote, err := tp.LoadResourcesFromRemote(context.Background())
	require.NoError(t, err)
	importRefResolver := &resolver.ImportRefResolver{
		Remote:     remote,
		Graph:      graph,
		Importable: resources.NewRemoteResources(),
	}

	h := destination.NewHandler(nil, testRegistry(t))
	collection := map[string]*destination.RemoteDestination{
		"webhook-1": {Destination: &client.Destination{
			ID:             "dst-1",
			Name:           "My Webhook",
			Type:           "WEBHOOK",
			Version:        1,
			IsEnabled:      true,
			Config:         []byte(`{"webhookUrl":"https://example.com/hook"}`),
			Transformation: &client.DestinationTransformationLink{ID: "trans-remote"},
		}},
	}

	entities, _, err := h.Impl.FormatForExport(collection, nil, importRefResolver)
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
		"transformation": "#transformation:my-transform",
	}, spec.Spec)
}
