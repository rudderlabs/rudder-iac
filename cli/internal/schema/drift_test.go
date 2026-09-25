package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hypotheticalSpec struct {
	ID       string `json:"id" validate:"required"`
	NewField string `json:"new_field" validate:"required,oneof=alpha beta"`
}

func TestSpecStructChangesFlowIntoSchema(t *testing.T) {
	s := specBlock(hypotheticalSpec{})
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(raw, &document))
	properties, ok := document["properties"].(map[string]any)
	require.True(t, ok)
	field, ok := properties["new_field"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"alpha", "beta"}, field["enum"])
	assert.ElementsMatch(t, []any{"id", "new_field"}, document["required"])
}
