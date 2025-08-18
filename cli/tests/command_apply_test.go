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

func TestProjectApply(t *testing.T) {
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
	dataCatalog := catalog.NewRudderDataCatalog(apiClient)
	reader := helpers.NewAPIClientAdapter(dataCatalog)

	// Verify state snapshot
	expectedStateDir := filepath.Join("testdata", "expected", "state", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	tester := helpers.NewStateSnapshotTester(
		reader,
		fileManager,
		[]string{
			"output.id",
			"output.createdAt",
			"output.updatedAt",
			"output.createdBy",
			"output.updatedBy",
			"output.workspaceId",
			"output.categoryId",
			"input.categoryId",
			"output.eventArgs.categoryId",
			"output.version",
			"output.events[0].eventId",
			"output.events[0].id",
			"output.events[1].eventId",
			"output.events[1].id",
			"output.events[2].eventId",
			"output.events[2].id",
			"output.customTypeArgs.properties[0].id",
			"output.customTypeArgs.properties[0].refToId",
			"output.customTypeArgs.properties[1].id",
			"output.customTypeArgs.properties[1].refToId",
			"output.trackingPlanArgs.events[0].categoryId",
			"output.trackingPlanArgs.events[1].categoryId",
			"output.trackingPlanArgs.events[2].categoryId",
			"output.trackingPlanArgs.events[0].id",
			"output.trackingPlanArgs.events[0].properties[0].id",
			"output.trackingPlanArgs.events[0].properties[1].id",
			"output.trackingPlanArgs.events[0].properties[2].id",
			"output.trackingPlanArgs.events[0].properties[3].id",
			"output.trackingPlanArgs.events[1].id",
			"output.trackingPlanArgs.events[1].properties[0].id",
			"output.trackingPlanArgs.events[1].properties[1].id",
			"output.trackingPlanArgs.events[1].properties[2].id",
			"output.trackingPlanArgs.events[1].properties[2].properties[0].id",
			"output.trackingPlanArgs.events[1].properties[2].properties[1].id",
			"output.trackingPlanArgs.events[1].properties[2].properties[1].properties[0].id",
			"output.trackingPlanArgs.events[1].properties[2].properties[1].properties[1].id",
			"output.trackingPlanArgs.events[1].properties[2].properties[1].properties[1].properties[0].id",
			"output.trackingPlanArgs.events[2].properties[2].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[0].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[1].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[1].properties[0].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[1].properties[1].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[1].properties[1].properties[0].id",
			"output.trackingPlanArgs.events[2].properties[2].properties[1].properties[1].properties[1].id",
			"output.trackingPlanArgs.events[1].properties[3].id",
			"output.trackingPlanArgs.events[2].id",
			"output.trackingPlanArgs.events[2].properties[0].id",
			"output.trackingPlanArgs.events[2].properties[1].id",
		},
	)

	err = tester.SnapshotTest(context.Background())
	assert.NoError(t, err, "State verification failed")

	expectedStateDir = filepath.Join("testdata", "expected", "upstream", dir)
	fileManager, err = helpers.NewSnapshotFileManager(expectedStateDir)
	require.NoError(t, err)

	upstreamTester := helpers.NewUpstreamSnapshotTester(
		dataCatalog,
		reader,
		fileManager,
		[]string{
			"id",
			"createdAt",
			"updatedAt",
			"createdBy",
			"updatedBy",
			"workspaceId",
			"categoryId",
			"version",
			"properties[0].id",
			"properties[1].id",
			"events[0].properties[0].id",
			"events[0].properties[1].id",
			"events[0].properties[2].id",
			"events[0].properties[3].id",
			"events[0].id",
			"events[0].createdAt",
			"events[0].updatedAt",
			"events[0].workspaceId",
			"events[0].createdBy",
			"events[0].updatedBy",
			"events[0].categoryId",
			"events[1].properties[0].id",
			"events[1].properties[1].id",
			"events[1].properties[2].id",
			"events[1].properties[2].properties[0].id",
			"events[1].properties[2].properties[0].properties[0].id",
			"events[1].properties[2].properties[0].properties[0].properties[0].id",
			"events[1].properties[2].properties[0].properties[1].id",
			"events[1].properties[2].properties[1].id",
			"events[1].properties[2].properties[1].properties[0].id",
			"events[1].properties[2].properties[1].properties[1].id",
			"events[1].properties[2].properties[1].properties[1].properties[0].id",
			"events[1].properties[3].id",
			"events[2].properties[0].id",
			"events[2].properties[1].id",
			"events[2].properties[1].properties[0].id",
			"events[2].properties[1].properties[0].properties[0].id",
			"events[2].properties[1].properties[0].properties[0].properties[0].id",
			"events[2].properties[1].properties[0].properties[0].properties[1].id",
			"events[2].properties[1].properties[1].id",
			"events[2].properties[2].id",
			"events[2].properties[2].properties[0].id",
			"events[2].properties[2].properties[1].id",
			"events[2].properties[2].properties[0].properties[0].id",
			"events[2].properties[2].properties[0].properties[1].id",
			"events[2].properties[2].properties[0].properties[0].properties[0].id",
			"events[2].properties[2].properties[0].properties[0].properties[1].id",
			"events[1].id",
			"events[1].createdAt",
			"events[1].updatedAt",
			"events[1].workspaceId",
			"events[1].createdBy",
			"events[1].updatedBy",
			"events[1].categoryId",
			"events[2].properties[0].id",
			"events[2].properties[1].id",
			"events[2].id",
			"events[2].createdAt",
			"events[2].updatedAt",
			"events[2].workspaceId",
			"events[2].createdBy",
			"events[2].updatedBy",
			"events[2].categoryId",
		},
	)
	err = upstreamTester.SnapshotTest(context.Background())
	assert.NoError(t, err, "Upstream state verification failed")
}
