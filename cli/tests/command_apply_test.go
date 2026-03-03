package tests

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const concurrencyForTest = 1

func TestProjectApply(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_TRANSFORMATIONS", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	output, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "Failed to destroy resources: %v, output: %s", err, string(output))

	t.Run("should create entities in catalog from project", func(t *testing.T) {
		output, err := executor.Execute(cliBinPath, "apply", "-l", filepath.Join("testdata", "project", "create"), "--confirm=false")
		require.NoError(t, err, "Initial apply command failed with output: %s", string(output))
		verifyState(t, "create")
	})

	t.Run("should update entities in catalog from project", func(t *testing.T) {
		time.Sleep(5 * time.Second)
		output, err := executor.Execute(cliBinPath, "apply", "-l", filepath.Join("testdata", "project", "update"), "--confirm=false")
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
	dataCatalog, err := catalog.NewRudderDataCatalog(
		apiClient,
		catalog.WithConcurrency(concurrencyForTest),
		catalog.WithEventUpdateBatchSize(1),
	)
	require.NoError(t, err)
	reader := helpers.NewAPIClientAdapter(dataCatalog)

	expectedStateDir := filepath.Join("testdata", "expected", "upstream", dir)
	fileManager, err := helpers.NewSnapshotFileManager(expectedStateDir)
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
			"definitionId",
			"itemDefinitionId",
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
			"events[0].variants[0].discriminator",
			"events[0].variants[0].cases[0].properties[0].id",
			"events[0].variants[0].cases[0].properties[1].id",
			"events[1].properties[0].id",
			"events[1].properties[1].id",
			"events[1].properties[1].properties[0].id",
			"events[1].properties[1].properties[0].properties[0].id",
			"events[1].properties[1].properties[0].properties[0].properties[0].id",
			"events[1].properties[1].properties[0].properties[1].id",
			"events[1].properties[1].properties[1].id",
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
			"events[2].properties[1].properties[0].properties[1].id",
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
			"events[1].properties[2].id",
			"events[1].properties[3].id",
			"events[1].variants[0].discriminator",
			"events[1].variants[0].cases[0].properties[0].id",
			"events[1].variants[0].cases[1].properties[0].id",
			"events[1].variants[0].default[0].id",
			"events[1].variants[0].default[1].id",
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
