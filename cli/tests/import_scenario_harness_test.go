package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// varReference matches the "{{ .VAR }}" token an exported secret is masked to,
// capturing the variable name so the scaffolded var file can be checked for the
// matching placeholder.
var varReference = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

type Scenario struct {
	Name      string
	Exclusive bool
	Seed      func(t *testing.T, c *client.Client) []Seeded
	Assert    func(t *testing.T, importedDir string, seeded []Seeded)
}

type Seeded struct {
	Type     string
	Name     string
	RemoteID string
	Secrets  []string
	Cleanup  func(ctx context.Context) error
}

type cleanupEntry struct {
	seeded  Seeded
	cleaned bool
}

type cleanupRegistry struct {
	entries []cleanupEntry
}

func (r *cleanupRegistry) Len() int {
	return len(r.entries)
}

func (r *cleanupRegistry) Add(seeded []Seeded) {
	for _, item := range seeded {
		if item.Cleanup == nil {
			continue
		}
		r.entries = append(r.entries, cleanupEntry{seeded: item})
	}
}

func (r *cleanupRegistry) CleanupFrom(t *testing.T, start int) {
	t.Helper()
	for i := len(r.entries) - 1; i >= start; i-- {
		if r.entries[i].cleaned {
			continue
		}
		r.entries[i].cleaned = true
		seeded := r.entries[i].seeded
		if err := seeded.Cleanup(context.Background()); err != nil {
			t.Logf("cleaning up seeded %s %s (%s): %v", seeded.Type, seeded.Name, seeded.RemoteID, err)
		}
	}
}

func runImportScenarios(t *testing.T, scenarios []Scenario) {
	t.Helper()

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)
	apiClient := newAPIClient(t)

	cleanups := &cleanupRegistry{}
	t.Cleanup(func() { cleanups.CleanupFrom(t, 0) })

	var shared []Scenario
	for _, scenario := range scenarios {
		if scenario.Exclusive {
			continue
		}
		shared = append(shared, scenario)
	}

	if len(shared) > 0 {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		require.NoError(t, err, "destroy failed: %s", out)

		start := cleanups.Len()
		seededByScenario := make(map[string][]Seeded, len(shared))
		for _, scenario := range shared {
			seeded := scenario.Seed(t, apiClient)
			seededByScenario[scenario.Name] = seeded
			cleanups.Add(seeded)
		}

		projectDir := t.TempDir()
		out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
		require.NoError(t, err, "import workspace failed: %s", out)
		assertNoSecretInBytes(t, out, collectSecrets(seededByScenario))

		importedDir := filepath.Join(projectDir, importer.ImportedDir)
		for _, scenario := range shared {
			scenario := scenario
			t.Run(scenario.Name, func(t *testing.T) {
				scenario.Assert(t, importedDir, seededByScenario[scenario.Name])
			})
		}
		cleanups.CleanupFrom(t, start)
	}

	for _, scenario := range scenarios {
		if !scenario.Exclusive {
			continue
		}

		scenario := scenario
		t.Run(scenario.Name, func(t *testing.T) {
			out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
			require.NoError(t, err, "destroy failed: %s", out)

			start := cleanups.Len()
			seeded := scenario.Seed(t, apiClient)
			cleanups.Add(seeded)

			projectDir := t.TempDir()
			out, err = executor.Execute(cliBinPath, "import", "workspace", "-l", projectDir)
			require.NoError(t, err, "import workspace failed: %s", out)
			assertNoSecretInBytes(t, out, secretsFromSeeded(seeded))

			scenario.Assert(t, filepath.Join(projectDir, importer.ImportedDir), seeded)
			cleanups.CleanupFrom(t, start)
		})
	}
}

func newAPIClient(t *testing.T) *client.Client {
	t.Helper()

	config.InitConfig(config.DefaultConfigFile())
	apiClient, err := client.New(
		config.GetConfig().Auth.AccessToken,
		client.WithBaseURL(config.GetConfig().APIURL),
		client.WithUserAgent("rudder-cli-test"),
	)
	require.NoError(t, err)
	return apiClient
}

// newAccountsAPIClient is retained for account apply tests that use the same
// config-backed API client as the CLI binary.
func newAccountsAPIClient(t *testing.T) *client.Client {
	t.Helper()
	return newAPIClient(t)
}

type importedSpec struct {
	Path string
	Raw  string
	Spec *specs.Spec
}

