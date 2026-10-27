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
		if !slices.Contains(all, def.Type) {
			all = append(all, def.Type)
		}
		if entry.Verified && !slices.Contains(verified, def.Type) {
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

func TestHTTPVersionsRegisteredSeparately(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(false)
	require.NoError(t, err)
	v1, err := registry.Get("http", 1)
	require.NoError(t, err)
	v2, err := registry.Get("http", 2)
	require.NoError(t, err)
	assert.NotSame(t, v1, v2)
	assert.Equal(t, "HTTP", v2.APIType)
	byAPI, err := registry.GetByAPIType("HTTP", 2)
	require.NoError(t, err)
	assert.Same(t, v2, byAPI)
	assert.True(t, IsVerified("http", 1))
	assert.True(t, IsVerified("http", 2))
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
