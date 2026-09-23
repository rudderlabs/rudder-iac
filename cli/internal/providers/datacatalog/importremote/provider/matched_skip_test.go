package provider

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Matched resources (import --merge) must produce a manifest entry only: no
// spec content and no spec-embedded import metadata. These tests build the
// importable collections directly, simulating the state after the matcher
// step adopted the local identity.

func matchedLocal(id, resourceType string) *resources.Resource {
	return resources.NewResource(id, resourceType, resources.ResourceData{}, []string{})
}

func specContent(t *testing.T, content any, key string) []map[string]any {
	t.Helper()

	spec, ok := content.(*specs.Spec)
	require.True(t, ok)
	items, ok := spec.Spec[key].([]map[string]any)
	require.True(t, ok)
	return items
}

func collectionOf(resourceType string, rs ...*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	m := make(map[string]*resources.RemoteResource, len(rs))
	for _, r := range rs {
		m[r.ID] = r
	}
	collection.Set(resourceType, m)
	return collection
}

func TestEventFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	p := &EventImportProvider{log: *logger.New("test"), filepath: "data-catalog"}
	collection := collectionOf(types.EventResourceType,
		&resources.RemoteResource{
			ID:          "ev1",
			ExternalID:  "page-viewed",
			Data:        &catalog.Event{ID: "ev1", Name: "Page Viewed", EventType: "track", WorkspaceId: "ws1"},
			MatchedWith: matchedLocal("page-viewed", types.EventResourceType),
		},
		&resources.RemoteResource{
			ID:         "ev2",
			ExternalID: "cart-viewed",
			Data:       &catalog.Event{ID: "ev2", Name: "Cart Viewed", EventType: "track", WorkspaceId: "ws1"},
		},
	)

	entities, entries, err := p.FormatForExport(collection, nil, &mockResolver{})
	require.Nil(t, err)

	require.Equal(t, 1, len(entities))
	assert.Equal(t, 1, len(specContent(t, entities[0].Content, "events")),
		"matched event must not appear in spec content")
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "event:page-viewed", RemoteID: "ev1"},
		{WorkspaceID: "ws1", URN: "event:cart-viewed", RemoteID: "ev2"},
	}, entries)
}

func TestPropertyFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	p := &PropertyImportProvider{log: *logger.New("test"), filepath: "data-catalog"}
	collection := collectionOf(types.PropertyResourceType,
		&resources.RemoteResource{
			ID:          "prop1",
			ExternalID:  "email",
			Data:        &catalog.Property{ID: "prop1", Name: "email", Type: "string", WorkspaceId: "ws1"},
			MatchedWith: matchedLocal("email", types.PropertyResourceType),
		},
		&resources.RemoteResource{
			ID:         "prop2",
			ExternalID: "count",
			Data:       &catalog.Property{ID: "prop2", Name: "count", Type: "integer", WorkspaceId: "ws1"},
		},
	)

	entities, entries, err := p.FormatForExport(collection, nil, &mockResolver{})
	require.Nil(t, err)

	require.Equal(t, 1, len(entities))
	assert.Equal(t, 1, len(specContent(t, entities[0].Content, "properties")),
		"matched property must not appear in spec content")
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "property:email", RemoteID: "prop1"},
		{WorkspaceID: "ws1", URN: "property:count", RemoteID: "prop2"},
	}, entries)
}

func TestCustomTypeFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	p := &CustomTypeImportProvider{log: *logger.New("test"), filepath: "data-catalog"}
	collection := collectionOf(types.CustomTypeResourceType,
		&resources.RemoteResource{
			ID:          "ct1",
			ExternalID:  "email-type",
			Data:        &catalog.CustomType{ID: "ct1", Name: "EmailType", Type: "string", WorkspaceId: "ws1"},
			MatchedWith: matchedLocal("email-type", types.CustomTypeResourceType),
		},
		&resources.RemoteResource{
			ID:         "ct2",
			ExternalID: "phone-type",
			Data:       &catalog.CustomType{ID: "ct2", Name: "PhoneType", Type: "string", WorkspaceId: "ws1"},
		},
	)

	entities, entries, err := p.FormatForExport(collection, nil, &mockResolver{})
	require.Nil(t, err)

	require.Equal(t, 1, len(entities))
	assert.Equal(t, 1, len(specContent(t, entities[0].Content, "types")),
		"matched custom type must not appear in spec content")
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "custom-type:email-type", RemoteID: "ct1"},
		{WorkspaceID: "ws1", URN: "custom-type:phone-type", RemoteID: "ct2"},
	}, entries)
}

func TestTrackingPlanFormatForExportSkipsMatched(t *testing.T) {
	t.Parallel()

	p := &TrackingPlanImportProvider{log: *logger.New("test"), baseImportDir: "data-catalog"}
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
	collection := collectionOf(types.TrackingPlanResourceType,
		&resources.RemoteResource{
			ID:         "tp1",
			ExternalID: "mobile-plan",
			Data: &catalog.TrackingPlanWithIdentifiers{
				TrackingPlan: catalog.TrackingPlan{ID: "tp1", Name: "Mobile Plan", WorkspaceID: "ws1"},
			},
			MatchedWith: matchedLocal("mobile-plan", types.TrackingPlanResourceType),
		},
		&resources.RemoteResource{
			ID:         "tp2",
			ExternalID: "web-plan",
			Data: &catalog.TrackingPlanWithIdentifiers{
				TrackingPlan: catalog.TrackingPlan{ID: "tp2", Name: "Web Plan", WorkspaceID: "ws1"},
			},
		},
	)

	entities, entries, err := p.FormatForExport(collection, idNamer, &mockResolver{})
	require.Nil(t, err)

	// One spec file per unmatched plan; the matched plan writes nothing.
	require.Equal(t, 1, len(entities))
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws1", URN: "tracking-plan:mobile-plan", RemoteID: "tp1"},
		{WorkspaceID: "ws1", URN: "tracking-plan:web-plan", RemoteID: "tp2"},
	}, entries)
}
