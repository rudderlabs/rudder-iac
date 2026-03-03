package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestNewEventSemanticValidRule_V1Patterns(t *testing.T) {
	t.Parallel()

	rule := NewEventSemanticValidRule()

	patterns := rule.AppliesTo()
	assert.Contains(t, patterns, rules.MatchKindVersion("events", specs.SpecVersionV1),
		"Rule should include V1 match pattern")
}

func TestEventSemanticV1Valid_CategoryRef(t *testing.T) {
	t.Parallel()

	t.Run("category ref found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("page_events", "category")
		catRef := "#category:page_events"

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{
					LocalID:     "page_viewed",
					Name:        "Page Viewed",
					Type:        "track",
					CategoryRef: &catRef,
				},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "category exists in graph — no errors expected")
	})

	t.Run("category ref not found", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		catRef := "#category:nonexistent"

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{
					LocalID:     "page_viewed",
					Name:        "Page Viewed",
					Type:        "track",
					CategoryRef: &catRef,
				},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0/category", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced category 'nonexistent' not found")
	})

	t.Run("nil category ref skipped", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{
					LocalID: "page_viewed",
					Name:    "Page Viewed",
					Type:    "track",
				},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "nil category ref should be skipped")
	})

	t.Run("multiple events mixed results", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("page_events", "category")

		var (
			validRef   = "#category:page_events"
			invalidRef = "#category:missing_cat"
		)

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
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
					Type:    "identify",
				},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/1/category", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced category 'missing_cat' not found")
	})
}

func TestEventSemanticV1Valid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("no duplicate — unique track events", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed", "Page Viewed", "track"))
		graph.AddResource(eventResource("button_clicked", "Button Clicked", "track"))

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "page_viewed", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "unique track event names should not trigger errors")
	})

	t.Run("duplicate track event detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed_v1", "Page Viewed", "track"))
		graph.AddResource(eventResource("page_viewed_v2", "Page Viewed", "track"))

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "page_viewed_v1", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("single track event in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("page_viewed", "Page Viewed", "track"))

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "page_viewed", Name: "Page Viewed", Type: "track"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)
		assert.Empty(t, results, "single track event should not be flagged")
	})

	t.Run("duplicate non-track event detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("screen_v1", "", "screen"))
		graph.AddResource(eventResource("screen_v2", "", "screen"))

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "screen_v1", Name: "", Type: "screen"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/events/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name")
	})

	t.Run("different non-track types — not duplicates", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(eventResource("screen_ev", "", "screen"))
		graph.AddResource(eventResource("page_ev", "", "page"))
		graph.AddResource(eventResource("group_ev", "", "group"))
		graph.AddResource(eventResource("identify_ev", "", "identify"))

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "screen_ev", Name: "", Type: "screen"},
				{LocalID: "page_ev", Name: "", Type: "page"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)
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

		spec := localcatalog.EventSpecV1{
			Events: []localcatalog.EventV1{
				{LocalID: "page_viewed_v1", Name: "Page Viewed", Type: "track"},
				{LocalID: "screen_v1", Name: "", Type: "screen"},
				{LocalID: "identify_ev", Name: "", Type: "identify"},
			},
		}

		results := validateEventSemanticV1(localcatalog.KindEvents, specs.SpecVersionV1, nil, spec, graph)

		require.Len(t, results, 2)
		assert.Contains(t, results[0].Message, "duplicate name")
		assert.Contains(t, results[1].Message, "duplicate name")
	})
}
