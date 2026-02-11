package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// eventResource creates an event resource with name and eventType in its data,
// matching the shape produced by EventArgs.ToResourceData().
func eventResource(id, name, eventType string) *resources.Resource {
	data := resources.ResourceData{
		"name":      name,
		"eventType": eventType,
	}
	return resources.NewResource(id, "event", data, nil)
}

func TestEventSemanticValid_CategoryRef(t *testing.T) {
	t.Parallel()

	t.Run("category ref found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("page_events", "category")
		catRef := "#category:page_events"

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{
					LocalID:     "page_viewed",
					Name:        "Page Viewed",
					Type:        "track",
					CategoryRef: &catRef,
				},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "category exists in graph — no errors expected")
	})

	t.Run("category ref not found", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		catRef := "#category:nonexistent"

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{
					LocalID:     "page_viewed",
					Name:        "Page Viewed",
					Type:        "track",
					CategoryRef: &catRef,
				},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0/category", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced category 'nonexistent' not found")
	})

	t.Run("nil category ref skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{
					LocalID: "page_viewed",
					Name:    "Page Viewed",
					Type:    "track",
				},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "nil category ref should be skipped")
	})

	t.Run("multiple events mixed results", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("page_events", "category")

		var (
			validRef   = "#category:page_events"
			invalidRef = "#category:missing_cat"
		)

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{
					LocalID:     "page_viewed",
					Name:        "Page Viewed",
					Type:        "track",
					CategoryRef: &validRef,
				},
				{
					LocalID:     "button_clicked",
					Name:        "Button Clicked",
					Type:        "track",
					CategoryRef: &invalidRef,
				},
				{
					LocalID: "user_identified",
					Name:    "User Identified",
					Type:    "identify",
				},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/1/category", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced category 'missing_cat' not found")
	})
}

func TestEventSemanticValid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("no duplicate — unique track events", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed", "Page Viewed", "track"))
		graph.AddResource(eventResource("button_clicked", "Button Clicked", "track"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "page_viewed", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "unique track event names should not trigger errors")
	})

	t.Run("duplicate track event detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed_v1", "Page Viewed", "track"))
		graph.AddResource(eventResource("page_viewed_v2", "Page Viewed", "track"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "page_viewed_v1", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
	})

	t.Run("single track event in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed", "Page Viewed", "track"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "page_viewed", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "single track event should not be flagged")
	})

	t.Run("duplicate non-track event detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("screen_v1", "", "screen"))
		graph.AddResource(eventResource("screen_v2", "", "screen"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "screen_v1", Name: "", Type: "screen"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
	})

	t.Run("different non-track types — not duplicates", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("screen_ev", "", "screen"))
		graph.AddResource(eventResource("page_ev", "", "page"))
		graph.AddResource(eventResource("group_ev", "", "group"))
		graph.AddResource(eventResource("identify_ev", "", "identify"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "screen_ev", Name: "", Type: "screen"},
				{LocalID: "page_ev", Name: "", Type: "page"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "different non-track event types are not duplicates")
	})

	t.Run("mixed track and non-track duplicates", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed_v1", "Page Viewed", "track"))
		graph.AddResource(eventResource("page_viewed_v2", "Page Viewed", "track"))
		graph.AddResource(eventResource("screen_v1", "", "screen"))
		graph.AddResource(eventResource("screen_v2", "", "screen"))
		graph.AddResource(eventResource("identify_ev", "", "identify"))

		spec := localcatalog.EventSpec{
			Events: []localcatalog.Event{
				{LocalID: "page_viewed_v1", Name: "Page Viewed", Type: "track"},
				{LocalID: "screen_v1", Name: "", Type: "screen"},
				{LocalID: "identify_ev", Name: "", Type: "identify"},
			},
		}

		results := validateEventSemantic(localcatalog.KindEvents, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 2)
		assert.Contains(t, results[0].Message, "not unique across the catalog")
		assert.Contains(t, results[1].Message, "not unique across the catalog")
	})
}
