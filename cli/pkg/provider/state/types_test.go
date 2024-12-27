package state_test

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
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

}
