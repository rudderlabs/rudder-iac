package docs

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata")

// testVersion stands in for the release version so golden files stay stable.
const testVersion = "1.2.3"

// run executes `rudder-cli <args>` against a fresh command tree, the way the
// real root mounts the docs group. Tests using it must not run in parallel,
// because telemetrytest.Record swaps package state.
func run(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	root := &cobra.Command{Use: "rudder-cli", Version: testVersion, SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(NewCmdDocs())

	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if *update {
		require.NoError(t, os.WriteFile(path, []byte(got), 0o600))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "run go test with -update to create %s", path)
	assert.Equal(t, string(want), got)
}

func TestDestinationGolden(t *testing.T) {
	telemetrytest.Record(t)

	cases := []struct {
		golden string
		args   []string
	}{
		{golden: "webhook.json", args: []string{"docs", "destination", "webhook", "--source", "javascript", "--id", "storefront-webhook", "--format", "json"}},
		{golden: "am.json", args: []string{"docs", "destination", "am", "--source", "javascript", "--id", "storefront-amplitude", "--json"}},
		{golden: "webhook.md", args: []string{"docs", "destination", "webhook", "--source", "javascript", "--id", "storefront-webhook"}},
	}

	for _, tc := range cases {
		t.Run(tc.golden, func(t *testing.T) {
			stdout, stderr, err := run(t, tc.args...)
			require.NoError(t, err)
			assert.Empty(t, stderr)
			assertGolden(t, tc.golden, stdout)
		})
	}
}

// The output may depend only on the binary and the flags.
func TestOutputIsByteIdenticalAcrossRuns(t *testing.T) {
	telemetrytest.Record(t)

	for _, args := range [][]string{
		{"docs", "destination", "am", "--source", "javascript", "--id", "storefront-amplitude", "--json"},
		{"docs", "destination", "braze", "--id", "shop-braze"},
		{"docs", "destination", "--json"},
		{"docs", "destination"},
		{"docs", "source", "--json"},
	} {
		first, _, err := run(t, args...)
		require.NoError(t, err)
		second, _, err := run(t, args...)
		require.NoError(t, err)
		assert.Equal(t, first, second, args)
	}
}

func TestDocsRunOffline(t *testing.T) {
	telemetrytest.Record(t)

	// Stands in for the RudderStack API and fails the test on any request.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("docs must not call the API, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	t.Setenv("RUDDERSTACK_API_URL", server.URL)
	t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	for _, args := range [][]string{
		{"docs", "destination"},
		{"docs", "destination", "adj"},
		{"docs", "destination", "webhook", "--source", "javascript", "--id", "storefront-webhook", "--json"},
		{"docs", "source"},
	} {
		stdout, _, err := run(t, args...)
		require.NoError(t, err, args)
		assert.NotEmpty(t, stdout, args)
	}
}

func TestUnknownTypeWithJSONPrintsAnErrorDocument(t *testing.T) {
	telemetrytest.Record(t)

	stdout, stderr, err := run(t, "docs", "destination", "bigquery", "--json")

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent, "the root exits 1 without printing to stderr")
	assert.Empty(t, stderr)

	var doc specdocs.ErrorDoc
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Equal(t, 1, doc.SchemaVersion)
	assert.Equal(t, testVersion, doc.CLIVersion)
	assert.Equal(t, "unknown_type", doc.Error.Code)
	assert.Equal(t, `destination type "bigquery" is not supported`, doc.Error.Message)
	assert.Equal(t, []string{"bq", "bqstream"}, doc.Error.Suggestions)
	assert.Contains(t, doc.Error.ValidTypes, "webhook")
	assert.Contains(t, doc.Error.ValidTypes, "adj", "unverified types are listed too")
}

func TestUnknownTypeWithMarkdownReturnsTheError(t *testing.T) {
	telemetrytest.Record(t)

	stdout, _, err := run(t, "docs", "destination", "bigquery")

	require.EqualError(t, err, `destination type "bigquery" is not supported; did you mean bq, bqstream? `+
		"Run `rudder-cli docs destination` to list the types.")
	var silent *cmderrors.SilentError
	assert.False(t, errors.As(err, &silent), "the root prints the message")
	assert.Empty(t, stdout)
}

