package retl

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSourceTypeDispatchAgree pins modelsSourceType and decodeConfigFor to the
// same partition of source types.
//
// The two are separate switches, and the risk is one-sided: a type added to
// decodeConfigFor but not to modelsSourceType is a source the client can fully
// represent and the list silently drops — the exact failure this filter exists
// to prevent. Merging them into one switch would mean returning a config
// pointer from a shared constructor, which changes what RETLSource.Config
// holds, so they are kept apart and pinned here instead.
func TestSourceTypeDispatchAgree(t *testing.T) {
	t.Parallel()

	for _, sourceType := range []SourceType{
		ModelSourceType,
		TableSourceType,
		"audience",
		"profiles-table",
		"activation-api",
		"",
	} {
		t.Run(string(sourceType), func(t *testing.T) {
			t.Parallel()

			_, err := decodeConfigFor(sourceType, "postgres", json.RawMessage(`{}`))
			dispatches := err == nil

			assert.Equal(t, dispatches, modelsSourceType(sourceType),
				"modelsSourceType(%q)=%v but decodeConfigFor dispatches=%v: the list filter and the config dispatch disagree, so this type is either dropped despite being representable or kept without a config shape",
				sourceType, modelsSourceType(sourceType), dispatches)
		})
	}
}
