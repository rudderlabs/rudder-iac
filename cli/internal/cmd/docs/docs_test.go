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
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
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

// supportedTypes lists every compiled destination type, unverified ones
// included, as the docs command registers them.
func supportedTypes(t *testing.T) []string {
	t.Helper()

	registry, err := builtin.NewRegistry(true)
	require.NoError(t, err)
	return registry.SupportedTypes()
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
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			first, _, err := run(t, args...)
			require.NoError(t, err)
			second, _, err := run(t, args...)
			require.NoError(t, err)
			assert.Equal(t, first, second)
		})
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
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, _, err := run(t, args...)
			require.NoError(t, err)
			assert.NotEmpty(t, stdout)
		})
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
	assert.Equal(t, specdocs.ErrorDoc{
		SchemaVersion: 1,
		CLIVersion:    testVersion,
		Error: specdocs.Error{
			Code:        "unknown_type",
			Message:     `destination type "bigquery" is not supported`,
			Suggestions: []string{"bq", "bqstream"},
			ValidTypes:  supportedTypes(t),
		},
	}, doc)
	assert.Contains(t, doc.Error.ValidTypes, "adj", "unverified types are listed too")
}

func TestUnknownTypeWithMarkdownReturnsTheError(t *testing.T) {
	telemetrytest.Record(t)

	stdout, _, err := run(t, "docs", "destination", "bigquery")

	require.EqualError(t, err, `destination type "bigquery" is not supported; did you mean bq, bqstream? `+
		"Valid destination types: "+strings.Join(supportedTypes(t), ", ")+". "+
		"Run `rudder-cli docs destination` for their display names.")
	var silent *cmderrors.SilentError
	assert.False(t, errors.As(err, &silent), "the root prints the message")
	assert.Empty(t, stdout)
}

