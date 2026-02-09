package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestEventSemanticValid_CategoryRefFound(t *testing.T) {
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

	results := funcs.ValidateReferences(spec, graph)
	assert.Empty(t, results, "category exists in graph — no errors expected")
}

func TestEventSemanticValid_CategoryRefNotFound(t *testing.T) {
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

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/events/0/category", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced category 'nonexistent' not found")
}

func TestEventSemanticValid_NilCategoryRef(t *testing.T) {
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

	results := funcs.ValidateReferences(spec, graph)
	assert.Empty(t, results, "nil category ref should be skipped")
}

func TestEventSemanticValid_MultipleEvents_MixedResults(t *testing.T) {
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
				Type:    "identify",
			},
		},
	}

	results := funcs.ValidateReferences(spec, graph)

	assert.Len(t, results, 1)
	assert.Equal(t, "/events/1/category", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced category 'missing_cat' not found")
}
