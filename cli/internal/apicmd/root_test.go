package apicmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The whole point of this binary is to surface the resource verbs first-class
// (un-gated, visible) — unlike rudder-cli where they're experimental/hidden.
func TestRootRegistersVerbsVisible(t *testing.T) {
	want := map[string]bool{
		"get":             false,
		"describe":        false,
		"set-external-id": false,
		"delete":          false,
	}

	for _, c := range rootCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
			assert.Falsef(t, c.Hidden, "verb %q must be visible in rudder-api", c.Name())
		}
	}

	for name, found := range want {
		require.Truef(t, found, "verb %q not registered on rudder-api root", name)
	}
}
