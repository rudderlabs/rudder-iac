package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A dry run is refused like an apply and changes nothing, so the test needs no
// opening destroy.
func TestApplyRefusesLinkToTransformationCreatedInSameApply(t *testing.T) {
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute(cliBinPath, "apply", "-l",
		filepath.Join("testdata", "project", "transformation-link-refused"),
		"--dry-run", "--confirm=false")

	require.Error(t, err, "apply must refuse the plan: %s", out)
	assert.Contains(t, string(out), "destination:refused-link-webhook links #transformation:refused-link-tx")
}
