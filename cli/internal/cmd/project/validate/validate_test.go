package validate

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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
		location string
		wantErr  string
	}{
		{name: "no token, valid project", location: "testdata/valid"},
		{name: "no token, invalid project fails locally", location: invalidDir, wantErr: "syntax validation failed"},
		{name: "no token, orphan in another workspace block", location: orphanDir, wantErr: "semantic validation failed"},
		{name: "no token, urn with local_id", location: urnAndLocalIDDir, wantErr: "syntax validation failed"},
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

			t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
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
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// fakeProject stands in for a loaded project. A real one renders its document
// to stdout during Load, which writes the same bytes here.
type fakeProject struct {
	project.Project
	loadErr error
	specs   map[string]*specs.Spec
	doc     string
	out     *bytes.Buffer
}

func (f *fakeProject) Load(string) error {
	f.out.WriteString(f.doc)
	return f.loadErr
}

func (f *fakeProject) Specs() map[string]*specs.Spec { return f.specs }

func TestRun(t *testing.T) {
	t.Parallel()

	legacy := map[string]*specs.Spec{"a.yaml": {Version: specs.SpecVersionV0_1}}

	tests := []struct {
		name       string
		json       bool
		loadErr    error
		specs      map[string]*specs.Spec
		wantErr    string
		wantSilent bool
		wantStdout string
		wantStderr string
	}{
		{
			name:       "json: a clean run leaves stdout to the document",
			json:       true,
			wantStdout: "{doc}",
		},
		{
			name:       "json: a legacy spec notice goes to stderr",
			json:       true,
			specs:      legacy,
			wantStdout: "{doc}",
			wantStderr: "v0.1 spec format is deprecated",
		},
		{
			name:       "json: a validation failure is not printed again",
			json:       true,
			loadErr:    fmt.Errorf("semantic %w", project.ErrValidationFailed),
			wantErr:    "validating project: semantic validation failed",
			wantSilent: true,
			wantStdout: "{doc}",
		},
		{
			name:       "json: an error the document does not hold stays on the error path",
			json:       true,
			loadErr:    errors.New("variable substitution failed"),
			wantErr:    "validating project: variable substitution failed",
			wantStdout: "{doc}",
		},
		{
			name:       "text: a validation failure is returned for printing",
			loadErr:    fmt.Errorf("semantic %w", project.ErrValidationFailed),
			wantErr:    "validating project: semantic validation failed",
			wantStdout: "{doc}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out, errOut bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)

			err := run(cmd, &fakeProject{loadErr: tt.loadErr, specs: tt.specs, doc: "{doc}", out: &out}, ".", tt.json)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
				var silent *cmderrors.SilentError
				assert.Equal(t, tt.wantSilent, errors.As(err, &silent), fmt.Sprintf("silent=%v", tt.wantSilent))
			}
			assert.Equal(t, tt.wantStdout, out.String())
			assert.Contains(t, errOut.String(), tt.wantStderr)
			if tt.wantStderr == "" {
				assert.Empty(t, errOut.String())
			}
		})
	}
}

func TestValidateTracksRunFailureAsErrored(t *testing.T) {
	// Validate runs offline, so no API stand-in is needed; only config.
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	calls := telemetrytest.Record(t)
	location := filepath.Join(t.TempDir(), "missing")

	err := telemetrytest.Execute(NewCmdValidate(), nil, []string{"--location", location})

	require.Error(t, err)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "validate",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "location", V: location},
			{K: "json", V: false},
		},
	}}, *calls)
}

// The flag is what swaps the renderer, so it decides whether stdout holds the
// document. A missing location fails inside Load, after the project exists.
func TestValidateJSONFlagSwapsRenderer(t *testing.T) {
	for _, tt := range []struct {
		name       string
		args       []string
		wantSilent bool
		wantStdout bool
	}{
		{name: "with --json stdout holds the document", args: []string{"--json"}, wantSilent: true, wantStdout: true},
		{name: "without --json stdout stays empty", wantStdout: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Validate runs offline, so only config is needed.
			t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
			config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
			telemetrytest.Record(t)
			location := filepath.Join(t.TempDir(), "missing")

			var out bytes.Buffer
			cmd := NewCmdValidate()
			cmd.SetOut(&out)
			cmd.SilenceUsage = true
			err := telemetrytest.Execute(cmd, nil, append([]string{"--location", location}, tt.args...))

			require.Error(t, err)
			var silent *cmderrors.SilentError
			assert.Equal(t, tt.wantSilent, errors.As(err, &silent))
			assert.Equal(t, tt.wantStdout, strings.Contains(out.String(), "project/load-failed"))
		})
	}
}