func TestInputErrorsWithJSON(t *testing.T) {
	telemetrytest.Record(t)

	var (
		destTypes   = supportedTypes(t)
		sourceTypes = slices.Sorted(slices.Values(source.SourceTypes()))
		idAdvice    = "use an ID of 2 to 100 letters, digits, underscores and hyphens that does not start with a digit"
		// displayNameRule is the display_name pattern's message.
		displayNameRule = "must be 2-100 characters and contain only letters, digits, underscores, spaces, periods, and hyphens"
	)
	cases := []struct {
		name string
		args []string
		want specdocs.Error
	}{
		{
			name: "a source token in place of a source type",
			args: []string{"docs", "destination", "webhook", "--source", "web", "--json"},
			want: specdocs.Error{
				Code:        "unknown_source",
				Message:     `source type "web" is not an event stream source spec type`,
				Suggestions: []string{"javascript"},
				ValidTypes:  sourceTypes,
			},
		},
		{
			name: "a listing narrowed to a source token",
			args: []string{"docs", "destination", "--source", "web", "--json"},
			want: specdocs.Error{
				Code:        "unknown_source",
				Message:     `source type "web" is not an event stream source spec type`,
				Suggestions: []string{"javascript"},
				ValidTypes:  sourceTypes,
			},
		},
		{
			name: "a source the type does not accept",
			args: []string{"docs", "destination", "adj", "--source", "javascript", "--json"},
			want: specdocs.Error{
				Code:        "unsupported_source",
				Message:     `destination type "adj" does not accept javascript sources, which connect as "web"`,
				Suggestions: []string{},
				ValidTypes: []string{
					"android", "android_kotlin", "cordova", "dotnet", "flutter", "go", "ios", "ios_swift",
					"java", "node", "php", "python", "react_native", "ruby", "rust", "unity",
				},
			},
		},
		{
			name: "an empty --source, as an unset shell variable gives",
			args: []string{"docs", "destination", "webhook", "--source", "", "--id", "shop", "--json"},
			want: specdocs.Error{
				Code:        "unknown_source",
				Message:     `source type "" is not an event stream source spec type`,
				Suggestions: []string{},
				ValidTypes:  sourceTypes,
			},
		},
		{
			name: "a listing narrowed to an empty --source",
			args: []string{"docs", "destination", "--source", "", "--json"},
			want: specdocs.Error{
				Code:        "unknown_source",
				Message:     `source type "" is not an event stream source spec type`,
				Suggestions: []string{},
				ValidTypes:  sourceTypes,
			},
		},
		{
			name: "an empty --id",
			args: []string{"docs", "destination", "webhook", "--source", "javascript", "--id", "", "--json"},
			want: specdocs.Error{
				Code:        "invalid_id",
				Message:     `--id "" is also the spec's display_name, which ` + displayNameRule + "; " + idAdvice,
				Suggestions: []string{},
			},
		},
		{
			name: "an empty --id without a type",
			args: []string{"docs", "destination", "--id=", "--json"},
			want: specdocs.Error{
				Code:        "invalid_id",
				Message:     `--id "" is also the spec's display_name, which ` + displayNameRule + "; " + idAdvice,
				Suggestions: []string{},
			},
		},
		{
			name: "an ID that makes an invalid variable name",
			args: []string{"docs", "destination", "webhook", "--id", "my.dest", "--json"},
			want: specdocs.Error{
				Code: "invalid_id",
				Message: `--id "my.dest" gives the secret variable name "MY.DEST", which does not match ` +
					"^[A-Za-z_][A-Za-z0-9_]*$; " + idAdvice,
				Suggestions: []string{},
			},
		},
		{
			name: "an unknown definition version",
			args: []string{"docs", "destination", "webhook", "--definition-version", "9", "--json"},
			want: specdocs.Error{
				Code:          "unknown_definition_version",
				Message:       `destination type "webhook" has no definition version 9`,
				Suggestions:   []string{},
				ValidVersions: []int64{1},
			},
		},
		{
			name: "an ID without a type",
			args: []string{"docs", "destination", "--id", "shop", "--json"},
			want: specdocs.Error{
				Code:        "missing_type",
				Message:     "--id and --definition-version need a destination type",
				Suggestions: []string{},
				ValidTypes:  destTypes,
			},
		},
		{
			name: "a definition version without a type",
			args: []string{"docs", "destination", "--definition-version", "1", "--json"},
			want: specdocs.Error{
				Code:        "missing_type",
				Message:     "--id and --definition-version need a destination type",
				Suggestions: []string{},
				ValidTypes:  destTypes,
			},
		},
		{
			name: "an empty type",
			args: []string{"docs", "destination", "", "--json"},
			want: specdocs.Error{
				Code:        "missing_type",
				Message:     "the destination type argument is empty",
				Suggestions: []string{},
				ValidTypes:  destTypes,
			},
		},
		{
			name: "an extra argument",
			args: []string{"docs", "destination", "webhook", "extra", "--json"},
			want: specdocs.Error{
				Code:        "invalid_arguments",
				Message:     "docs destination takes at most one destination type, got 2 arguments",
				Suggestions: []string{},
			},
		},
		{
			name: "--json with --format md",
			args: []string{"docs", "destination", "webhook", "--format", "md", "--json"},
			want: specdocs.Error{
				Code:        "invalid_arguments",
				Message:     `--json and --format "md" ask for different formats; pass one of them`,
				Suggestions: []string{},
			},
		},
		{
			name: "-j with an unknown --format",
			args: []string{"docs", "destination", "webhook", "--format", "yaml", "-j"},
			want: specdocs.Error{
				Code:        "invalid_arguments",
				Message:     `--json and --format "yaml" ask for different formats; pass one of them`,
				Suggestions: []string{},
			},
		},
		{
			name: "an argument to docs source",
			args: []string{"docs", "source", "extra", "--json"},
			want: specdocs.Error{
				Code:        "invalid_arguments",
				Message:     "docs source takes no arguments, got 1",
				Suggestions: []string{},
			},
		},
		{
			name: "docs source with --json and --format md",
			args: []string{"docs", "source", "--format", "md", "--json"},
			want: specdocs.Error{
				Code:        "invalid_arguments",
				Message:     `--json and --format "md" ask for different formats; pass one of them`,
				Suggestions: []string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := run(t, tc.args...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			assert.Empty(t, stderr)

			var doc specdocs.ErrorDoc
			require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
			assert.Equal(t, specdocs.ErrorDoc{SchemaVersion: 1, CLIVersion: testVersion, Error: tc.want}, doc)
		})
	}
}

// Without --json, an input error is the command's error, which the root prints.
func TestFlagErrors(t *testing.T) {
	telemetrytest.Record(t)

	var (
		destTypes   = strings.Join(supportedTypes(t), ", ")
		sourceTypes = strings.Join(slices.Sorted(slices.Values(source.SourceTypes())), ", ")
	)
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "an unknown --format",
			args:    []string{"docs", "destination", "webhook", "--format", "yaml"},
			wantErr: `--format must be md or json, got "yaml".`,
		},
		{
			name: "an ID without a type",
			args: []string{"docs", "destination", "--id", "shop"},
			wantErr: "--id and --definition-version need a destination type. " +
				"Valid destination types: " + destTypes + ". " +
				"Run `rudder-cli docs destination` for their display names.",
		},
		{
			name:    "an empty --source",
			args:    []string{"docs", "destination", "webhook", "--source", ""},
			wantErr: `source type "" is not an event stream source spec type. Valid source types: ` + sourceTypes + ".",
		},
		{
			name:    "an extra argument",
			args:    []string{"docs", "destination", "webhook", "am"},
			wantErr: "docs destination takes at most one destination type, got 2 arguments.",
		},
		{
			name:    "an argument to docs source",
			args:    []string{"docs", "source", "extra"},
			wantErr: "docs source takes no arguments, got 1.",
		},
		{
			name:    "an unknown subcommand",
			args:    []string{"docs", "nothing"},
			wantErr: `unknown command "nothing" for "rudder-cli docs"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := run(t, tc.args...)
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

// --format json and --json agree, so together they print JSON.
func TestFormatAndJSONFlagsThatAgree(t *testing.T) {
	telemetrytest.Record(t)

	stdout, stderr, err := run(t, "docs", "destination", "webhook", "--format", "json", "--json")
	require.NoError(t, err)
	assert.Empty(t, stderr)

	var doc specdocs.DestinationDoc
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Equal(t, "webhook", doc.Type)
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
	_, _, _ = run(t, "docs", "source", "--format", "private-text")
	_, _, _ = run(t, "docs", "destination", "", "--json")

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
		{
			Command: "docs source",
			Errored: true,
			Extras:  []telemetry.KV{{K: "format", V: "unknown"}},
		},
		{
			Command: "docs destination",
			Errored: true,
			Extras: []telemetry.KV{
				{K: "type", V: ""},
				{K: "format", V: "json"},
				{K: "source", V: ""},
				{K: "listing", V: false},
				{K: "has_id", V: false},
			},
		},
	}, *calls)
}
