package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hypotheticalSpec stands in for a spec struct gaining a new field. The schema
// is derived purely by reflection over the struct, so a field added here must
// appear in the emitted schema with no hand-authored JSON — proving the
// generator is the single source of truth and cannot drift.
type hypotheticalSpec struct {
	ID       string `json:"id" validate:"required"`
	NewField string `json:"new_field" validate:"required,oneof=alpha beta"`
}

func TestSpecStructChangesFlowIntoSchema(t *testing.T) {
	s := specBlock(hypotheticalSpec{})

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	props, ok := doc["properties"].(map[string]any)
	require.True(t, ok)

	// The newly added field is present without touching any schema file.
	field, ok := props["new_field"].(map[string]any)
	require.True(t, ok, "new struct field must surface in the schema")
	assert.Equal(t, []any{"alpha", "beta"}, field["enum"], "validate:oneof must become an enum")

	required, _ := doc["required"].([]any)
	assert.Subset(t, required, []any{"id", "new_field"}, "validate:required must become required")
}