func findSpecByName(t *testing.T, dir, name string) importedSpec {
	t.Helper()

	var matches []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
			matches = append(matches, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, matches, "expected scaffolded specs in %s", dir)

	for _, path := range matches {
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		var spec specs.Spec
		if err := yaml.Unmarshal(quoteVarTokensForYAML(content), &spec); err != nil {
			continue
		}
		if specName, ok := spec.Spec["name"].(string); ok && specName == name {
			return importedSpec{Path: path, Raw: string(content), Spec: &spec}
		}
	}

	t.Fatalf("no scaffolded spec found for name %q among %v", name, matches)
	return importedSpec{}
}

func quoteVarTokensForYAML(data []byte) []byte {
	return varReference.ReplaceAllFunc(data, func(match []byte) []byte {
		return []byte(strconv.Quote(string(match)))
	})
}

func filterManifest(t *testing.T, path string, remoteIDs ...string) {
	t.Helper()

	remoteIDSet := make(map[string]struct{}, len(remoteIDs))
	for _, remoteID := range remoteIDs {
		remoteIDSet[remoteID] = struct{}{}
	}

	var kept []importmanifest.ImportEntry
	for _, entry := range manifestEntries(t, path) {
		if _, ok := remoteIDSet[entry.RemoteID]; ok {
			kept = append(kept, entry)
		}
	}
	require.Len(t, kept, len(remoteIDs), "manifest must map the requested remote ids")

	node, err := importmanifest.BuildNode(kept)
	require.NoError(t, err)
	filtered, err := yaml.Marshal(node)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, filtered, 0o644))
}

func pruneTo(t *testing.T, importedDir string, keepSpecs []string, remoteIDs []string) {
	t.Helper()

	keep := make(map[string]struct{}, len(keepSpecs))
	for _, path := range keepSpecs {
		keep[path] = struct{}{}
	}

	err := filepath.WalkDir(importedDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path == filepath.Join(importedDir, importmanifest.FileName) || path == filepath.Join(importedDir, importer.SecretsVarFileName) {
			return err
		}
		if _, ok := keep[path]; ok {
			return nil
		}
		return os.Remove(path)
	})
	require.NoError(t, err)
	filterManifest(t, filepath.Join(importedDir, importmanifest.FileName), remoteIDs...)
}

func manifestEntryByRemoteID(t *testing.T, importedDir, remoteID string) importmanifest.ImportEntry {
	t.Helper()

	entries := manifestEntries(t, filepath.Join(importedDir, importmanifest.FileName))
	for _, entry := range entries {
		if entry.RemoteID == remoteID {
			return entry
		}
	}

	t.Fatalf("manifest must map remote id %s; entries: %#v", remoteID, entries)
	return importmanifest.ImportEntry{}
}

func manifestEntries(t *testing.T, path string) []importmanifest.ImportEntry {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var manifest struct {
		Spec specs.WorkspacesImportMetadata `yaml:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &manifest))

	var entries []importmanifest.ImportEntry
	for _, workspace := range manifest.Spec.Workspaces {
		for _, resource := range workspace.Resources {
			entries = append(entries, importmanifest.ImportEntry{
				WorkspaceID: workspace.WorkspaceID,
				URN:         resource.URN,
				RemoteID:    resource.RemoteID,
			})
		}
	}
	return entries
}

func assertNoSecretOnDisk(t *testing.T, dir string, secrets []string) {
	t.Helper()

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		assertNoSecretInBytes(t, content, secrets, "raw secret leaked into %s", strings.TrimPrefix(path, dir))
		return nil
	})
	require.NoError(t, err)
}

func assertNoSecretInBytes(t *testing.T, data []byte, secrets []string, msgAndArgs ...any) {
	t.Helper()
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if len(msgAndArgs) > 0 {
			assert.NotContains(t, string(data), secret, msgAndArgs...)
			continue
		}
		assert.NotContains(t, string(data), secret, "raw secret must not appear")
	}
}

func collectSecrets(seededByScenario map[string][]Seeded) []string {
	var secrets []string
	for _, seeded := range seededByScenario {
		secrets = append(secrets, secretsFromSeeded(seeded)...)
	}
	return secrets
}

func secretsFromSeeded(seeded []Seeded) []string {
	var secrets []string
	for _, item := range seeded {
		secrets = append(secrets, item.Secrets...)
	}
	return secrets
}

func uniqueImportName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d-%d", prefix, os.Getpid(), time.Now().UnixNano())
}
