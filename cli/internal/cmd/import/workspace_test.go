package importcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The import sync guard diffs the local graph against remote state, so import
// has to resolve variables exactly as apply does — otherwise an unsubstituted
// "{{ .VAR }}" reads as drift and blocks the import.
func TestWorkspaceImportAcceptsVarFile(t *testing.T) {
	t.Parallel()

	flag := NewCmdWorkspaceImport().Flags().Lookup("var-file")
	require.NotNil(t, flag, "import workspace must accept --var-file")
	assert.Equal(t, "stringArray", flag.Value.Type())
}
