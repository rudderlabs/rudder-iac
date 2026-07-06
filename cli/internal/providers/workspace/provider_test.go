package workspace

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MapRemoteToState must return an empty state regardless of input: accounts are
// read-only and must never enter the apply/destroy source graph, or apply would
// try to delete them.
func TestProvider_MapRemoteToState_AlwaysEmpty(t *testing.T) {
	p := &Provider{}

	coll := resources.NewRemoteResources()
	coll.Set(AccountResourceType, map[string]*resources.RemoteResource{
		"acc_1": {ID: "acc_1", Data: map[string]any{"name": "Prod DB"}},
	})

	st, err := p.MapRemoteToState(coll)
	require.NoError(t, err)
	assert.Empty(t, st.Resources)
}

// FormatForExport surfaces each account's fields as the renderable content used
// by `get -o yaml/json` and `describe`.
func TestProvider_FormatForExport(t *testing.T) {
	p := &Provider{}

	coll := resources.NewRemoteResources()
	coll.Set(AccountResourceType, map[string]*resources.RemoteResource{
		"acc_1": {ID: "acc_1", Data: map[string]any{"name": "Prod DB", "id": "acc_1"}},
	})

	entities, err := p.FormatForExport(coll, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, "acc_1.yaml", entities[0].RelativePath)
	assert.Equal(t, map[string]any{"name": "Prod DB", "id": "acc_1"}, entities[0].Content)
}
