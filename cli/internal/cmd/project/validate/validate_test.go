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

// newForbiddenAPI stands in for the RudderStack API and fails the test on any
// request, so a validate run that reaches the network cannot pass silently.
func newForbiddenAPI(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("validate must not call the API, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	return server
}

func setupEnv(t *testing.T, token, apiURL string) {
	t.Helper()
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", token)
	t.Setenv("RUDDERSTACK_API_URL", apiURL)
	t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
	// The import-manifest kind is gated; enabling it exercises the
	// workspace-aware rules that used to need the workspace lookup.
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_IMPORT_MERGE", "true")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
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

	cases := []struct {
		name     string
		token    string
		location string
		wantErr  bool
	}{
		{name: "no token, valid project", token: "", location: "testdata/valid"},
		{name: "invalid token, valid project", token: "not-a-real-token", location: "testdata/valid"},
		{name: "no token, invalid project fails locally", token: "", location: invalidDir, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupEnv(t, tc.token, newForbiddenAPI(t).URL)

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
