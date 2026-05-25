package state_test

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/stretchr/testify/assert"
)

func TestMapValues(t *testing.T) {

	address := fmt.Sprintf("%s %s %s",
		"548 Market St. PMB 48141",
		"San Francisco, California",
		"94104-5401",
	)

	defaultmap := map[string]interface{}{
		"name":     "Rudderstack",
		"address":  &address,
		"empCount": 84,
		"hiring":   true,
	}

	t.Run("string values are extracted from map successfully", func(t *testing.T) {
		assert.Equal(t, "Rudderstack", state.MustString(defaultmap, "name"))
		assert.Equal(t, address, *state.MustStringPtr(defaultmap, "address"))
		assert.Equal(t, "Invalid", state.String(defaultmap, "invalid", "Invalid"))
	})

	t.Run("bool values are extracted from map successfully", func(t *testing.T) {
		assert.True(t, state.MustBool(defaultmap, "hiring"))
		assert.False(t, state.Bool(defaultmap, "invalid", false))
	})

	t.Run("numeric values are extracted from map successfully", func(t *testing.T) {
		assert.Equal(t, 84, state.MustInt(defaultmap, "empCount"))
		assert.Equal(t, 0, state.Int(defaultmap, "invalid", 0))

		defaultmap["score"] = 9.5
		assert.InDelta(t, 9.5, state.MustFloat64(defaultmap, "score"), 0)
		assert.InDelta(t, 0.0, state.Float64(defaultmap, "invalid", 0), 0)
	})

	t.Run("nested map and slice values are extracted from map successfully", func(t *testing.T) {
		nested := map[string]interface{}{"region": "us"}
		nestedPtr := &nested
		defaultmap["nested"] = nested
		defaultmap["nestedPtr"] = nestedPtr

		assert.Equal(t, nested, state.MustMapStringInterface(defaultmap, "nested"))
		assert.Equal(t, nestedPtr, state.MustMapStringInterfacePtr(defaultmap, "nestedPtr"))
		assert.Equal(t, map[string]interface{}{"fallback": true}, state.MapStringInterface(defaultmap, "missing", map[string]interface{}{"fallback": true}))

		defaultmap["tags"] = []interface{}{"go", "cli"}
		assert.Equal(t, []string{"go", "cli"}, state.MustStringSlice(defaultmap, "tags"))
	})

}
