package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLister serves canned rows per resource type and records what was asked
// for, so a fan-out can be checked without a provider or a network.
type fakeLister struct {
	rows  map[string][]resources.ResourceData
	err   error
	asked []string
}

func (f *fakeLister) List(_ context.Context, resourceType string, _ lister.Filters) ([]resources.ResourceData, error) {
	f.asked = append(f.asked, resourceType)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows[resourceType], nil
}

func TestAllKindsList(t *testing.T) {
	t.Parallel()

	t.Run("tags every row with the kind it came from", func(t *testing.T) {
		t.Parallel()

		f := &fakeLister{rows: map[string][]resources.ResourceData{
			sqlmodel.ResourceType: {{"id": "src-1", "name": "Users"}},
			table.ResourceType:    {{"id": "src-2", "name": "Orders"}},
		}}

		got, err := allKinds{provider: f, kinds: []string{sqlmodel.ResourceType, table.ResourceType}}.
			List(context.Background(), "retl-sources", nil)

		require.NoError(t, err)
		assert.Equal(t, []resources.ResourceData{
			{"id": "src-1", "name": "Users", "kind": sqlmodel.ResourceType},
			{"id": "src-2", "name": "Orders", "kind": table.ResourceType},
		}, got)
		assert.Equal(t, []string{sqlmodel.ResourceType, table.ResourceType}, f.asked,
			"kinds are listed in the declared order, so the table is deterministic")
	})

	// The whole point of the fan-out: before it, a table source existed in the
	// workspace and never appeared in the list.
	t.Run("a kind whose handler is off is never asked for", func(t *testing.T) {
		t.Parallel()

		f := &fakeLister{rows: map[string][]resources.ResourceData{
			sqlmodel.ResourceType: {{"id": "src-1"}},
			table.ResourceType:    {{"id": "src-2"}},
		}}

		got, err := allKinds{provider: f, kinds: []string{sqlmodel.ResourceType}}.
			List(context.Background(), "retl-sources", nil)

		require.NoError(t, err)
		assert.Equal(t, []resources.ResourceData{{"id": "src-1", "kind": sqlmodel.ResourceType}}, got)
		assert.Equal(t, []string{sqlmodel.ResourceType}, f.asked)
	})

	t.Run("an error on any kind fails the whole list", func(t *testing.T) {
		t.Parallel()

		f := &fakeLister{err: errors.New("listing RETL sources: boom")}

		_, err := allKinds{provider: f, kinds: []string{sqlmodel.ResourceType, table.ResourceType}}.
			List(context.Background(), "retl-sources", nil)

		assert.EqualError(t, err, "listing RETL sources: boom")
	})

	t.Run("no registered kinds lists nothing rather than erroring", func(t *testing.T) {
		t.Parallel()

		f := &fakeLister{}

		got, err := allKinds{provider: f, kinds: nil}.List(context.Background(), "retl-sources", nil)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Empty(t, f.asked)
	})
}

func TestSourceKindsOrder(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{sqlmodel.ResourceType, table.ResourceType}, sourceKinds)
}
