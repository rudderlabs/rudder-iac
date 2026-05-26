package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/require"
)

func TestCategorySpecRule_ImplementsDocumented(t *testing.T) {
	rule := NewCategorySpecSyntaxValidRule()
	d, ok := rule.(docs.Documented)
	require.True(t, ok, "rule must implement docs.Documented")
	entries := d.DocExamples()
	require.GreaterOrEqual(t, len(entries), 2, "must cover both v0.1 and v1")
}
