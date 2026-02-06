package funcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildLegacyReferenceRegex tests the legacy regex builder for #/<kind>/<group>/<id> format
func TestBuildLegacyReferenceRegex(t *testing.T) {
	t.Parallel()

	t.Run("matches valid legacy references", func(t *testing.T) {
		regex := BuildLegacyReferenceRegex([]string{"custom-types", "properties"})

		assert.True(t, regex.MatchString("#/custom-types/user-traits/email"))
		assert.True(t, regex.MatchString("#/properties/tracking/page_viewed"))
	})

	t.Run("rejects invalid legacy references", func(t *testing.T) {
		regex := BuildLegacyReferenceRegex([]string{"custom-types"})

		assert.False(t, regex.MatchString("#/custom-types/missing-id"))
		assert.False(t, regex.MatchString("#custom-types:email"))
		assert.False(t, regex.MatchString("not-a-reference"))
	})
}
