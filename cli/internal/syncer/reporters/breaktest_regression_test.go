package reporters

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainReporterDoesNotSilentlyAcceptSkippedApply(t *testing.T) {
	confirmed, err := (&PlainSyncReporter{}).AskConfirmation()
	require.Error(t, err)
	assert.False(t, confirmed)
	assert.ErrorContains(t, err, "--confirm=false")
}

func TestImportPlanWarnsThatAdoptionMayMutateRemoteConfig(t *testing.T) {
	output := renderDiff(&differ.Diff{ImportableResources: []string{"destination:existing"}})
	assert.True(t, strings.Contains(output, "may update") || strings.Contains(output, "will update"),
		"an import plan must disclose that adoption can overwrite remote configuration: %s", output)
}
