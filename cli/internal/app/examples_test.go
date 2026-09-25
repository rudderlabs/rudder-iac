package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exampleKindExceptions documents kinds that cannot yet have a hermetic example.
// Keep entries narrow and include a rationale; an empty map means full coverage.
var exampleKindExceptions = map[string]string{}

func TestExamplesLoadWithoutDiagnostics(t *testing.T) {
	examplesDir := examplesRoot(t)
	scenarios := discoverExampleScenarios(t, examplesDir)
	require.NotEmpty(t, scenarios, "examples directory must contain scenarios")

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.Name(), func(t *testing.T) {
			configureExamplesTest(t, scenario.Flags)

			diagnostics := &diagnosticCollector{}
			p := project.New(newExamplesProvider(t), project.WithRenderer(diagnostics))
			err := p.Load(scenario.ProjectDir)
			require.NoError(t, err, "diagnostics:\n%s", diagnostics.String())
			assert.Empty(t, diagnostics.items, "examples must load with zero diagnostics:\n%s", diagnostics.String())
		})
	}
}

func TestEverySupportedKindHasExamples(t *testing.T) {
	examplesDir := examplesRoot(t)
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	configureExamplesTest(t, allExperimentalFlags(t))

	provider := newExamplesProvider(t)
	kinds := supportedExampleKinds(provider, true)
	kindSet := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		kindSet[kind] = struct{}{}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		_, supported := kindSet[entry.Name()]
		assert.True(t, supported, "example directory %q does not match a supported kind", entry.Name())
	}

	for _, kind := range kinds {
		if reason, ok := exampleKindExceptions[kind]; ok {
			t.Logf("allowing kind %q without examples: %s", kind, reason)
			continue
		}

		kindDir := filepath.Join(examplesDir, kind)
		assert.DirExists(t, kindDir, "supported kind %q must have an examples directory", kind)
		assert.FileExists(t, filepath.Join(kindDir, "minimal.yaml"), "supported kind %q must have a minimal example", kind)
		assert.FileExists(t, filepath.Join(kindDir, "full.yaml"), "supported kind %q must have a full example", kind)
	}
}

type exampleScenario struct {
	Kind       string
	Primary    string
	ProjectDir string
	Flags      []string
}

func (s exampleScenario) Name() string {
	return s.Kind + "/" + s.Primary
}

func discoverExampleScenarios(t *testing.T, examplesDir string) []exampleScenario {
	t.Helper()

	kindDirs, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	var scenarios []exampleScenario
	for _, kindDir := range kindDirs {
		if !kindDir.IsDir() {
			continue
		}

		kind := kindDir.Name()
		dir := filepath.Join(examplesDir, kind)
		flags := readExampleFlags(t, filepath.Join(dir, ".flags"))
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)

		for _, entry := range entries {
			if entry.IsDir() || !isYAMLFile(entry.Name()) {
				continue
			}

			primary := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if strings.Contains(primary, "-") {
				continue
			}
			assertPrimaryKind(t, filepath.Join(dir, entry.Name()), kind)

			scenarios = append(scenarios, exampleScenario{
				Kind:       kind,
				Primary:    primary,
				ProjectDir: materializeExampleScenario(t, dir, primary),
				Flags:      flags,
			})
		}
	}

	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Name() < scenarios[j].Name() })
	return scenarios
}

func materializeExampleScenario(t *testing.T, sourceDir, primary string) string {
	t.Helper()

	projectDir := t.TempDir()
	entries, err := os.ReadDir(sourceDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		name := entry.Name()
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if base != primary && !strings.HasPrefix(base, primary+"-") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sourceDir, name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, name), data, 0o600))
	}

	return projectDir
}

func assertPrimaryKind(t *testing.T, path, kind string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	raw := &specs.RawSpec{Data: data}
	parsed, err := raw.Parse()
	require.NoError(t, err, "primary example %s must parse", path)
	require.Equal(t, kind, parsed.Kind, "primary example %s kind must match its examples directory", path)
}

func supportedExampleKinds(p provider.Provider, importMergeEnabled bool) []string {
	kindSet := make(map[string]struct{})
	for _, kind := range p.SupportedKinds() {
		kindSet[kind] = struct{}{}
	}
	if importMergeEnabled {
		for _, kind := range importmanifest.New().SupportedKinds() {
			kindSet[kind] = struct{}{}
		}
	}

	kinds := make([]string, 0, len(kindSet))
	for kind := range kindSet {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func isYAMLFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

func examplesRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve examples test path")
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../examples"))
}

func configureExamplesTest(t *testing.T, flags []string) {
	t.Helper()

	viper.Reset()
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "")
	for _, flag := range allExperimentalFlags(t) {
		t.Setenv(config.GetEnvironmentVariableName(flag), "")
	}

	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	viper.Set("experimental", len(flags) > 0)
	for _, flag := range flags {
		require.True(t, config.IsValidExperimentalFlag(flag), "unknown example flag %q", flag)
		viper.Set("flags."+flag, true)
	}
	t.Cleanup(viper.Reset)
}

func allExperimentalFlags(t *testing.T) []string {
	t.Helper()

	typeOfConfig := reflect.TypeOf(config.ExperimentalConfig{})
	flags := make([]string, 0, typeOfConfig.NumField())
	for i := 0; i < typeOfConfig.NumField(); i++ {
		flag := typeOfConfig.Field(i).Tag.Get("mapstructure")
		require.NotEmpty(t, flag, "experimental config fields must declare mapstructure tags")
		flags = append(flags, flag)
	}
	return flags
}

func newExamplesProvider(t *testing.T) provider.Provider {
	t.Helper()

	c, err := client.New(
		"examples-validation",
		client.WithHTTPClient(&http.Client{Transport: noNetworkTransport{}}),
	)
	require.NoError(t, err)

	cp, _, err := composeProviders(c)
	require.NoError(t, err)
	return cp
}

func readExampleFlags(t *testing.T, path string) []string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)

	var flags []string
	for _, line := range strings.Split(string(contents), "\n") {
		flag := strings.TrimSpace(line)
		if flag == "" || strings.HasPrefix(flag, "#") {
			continue
		}
		flags = append(flags, flag)
	}
	return flags
}

type diagnosticCollector struct {
	items validation.Diagnostics
}

func (r *diagnosticCollector) Render(diagnostics validation.Diagnostics) error {
	r.items = append(r.items, diagnostics...)
	return nil
}

func (r *diagnosticCollector) String() string {
	var output strings.Builder
	for _, diagnostic := range r.items {
		fmt.Fprintf(&output, "%s:%d:%d: %s: %s\n", diagnostic.File, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.RuleID, diagnostic.Message)
	}
	return output.String()
}

type noNetworkTransport struct{}

func (noNetworkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("examples validation attempted network request: %s %s", req.Method, req.URL)
}
