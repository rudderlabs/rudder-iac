package validate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const manifestHeader = `version: rudder/v1
kind: import-manifest
metadata:
  name: import-manifest
spec:
  workspaces:
`

// projectWithManifest copies the valid fixture into a temp dir and replaces its
// manifest, so each case differs from the valid project in one place.
func projectWithManifest(t *testing.T, workspaces string) string {
	t.Helper()
	dir := t.TempDir()

	props, err := os.ReadFile("testdata/valid/properties.yaml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "properties.yaml"), props, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "import-manifest.yaml"), []byte(manifestHeader+workspaces), 0o600))
	return dir
}

func TestValidateRunsOffline(t *testing.T) {
	invalidDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir, "properties.yaml"), []byte(`version: rudder/v1
kind: properties
metadata:
  name: broken
spec:
  properties:
    - id: missing_name_and_type
`), 0o600))

	// The orphan sits in ws-2 while ws-1 is fine. Without a workspace ID,
	// validate checks every block, so it fails even though an apply targeting
	// ws-1 accepts the file.
	orphanDir := projectWithManifest(t, `    - workspace_id: "ws-1"
      resources:
        - urn: "property:email_address"
          remote_id: "remote-ws-1"
    - workspace_id: "ws-2"
      resources:
        - urn: "property:does_not_exist"
          remote_id: "remote-ws-2"
`)

	// Apply rejects an entry with both urn and local_id when it broadcasts the
	// manifest, and validate never broadcasts, so validate must reject it too.
	urnAndLocalIDDir := projectWithManifest(t, `    - workspace_id: "ws-1"
      resources:
        - urn: "property:email_address"
          local_id: "email_address"
          remote_id: "remote-ws-1"
`)

	cases := []struct {
		name     string
		token    string
		location string
		wantErr  bool
	}{
		{name: "no token, valid project", token: "", location: "testdata/valid"},
		{name: "invalid token, valid project", token: "not-a-real-token", location: "testdata/valid"},
		{name: "no token, invalid project fails locally", token: "", location: invalidDir, wantErr: true},
		{name: "no token, orphan in another workspace block", token: "", location: orphanDir, wantErr: true},
		{name: "no token, urn with local_id", token: "", location: urnAndLocalIDDir, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Stands in for the RudderStack API and fails the test on any
			// request, so a run that reaches the network cannot pass silently.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("validate must not call the API, got %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusUnauthorized)
			}))
			t.Cleanup(server.Close)

			t.Setenv("RUDDERSTACK_ACCESS_TOKEN", tc.token)
			t.Setenv("RUDDERSTACK_API_URL", server.URL)
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
			// The import-manifest kind is gated; enabling it exercises the
			// workspace-aware rules that used to need the workspace lookup.
			t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
			t.Setenv("RUDDERSTACK_X_IMPORT_MERGE", "true")
			config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

			cmd := NewCmdValidate()
			cmd.SetArgs([]string{"--location", tc.location})
			cmd.SilenceUsage = true

			err := cmd.Execute()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "validating project")
				return
			}
			require.NoError(t, err)
		})
	}
}
