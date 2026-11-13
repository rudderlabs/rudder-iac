package apply

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintWorkspaceBannerEU(t *testing.T) {
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)

	printWorkspaceBanner(&client.Workspace{ID: "ws-123", Name: "Acme EU", Region: "EU"}, client.BASE_URL_EU)
	assert.Contains(t, output.String(), "Region: EU")
	assert.Contains(t, output.String(), "API: "+client.BASE_URL_EU)
}

func TestPrintWorkspaceBannerDefaultAndOverride(t *testing.T) {
	for _, tc := range []struct {
		name, apiURL, expected string
	}{
		{"default", client.BASE_URL, ""},
		{"explicit override", "https://api.custom.test", "API: https://api.custom.test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			ui.SetWriter(&output)
			t.Cleanup(ui.RestoreWriter)

			printWorkspaceBanner(&client.Workspace{ID: "ws-123", Name: "Acme", Region: "US"}, tc.apiURL)
			assert.Contains(t, output.String(), "Workspace: Acme (ws-123)  Region: US")
			if tc.expected == "" {
				assert.NotContains(t, output.String(), "API:")
			} else {
				assert.Contains(t, output.String(), tc.expected)
			}
		})
	}
}

func TestPrintWorkspaceBannerRedactsURLSecrets(t *testing.T) {
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)

	printWorkspaceBanner(&client.Workspace{ID: "ws-123", Name: "Acme", Region: "EU"}, "https://user:password@api.custom.test/secret?token=abc#fragment")
	assert.Contains(t, output.String(), "API: https://api.custom.test")
	assert.Contains(t, output.String(), "API: https://api.custom.test/secret")
	for _, secret := range []string{"user", "password", "token=abc", "fragment"} {
		assert.NotContains(t, output.String(), secret)
	}
}

func TestPrintWorkspaceBannerOmitsEmptyRegion(t *testing.T) {
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)
	printWorkspaceBanner(&client.Workspace{ID: "ws-123", Name: "Acme"}, client.BASE_URL)
	assert.Equal(t, "Workspace: Acme (ws-123)\n", output.String())
}

func TestApplyPrintsDiscoveredRegion(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	testutils.UseFakeAPIWithRegion(t, "EU")
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", t.TempDir(), "--confirm=false"})
	require.Error(t, err) // The fake API deliberately fails the first project remote load.
	assert.Contains(t, output.String(), "Region: EU")
	assert.Contains(t, output.String(), "API:") // Explicit httptest URL wins even for an EU workspace.
}

func TestApplyWithOverrideFetchesWorkspaceOnce(t *testing.T) {
	var workspaceCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/workspace" {
			workspaceCalls++
			_, _ = w.Write([]byte(`{"workspace":{"id":"ws-123","name":"Acme","region":"EU"}}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"fake API failure"}`))
	}))
	t.Cleanup(server.Close)
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	testutils.SetConfig(t, "apiURL", server.URL)
	testutils.SetConfig(t, "auth.accessToken", "fake-token")
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", t.TempDir(), "--confirm=false"})
	require.Error(t, err)
	assert.Equal(t, 1, workspaceCalls)
	assert.Contains(t, output.String(), "Region: EU")
}

type failingHTTPClient struct{}

func (failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected API request")
}

type cachedWorkspaceDeps struct {
	app.Deps
	client         *client.Client
	workspaceCalls int
}

func (d *cachedWorkspaceDeps) Workspace(context.Context) (*client.Workspace, error) {
	d.workspaceCalls++
	return &client.Workspace{ID: "ws-123", Name: "Acme", Region: "US"}, nil
}

func (d *cachedWorkspaceDeps) Client() *client.Client { return d.client }

func TestApplyUsesCachedWorkspaceWithoutAnotherRequest(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	c, err := client.New("test-token", client.WithHTTPClient(failingHTTPClient{}))
	require.NoError(t, err)
	d := &cachedWorkspaceDeps{client: c}
	var output bytes.Buffer
	ui.SetWriter(&output)
	t.Cleanup(ui.RestoreWriter)
	cmd := newCmdApply(func() (app.Deps, error) { return d, nil })
	cmd.SetArgs([]string{"--var-file", filepath.Join(t.TempDir(), "missing.vars.yaml")})

	err = cmd.Execute()
	require.Error(t, err) // Stops after the banner while reading the missing variable file.
	assert.NotContains(t, err.Error(), "unexpected API request")
	assert.Equal(t, 1, d.workspaceCalls)
	assert.Contains(t, output.String(), "Region: US")
	assert.NotContains(t, output.String(), "API:")
}

func TestApplyTracksRunFailureAsErrored(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)
	location := t.TempDir()

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", location, "--confirm=false"})

	require.Error(t, err)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "apply",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "location", V: location},
			{K: "dryRun", V: false},
			{K: "confirm", V: false},
		},
	}}, *calls)
}
