package lister

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

// The "Managed as" column is keyed on the data, so a resource type that does
// not carry adoption state keeps the three-column table it had.
func TestHasManaged(t *testing.T) {
	t.Parallel()

	assert.False(t, hasManaged(nil))
	assert.False(t, hasManaged([]resources.ResourceData{{"id": "src-1", "name": "Example"}}))
	assert.True(t, hasManaged([]resources.ResourceData{{"id": "acct-1", "managed": "-"}}))
	assert.True(t, hasManaged([]resources.ResourceData{
		{"id": "src-1", "name": "Example"},
		{"id": "acct-1", "managed": "google-ads-prod"},
	}), "one account carrying the key is enough to add the column")
}
