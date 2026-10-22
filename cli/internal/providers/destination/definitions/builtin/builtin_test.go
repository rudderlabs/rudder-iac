package builtin

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryRegistersTheTable(t *testing.T) {
	t.Parallel()

	var verified, all []string
	for _, entry := range entries {
		def := entry.NewDefinition()
		all = append(all, def.Type)
		if entry.Verified {
			verified = append(verified, def.Type)
		}
		assert.Equal(t, entry.Verified, IsVerified(def.Type, def.Version), def.Type)
	}
	slices.Sort(verified)
	slices.Sort(all)

	off, err := NewRegistry(false)
	require.NoError(t, err)
	assert.Equal(t, verified, off.SupportedTypes())

	on, err := NewRegistry(true)
	require.NoError(t, err)
	assert.Equal(t, all, on.SupportedTypes())

	assert.False(t, IsVerified("not_a_type", 1))
}

// Registration describes every config model, so a definition whose tags the
// docs command cannot read fails here, with the flag off and on, rather than
// in front of a user. Every secret key must come out as a secret field.
func TestEveryDefinitionDescribesItsSecrets(t *testing.T) {
	t.Parallel()

	for _, includeUnverified := range []bool{false, true} {
		registry, err := NewRegistry(includeUnverified)
		require.NoError(t, err)

		for _, registered := range registry.Definitions() {
			desc := registered.Describe()
			assert.NotEmpty(t, desc.DisplayName, desc.Type)
			assert.NotEmpty(t, desc.Fields, desc.Type)

			var secretFields []string
			for _, field := range desc.Fields {
				if field.Secret {
					secretFields = append(secretFields, strings.ReplaceAll(field.Key, "[]", ""))
				}
			}
			assert.ElementsMatch(t, registered.SecretKeys(), secretFields, desc.Type)
		}
	}
}
