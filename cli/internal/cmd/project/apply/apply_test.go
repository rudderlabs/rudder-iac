package apply

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Guards the other side of the offline validate change: apply must keep
// requiring a token and resolving the workspace before touching the project.
func TestApplyRequiresAuth(t *testing.T) {
	cases := []struct {
		name          string
		token         string
		wantErr       string
		wantAPICalled bool
	}{
		{name: "no token", token: "", wantErr: "access token is required"},
		{name: "invalid token", token: "not-a-real-token", wantErr: "fetching workspace information", wantAPICalled: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var workspaceLookups atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/workspace" {
					workspaceLookups.Add(1)
				}
				w.WriteHeader(http.StatusUnauthorized)
			}))
			t.Cleanup(server.Close)

			t.Setenv("RUDDERSTACK_ACCESS_TOKEN", tc.token)
			t.Setenv("RUDDERSTACK_API_URL", server.URL)
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
			config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

			cmd := NewCmdApply()
			cmd.SetArgs([]string{"--location", t.TempDir(), "--dry-run"})
			cmd.SilenceUsage = true

			err := cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Equal(t, tc.wantAPICalled, workspaceLookups.Load() > 0)
		})
	}
}
