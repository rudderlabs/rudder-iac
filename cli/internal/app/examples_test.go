package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const examplesFlagsFile = ".flags"

var requiredExampleScenarios = []string{"minimal.yaml", "full.yaml"}

// exampleKindExceptions documents supported kinds that intentionally have no
// canonical example. Keep this empty unless a kind cannot be represented as a
// standalone local project; every entry must include the reason.
var exampleKindExceptions = map[string]string{}

// exampleScenarioExceptions documents kinds that intentionally omit one of the
// required scenario files. Keep this empty unless a scenario cannot be useful for
// that kind; every entry must include the reason.
var exampleScenarioExceptions = map[string]map[string]string{}

type exampleDiagnosticsRecorder struct {
	diagnostics validation.Diagnostics
}

func (r *exampleDiagnosticsRecorder) Render(diagnostics validation.Diagnostics) error {
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return nil
}

type offlineHTTPClient struct {
	t *testing.T
}

func (c offlineHTTPClient) Do(request *http.Request) (*http.Response, error) {
	c.t.Helper()
	c.t.Fatalf("offline example validation attempted %s %s", request.Method, request.URL)
	return nil, fmt.Errorf("network access is disabled during example validation")
}

func TestExamplesValidateOffline(t *testing.T) {
	examplesRoot := filepath.Join("..", "..", "..", "examples")
	entries, err := os.ReadDir(examplesRoot)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			directory := filepath.Join(examplesRoot, entry.Name())
			configureExampleFlags(t, directory)
			config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

			composite := newOfflineCompositeProvider(t)
			recorder := &exampleDiagnosticsRecorder{}
			exampleProject := project.New(composite, project.WithRenderer(recorder))
			err := exampleProject.Load(directory)
			assert.NoError(t, err, "diagnostics:\n%s", describeExampleDiagnostics(recorder.diagnostics))
			assert.Empty(t, recorder.diagnostics.Errors(), "diagnostics:\n%s", describeExampleDiagnostics(recorder.diagnostics))
			assertExampleScenarios(t, entry.Name(), directory, exampleProject.Specs())
		})
	}
}

func TestExamplesCoverSupportedKinds(t *testing.T) {
	configureAllExampleFlags(t)
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	composite := newOfflineCompositeProvider(t)

	examplesRoot := filepath.Join("..", "..", "..", "examples")
	entries, err := os.ReadDir(examplesRoot)
	require.NoError(t, err)

	var exampleKinds []string
	for _, entry := range entries {
		if entry.IsDir() {
			exampleKinds = append(exampleKinds, entry.Name())
		}
	}
	sort.Strings(exampleKinds)

	supportedKinds := expectedExampleKinds(composite, importmanifest.New())
	assert.Equal(t, supportedKinds, exampleKinds, "examples directories must exactly match supported kinds")
}

func newOfflineCompositeProvider(t *testing.T) provider.Provider {
	t.Helper()

	c, err := client.New(
		"offline-example-validation",
		client.WithBaseURL("http://offline.invalid"),
		client.WithHTTPClient(offlineHTTPClient{t: t}),
	)
	require.NoError(t, err)

	composite, _, err := composeProviders(c)
	require.NoError(t, err)
	return composite
}

func expectedExampleKinds(composite provider.Provider, manifestProvider project.ProjectProvider) []string {
	unique := map[string]struct{}{}
	for _, kind := range composite.SupportedKinds() {
		unique[kind] = struct{}{}
	}
	if config.GetConfig().ExperimentalFlags.ImportMerge {
		for _, kind := range manifestProvider.SupportedKinds() {
			unique[kind] = struct{}{}
		}
	}

	supportedKinds := make([]string, 0, len(unique))
	for kind := range unique {
		if _, excepted := exampleKindExceptions[kind]; excepted {
			continue
		}
		supportedKinds = append(supportedKinds, kind)
	}
	sort.Strings(supportedKinds)
	return supportedKinds
}

func assertExampleScenarios(t *testing.T, kind string, directory string, projectSpecs map[string]*specs.Spec) {
	t.Helper()

	for _, scenario := range requiredExampleScenarios {
		if scenarioExempted(kind, scenario) {
			continue
		}
		path := filepath.Join(directory, scenario)
		_, err := os.Stat(path)
		require.NoError(t, err, "examples/%s must include %s", kind, scenario)
		assertSpecFileContainsKind(t, kind, path, projectSpecs)
	}
}

func scenarioExempted(kind string, scenario string) bool {
	kindExceptions, ok := exampleScenarioExceptions[kind]
	if !ok {
		return false
	}
	_, ok = kindExceptions[scenario]
	return ok
}

func assertSpecFileContainsKind(t *testing.T, kind string, path string, projectSpecs map[string]*specs.Spec) {
	t.Helper()

	spec, ok := projectSpecs[path]
	require.True(t, ok, "%s must be loaded as a spec", path)
	assert.Equal(t, kind, spec.Kind, "%s must contain a spec with kind %q", path, kind)
}

func configureExampleFlags(t *testing.T, directory string) {
	t.Helper()
	configureFlags(t, nil)

	data, err := os.ReadFile(filepath.Join(directory, examplesFlagsFile))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)

	flags := strings.Fields(string(data))
	for _, flag := range flags {
		require.True(t, config.IsValidExperimentalFlag(flag), "unknown experimental flag %q in %s", flag, directory)
	}
	configureFlags(t, flags)
}

func configureAllExampleFlags(t *testing.T) {
	t.Helper()

	var all []string
	experimentalType := reflect.TypeOf(config.ExperimentalConfig{})
	for i := range experimentalType.NumField() {
		all = append(all, experimentalType.Field(i).Tag.Get("mapstructure"))
	}
	configureFlags(t, all)
}

func configureFlags(t *testing.T, enabled []string) {
	t.Helper()
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")

	experimentalType := reflect.TypeOf(config.ExperimentalConfig{})
	for i := range experimentalType.NumField() {
		flag := experimentalType.Field(i).Tag.Get("mapstructure")
		t.Setenv(config.GetEnvironmentVariableName(flag), "false")
	}
	for _, flag := range enabled {
		t.Setenv(config.GetEnvironmentVariableName(flag), "true")
	}
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", fmt.Sprintf("%t", len(enabled) > 0))
}

func describeExampleDiagnostics(diagnostics validation.Diagnostics) string {
	lines := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		lines = append(lines, fmt.Sprintf(
			"%s:%d:%d %s[%s] %s",
			diagnostic.File,
			diagnostic.Position.Line,
			diagnostic.Position.Column,
			diagnostic.Severity,
			diagnostic.RuleID,
			diagnostic.Message,
		))
	}
	return strings.Join(lines, "\n")
}
