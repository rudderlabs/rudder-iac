package apply

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
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
	for _, secret := range []string{"user", "password", "secret", "token=abc", "fragment"} {
		assert.NotContains(t, output.String(), secret)
	}
}

func TestApplyPrintsDiscoveredRegion(t *testing.T) {
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
	config.InitConfig("")
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

func TestApplyTracksRunFailureAsErrored(t *testing.T) {
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