func TestInputErrorsWithJSON(t *testing.T) {
	telemetrytest.Record(t)

	cases := []struct {
		args     []string
		wantCode string
	}{
		{args: []string{"docs", "destination", "webhook", "--source", "web", "--json"}, wantCode: "unknown_source"},
		{args: []string{"docs", "destination", "--source", "web", "--json"}, wantCode: "unknown_source"},
		{args: []string{"docs", "destination", "adj", "--source", "javascript", "--json"}, wantCode: "unsupported_source"},
		{args: []string{"docs", "destination", "webhook", "--id", "my.dest", "--json"}, wantCode: "invalid_id"},
		{args: []string{"docs", "destination", "webhook", "--definition-version", "9", "--json"}, wantCode: "unknown_definition_version"},
	}

	for _, tc := range cases {
		stdout, stderr, err := run(t, tc.args...)

		var silent *cmderrors.SilentError
		require.ErrorAs(t, err, &silent, tc.args)
		assert.Empty(t, stderr, tc.args)

		var doc specdocs.ErrorDoc
		require.NoError(t, json.Unmarshal([]byte(stdout), &doc), tc.args)
		assert.Equal(t, tc.wantCode, doc.Error.Code, tc.args)
		assert.NotNil(t, doc.Error.Suggestions, tc.args)
	}
}

func TestFlagErrors(t *testing.T) {
	telemetrytest.Record(t)

	_, _, err := run(t, "docs", "destination", "webhook", "--format", "yaml")
	require.EqualError(t, err, `--format must be md or json, got "yaml"`)

	_, _, err = run(t, "docs", "destination", "webhook", "--format", "md", "--json")
	require.Error(t, err)

	_, _, err = run(t, "docs", "destination", "--id", "shop")
	require.EqualError(t, err, "--id and --definition-version need a destination type")

	_, _, err = run(t, "docs", "destination", "webhook", "am")
	require.Error(t, err)

	_, _, err = run(t, "docs", "nothing")
	require.Error(t, err)
}

func TestSourceListsTheSpecTypes(t *testing.T) {
	telemetrytest.Record(t)

	stdout, _, err := run(t, "docs", "source", "-j")
	require.NoError(t, err)

	var list specdocs.SourceList
	require.NoError(t, json.Unmarshal([]byte(stdout), &list))
	assert.Equal(t, 1, list.SchemaVersion)
	assert.Equal(t, testVersion, list.CLIVersion)

	var types []string
	for _, s := range list.Sources {
		types = append(types, s.Type)
		assert.Equal(t, common.SourceTypeToken(s.Type, ""), s.Token, s.Type)
	}
	assert.ElementsMatch(t, source.SourceTypes(), types)
}

func TestTelemetryNeverCarriesFreeText(t *testing.T) {
	calls := telemetrytest.Record(t)

	_, _, _ = run(t, "docs", "destination", "my-private-thing", "--source", "my-private-source", "--id", "secret-id", "--json")
	_, _, _ = run(t, "docs", "destination", "webhook", "--source", "javascript", "--id", "secret-id")
	_, _, _ = run(t, "docs", "destination", "--json")
	_, _, _ = run(t, "docs", "source")

	assert.Equal(t, []telemetrytest.Call{
		{
			Command: "docs destination",
			Errored: true,
			Extras: []telemetry.KV{
				{K: "type", V: "unknown"},
				{K: "format", V: "json"},
				{K: "source", V: "unknown"},
				{K: "listing", V: false},
				{K: "has_id", V: true},
			},
		},
		{
			Command: "docs destination",
			Extras: []telemetry.KV{
				{K: "type", V: "webhook"},
				{K: "format", V: "md"},
				{K: "source", V: "javascript"},
				{K: "listing", V: false},
				{K: "has_id", V: true},
			},
		},
		{
			Command: "docs destination",
			Extras: []telemetry.KV{
				{K: "type", V: ""},
				{K: "format", V: "json"},
				{K: "source", V: ""},
				{K: "listing", V: true},
				{K: "has_id", V: false},
			},
		},
		{
			Command: "docs source",
			Extras:  []telemetry.KV{{K: "format", V: "md"}},
		},
	}, *calls)
}
