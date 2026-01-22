package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformationsApply(t *testing.T) {
	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	// Clean up any existing transformations before running tests
	output, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "Failed to destroy resources: %v, output: %s", err, string(output))

	t.Run("should create transformations and libraries from project", func(t *testing.T) {
		output, err := executor.Execute(cliBinPath, "apply", "-l", filepath.Join("testdata", "transformations", "create"), "--confirm=false")
		require.NoError(t, err, "Initial apply command failed with output: %s", string(output))
		verifyTransformationsState(t, "create")
	})

	t.Run("should update transformations and libraries from project", func(t *testing.T) {
		time.Sleep(5 * time.Second)
		output, err := executor.Execute(cliBinPath, "apply", "-l", filepath.Join("testdata", "transformations", "update"), "--confirm=false")
		require.NoError(t, err, "Update apply command failed with output: %s", string(output))
		verifyTransformationsState(t, "update")
	})
}

func verifyTransformationsState(t *testing.T, dir string) {
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(apiClient)
	adapter := helpers.NewTransformationAdapter(store)

	expectedStateDir := filepath.Join("testdata", "transformations", "expected", "upstream", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	tester := helpers.NewUpstreamSnapshotTester(
		adapter,
		fileManager,
		[]string{
			"id",
			"versionId",
			"workspaceId",
			"imports",
		},
	)
	err = tester.SnapshotTest(context.Background())
	assert.NoError(t, err, "Upstream state verification failed")
}
