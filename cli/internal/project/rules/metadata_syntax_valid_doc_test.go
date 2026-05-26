package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataSyntaxValidRule_ImplementsDocumented(t *testing.T) {
	rule := NewMetadataSyntaxValidRule(nil, []string{"rudder/0.1", "rudder/v0.1", "rudder/v1"})
	d, ok := rule.(docs.Documented)
	require.True(t, ok)

	entries := d.DocExamples()
	require.Len(t, entries, 1)
	assert.Equal(t, "*", entries[0].AppliesTo[0].Kind)
	require.NotEmpty(t, entries[0].Invalid)
}
