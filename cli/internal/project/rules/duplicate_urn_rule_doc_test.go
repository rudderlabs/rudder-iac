package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicateURNRule_ImplementsDocumented(t *testing.T) {
	rule := NewDuplicateURNRule(nil)
	d, ok := rule.(docs.Documented)
	require.True(t, ok)

	entries := d.DocExamples()
	require.Len(t, entries, 1)

	require.Len(t, entries[0].Invalid, 1)
	ex := entries[0].Invalid[0]
	assert.Len(t, ex.Files, 2, "duplicate URN example must span 2 files")
}
