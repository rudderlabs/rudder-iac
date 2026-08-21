package datagraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	dghandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

func dgRefsFor(t *testing.T, p *Provider, resourceType string, r *resources.RemoteResource) []importmatcher.Ref {
	t.Helper()
	for _, l := range p.ImportableRefs() {
		if l.ResourceType == resourceType {
			return l.Refs(r)
		}
	}
	require.Failf(t, "no lister", "no ImportableRefs lister for %q", resourceType)
	return nil
}

func TestImportableRefs(t *testing.T) {
	t.Parallel()
	p := &Provider{}

	t.Run("registers listers for model and relationship", func(t *testing.T) {
		t.Parallel()

		got := make(map[string]bool)
		for _, l := range p.ImportableRefs() {
			got[l.ResourceType] = true
		}
		assert.True(t, got[modelhandler.HandlerMetadata.ResourceType], "model lister")
		assert.True(t, got[relhandler.HandlerMetadata.ResourceType], "relationship lister")
	})

	t.Run("model references its data graph", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &dgModel.RemoteModel{
			Model: &dgClient.Model{DataGraphID: "dg_1"},
		}}

		refs := dgRefsFor(t, p, modelhandler.HandlerMetadata.ResourceType, r)

		assert.Equal(t, []importmatcher.Ref{
			{EntityType: dghandler.HandlerMetadata.ResourceType, RemoteID: "dg_1"},
		}, refs)
	})

	t.Run("relationship references its data graph and both models", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &dgModel.RemoteRelationship{
			Relationship: &dgClient.Relationship{
				DataGraphID:   "dg_1",
				SourceModelID: "model_a",
				TargetModelID: "model_b",
			},
		}}

		refs := dgRefsFor(t, p, relhandler.HandlerMetadata.ResourceType, r)

		assert.ElementsMatch(t, []importmatcher.Ref{
			{EntityType: dghandler.HandlerMetadata.ResourceType, RemoteID: "dg_1"},
			{EntityType: modelhandler.HandlerMetadata.ResourceType, RemoteID: "model_a"},
			{EntityType: modelhandler.HandlerMetadata.ResourceType, RemoteID: "model_b"},
		}, refs)
	})

	t.Run("empty reference ids are skipped", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &dgModel.RemoteRelationship{
			Relationship: &dgClient.Relationship{SourceModelID: "model_a"},
		}}

		refs := dgRefsFor(t, p, relhandler.HandlerMetadata.ResourceType, r)

		assert.Equal(t, []importmatcher.Ref{
			{EntityType: modelhandler.HandlerMetadata.ResourceType, RemoteID: "model_a"},
		}, refs)
	})
}
