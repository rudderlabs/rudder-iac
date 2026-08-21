package datacatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// refsFor runs the lister registered for resourceType against r.
func refsFor(t *testing.T, p *Provider, resourceType string, r *resources.RemoteResource) []importmatcher.Ref {
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

	t.Run("registers listers for the referencing resource types", func(t *testing.T) {
		t.Parallel()

		got := make(map[string]bool)
		for _, l := range p.ImportableRefs() {
			got[l.ResourceType] = true
		}
		assert.True(t, got[types.EventResourceType], "event lister")
		assert.True(t, got[types.PropertyResourceType], "property lister")
		assert.True(t, got[types.CustomTypeResourceType], "custom-type lister")
		assert.True(t, got[types.TrackingPlanResourceType], "tracking-plan lister")
	})

	t.Run("event references its category", func(t *testing.T) {
		t.Parallel()

		catID := "cat_1"
		r := &resources.RemoteResource{Data: &catalog.Event{CategoryId: &catID}}

		refs := refsFor(t, p, types.EventResourceType, r)

		assert.Equal(t, []importmatcher.Ref{{EntityType: types.CategoryResourceType, RemoteID: "cat_1"}}, refs)
	})

	t.Run("event without a category references nothing", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &catalog.Event{CategoryId: nil}}

		assert.Empty(t, refsFor(t, p, types.EventResourceType, r))
	})

	t.Run("property references its custom type definitions", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &catalog.Property{
			DefinitionId:     "ct_1",
			ItemDefinitionId: "ct_2",
		}}

		refs := refsFor(t, p, types.PropertyResourceType, r)

		assert.ElementsMatch(t, []importmatcher.Ref{
			{EntityType: types.CustomTypeResourceType, RemoteID: "ct_1"},
			{EntityType: types.CustomTypeResourceType, RemoteID: "ct_2"},
		}, refs)
	})

	t.Run("property without definitions references nothing", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &catalog.Property{}}

		assert.Empty(t, refsFor(t, p, types.PropertyResourceType, r))
	})

	t.Run("custom type references its properties and nested custom types", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &catalog.CustomType{
			Properties: []catalog.CustomTypeProperty{{ID: "prop_1"}, {ID: "prop_2"}},
			ItemDefinitions: []any{
				map[string]any{"id": "ct_nested"},
			},
		}}

		refs := refsFor(t, p, types.CustomTypeResourceType, r)

		assert.ElementsMatch(t, []importmatcher.Ref{
			{EntityType: types.PropertyResourceType, RemoteID: "prop_1"},
			{EntityType: types.PropertyResourceType, RemoteID: "prop_2"},
			{EntityType: types.CustomTypeResourceType, RemoteID: "ct_nested"},
		}, refs)
	})

	t.Run("tracking plan references its events and their properties", func(t *testing.T) {
		t.Parallel()

		r := &resources.RemoteResource{Data: &catalog.TrackingPlanWithIdentifiers{
			Events: []*catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID: "evt_1",
					Properties: []*catalog.TrackingPlanEventProperty{
						{ID: "prop_1"},
					},
				},
				{ID: "evt_2"},
			},
		}}

		refs := refsFor(t, p, types.TrackingPlanResourceType, r)

		assert.ElementsMatch(t, []importmatcher.Ref{
			{EntityType: types.EventResourceType, RemoteID: "evt_1"},
			{EntityType: types.EventResourceType, RemoteID: "evt_2"},
			{EntityType: types.PropertyResourceType, RemoteID: "prop_1"},
		}, refs)
	})
}
