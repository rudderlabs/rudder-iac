package testutil

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// AssertModelsSchemaKeys checks that properties model exactly the top-level
// API config keys upstream's schema.json declares, less the ones the
// definition leaves out on purpose, plus the ones it models beyond the schema
// (each with its reason). Destination update replaces the whole
// config, so a schema key left unmodelled is erased from the remote on the
// first apply or adopt (DEX-525); a new schema key fails this until it is
// modelled or excluded with a reason.
func AssertModelsSchemaKeys(t *testing.T, properties []converter.ConfigProperty, schemaKeys []string, excluded, extra map[string]string) {
	t.Helper()

	var modelled []string
	for _, prop := range properties {
		root, _, _ := strings.Cut(prop.APIKey, ".")
		if root != "" && !slices.Contains(modelled, root) {
			modelled = append(modelled, root)
		}
	}
	slices.Sort(modelled)

	var expected []string
	for _, key := range schemaKeys {
		if _, skip := excluded[key]; !skip {
			expected = append(expected, key)
		}
	}
	for key := range extra {
		expected = append(expected, key)
	}
	slices.Sort(expected)

	assert.Equal(t, expected, modelled)
}
