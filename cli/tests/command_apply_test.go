package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	globalIgnoreFields = []string{
		"output.createdAt",
		"output.updatedAt",
		"output.id",
		"output.workspaceId",
		"input.categoryId",
		"output.categoryId",
		"output.eventArgs.categoryId",
	}
)

func TestTrackingPlanApply(t *testing.T) {
	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	output, err := executor.Execute(cliBinPath, "tp", "destroy", "--confirm=false")
	require.NoError(t, err, "Failed to destroy resources: %v, output: %s", err, string(output))

	t.Run("should create entities in catalog from project", func(t *testing.T) {
		output, err := executor.Execute(cliBinPath, "tp", "apply", "-l", filepath.Join("testdata", "project", "create"), "--confirm=false")
		require.NoError(t, err, "Initial apply command failed with output: %s", string(output))
		verifyState(t, "create")
	})

	t.Run("should update entities in catalog from project", func(t *testing.T) {
		output, err := executor.Execute(cliBinPath, "tp", "apply", "-l", filepath.Join("testdata", "project", "update"), "--confirm=false")
		require.NoError(t, err, "Update apply command failed with output: %s", string(output))
		verifyState(t, "update")
	})

}

func verifyState(t *testing.T, dir string) {
	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)

	require.NoError(t, err)

	reader := helpers.NewAPIClientAdapter(
		catalog.NewRudderDataCatalog(apiClient),
	)

	expectedStateDir := filepath.Join("testdata", "expected", "state", dir)
	fileManager, err := helpers.NewStateFileManager(expectedStateDir)
	require.NoError(t, err)

	tester := helpers.NewStateSnapshotTester(
		reader,
		fileManager,
		globalIgnoreFields,
	)

	err = tester.SnapshotTest(context.Background())
	assert.NoError(t, err, "State verification failed")
}
