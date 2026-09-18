package project_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sourceIndexSpec = `version: rudder/v1
kind: Source
metadata:
  name: web
spec:
  sources:
    - id: web_source
      name: Web
    - id: mobile_source
      name: Mobile
`

// loadWithURNs runs a real project load over one in-memory spec, with the
// provider reporting the given URN -> JSON Pointer entries, and returns the
// positions the project resolved for them.
func loadWithURNs(t *testing.T, entries []specs.URNEntry) map[string]project.SourceLocation {
	t.Helper()

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockProvider.MatchPatterns = fixtureMatchPatterns
	mockProvider.ParseSpecVal = &specs.ParsedSpec{URNs: entries}

	mockLoader := &MockLoader{
		LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
			return map[string]*specs.RawSpec{
				"sources.yaml": {Data: []byte(sourceIndexSpec)},
			}, nil
		},
	}

	p := project.New(mockProvider, project.WithLoader(mockLoader))
	require.NoError(t, p.Load("."))

	locator, ok := p.(project.SourceLocator)
	require.True(t, ok, "the default project must expose source locations")

	return locator.SourceLocations()
}

func TestSourceLocations_ResolvesExactPointers(t *testing.T) {
	locations := loadWithURNs(t, []specs.URNEntry{
		{URN: "source:web_source", JSONPointerPath: "/spec/sources/0/id"},
		{URN: "source:mobile_source", JSONPointerPath: "/spec/sources/1/id"},
	})

	assert.Equal(t, map[string]project.SourceLocation{
		"source:web_source":    {File: "sources.yaml", Line: 7, Column: 7},
		"source:mobile_source": {File: "sources.yaml", Line: 9, Column: 7},
	}, locations)
}

// A URN whose exact pointer is not in the index must still land the reader on
// the nearest enclosing node rather than being dropped from the graph.
func TestSourceLocations_FallsBackToNearestAncestor(t *testing.T) {
	locations := loadWithURNs(t, []specs.URNEntry{
		{URN: "source:generated", JSONPointerPath: "/spec/sources/0/generated/deeply/nested"},
	})

	assert.Equal(t, map[string]project.SourceLocation{
		"source:generated": {File: "sources.yaml", Line: 7, Column: 7},
	}, locations)
}

func TestSourceLocations_EmptyBeforeLoad(t *testing.T) {
	p := project.New(testutils.NewMockProvider(nil, nil))

	locator, ok := p.(project.SourceLocator)
	require.True(t, ok)
	assert.Empty(t, locator.SourceLocations())
}
