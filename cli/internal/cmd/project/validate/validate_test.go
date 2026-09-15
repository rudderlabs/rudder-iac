package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestValidateRunsOfflineWithoutAccessToken(t *testing.T) {
	initValidateTestConfig(t)

	location := writeCatalogProject(t)
	cmd := NewCmdValidate()
	require.NoError(t, cmd.Flags().Set("location", location))

	require.NoError(t, cmd.PreRunE(cmd, nil))
	require.NoError(t, cmd.RunE(cmd, nil))
}

func TestValidateWorkspaceIDScopesImportManifestOffline(t *testing.T) {
	initValidateTestConfig(t)
	enableImportMerge(t)

	location := writeImportManifestProject(t)
	cmd := NewCmdValidate()
	require.NoError(t, cmd.Flags().Set("location", location))
	require.NoError(t, cmd.Flags().Set("workspace-id", "ws-a"))

	require.NoError(t, cmd.PreRunE(cmd, nil))
	require.NoError(t, cmd.RunE(cmd, nil))
}

func TestValidateWithoutWorkspaceIDChecksAllImportManifestWorkspaces(t *testing.T) {
	initValidateTestConfig(t)
	enableImportMerge(t)

	location := writeImportManifestProject(t)
	cmd := NewCmdValidate()
	require.NoError(t, cmd.Flags().Set("location", location))

	require.NoError(t, cmd.PreRunE(cmd, nil))
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "semantic validation failed")
}

func initValidateTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
	viper.Reset()
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	t.Cleanup(viper.Reset)
}

func enableImportMerge(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.importMerge")
	viper.Set("experimental", true)
	viper.Set("flags.importMerge", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.importMerge", prevFlag)
	})
}

func writeCatalogProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"events.yaml": `version: rudder/v0.1
kind: events
metadata:
  name: checkout
spec:
  events:
    - id: checkout_started
      name: Checkout Started
      event_type: track
`,
		"properties.yaml": `version: rudder/v0.1
kind: properties
metadata:
  name: checkout
spec:
  properties:
    - id: cart_id
      name: cartId
      type: string
`,
		"tracking-plan.yaml": `version: rudder/v0.1
kind: tp
metadata:
  name: checkout
spec:
  id: checkout
  display_name: Checkout
  rules:
    - type: event_rule
      id: checkout_started_rule
      event:
        $ref: "#/events/checkout/checkout_started"
        allow_unplanned: false
      properties:
        - $ref: "#/properties/checkout/cart_id"
          required: true
`,
	}

	writeProjectFiles(t, dir, files)
	return dir
}

func writeImportManifestProject(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"source.yaml": `version: rudder/v1
kind: event-stream-source
metadata:
  name: active-source
spec:
  id: active-source
  name: Active Source
  type: android
`,
		"import-manifest.yaml": `version: rudder/v1
kind: import-manifest
metadata:
  name: import-manifest
spec:
  workspaces:
    - workspace_id: ws-a
      resources:
        - urn: event-stream-source:active-source
          remote_id: remote-active
    - workspace_id: ws-b
      resources:
        - urn: event-stream-source:stale-source
          remote_id: remote-stale
`,
	}

	writeProjectFiles(t, dir, files)
	return dir
}

func writeProjectFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
	}
}
