package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// End-to-end shape of DEX-410: importing a resource with a secret yields a
// spec with a quoted variable reference (never a masked literal) plus a var
// file with a placeholder; filling the var file and applying round-trips the
// real value to the backend through the secret type.
//
// Not parallel: toggles the global experimental config.
func TestImportScaffoldsSecretsViaVarSubstitution(t *testing.T) {
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})

	b := backend.NewBackend()
	testDir := t.TempDir()

	// A remote book with a secret the API will never return.
	w, err := b.CreateWriter("Tolkien", "")
	require.NoError(t, err)
	_, err = b.CreateBook("The Hobbit", w.ID, "", "remote-only-access-key")
	require.NoError(t, err)

	// Import into an empty project.
	importProvider := example.NewProvider(b)
	proj := project.New(importProvider)
	require.NoError(t, proj.Load(testDir))
	require.NoError(t, importer.WorkspaceImport(context.Background(), proj, importProvider))

	// The generated spec carries a quoted variable reference, not a mask. The
	// handler names the variable from the resource's identity (BOOK_<id>_...),
	// normalized to the substitution grammar.
	const wantVar = "BOOK_THE_HOBBIT_ACCESS_KEY"
	specBytes, err := os.ReadFile(filepath.Join(testDir, "imported", "books", "books.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(specBytes), `accessKey: "{{ .`+wantVar+` }}"`)
	assert.NotContains(t, string(specBytes), "(unknown)")

	// The var file scaffolds an unfilled (null) placeholder for exactly that
	// variable — null so that applying without filling it fails loudly.
	varFilePath := filepath.Join(testDir, "imported", importer.SecretsVarFileName)
	varFileBytes, err := os.ReadFile(varFilePath)
	require.NoError(t, err)
	varFileVars := map[string]any{}
	require.NoError(t, yaml.Unmarshal(varFileBytes, &varFileVars))
	assert.Equal(t, map[string]any{wantVar: nil}, varFileVars)

	// Applying with the unfilled placeholder is rejected by the var-file
	// resolver, so a forgotten secret can never silently apply as "".
	_, err = resolver.NewFileResolver(varFilePath)
	require.ErrorContains(t, err, wantVar)

	// The user fills in the real secret.
	require.NoError(t, os.WriteFile(varFilePath, []byte(wantVar+`: "filled-in-access-key"`+"\n"), 0600))

	workspace := &client.Workspace{ID: "test-workspace-id", Name: "Test Workspace"}

	// First apply links the imported resources; the secret is unknown remotely,
	// so the second apply re-sends it (the always-re-apply diff rule).
	for range 2 {
		applyProvider := example.NewProvider(b)
		fileResolver, err := resolver.NewFileResolver(varFilePath)
		require.NoError(t, err)

		proj := project.New(applyProvider, project.WithSubstitutor(varsubst.NewSubstitutor(fileResolver)))
		require.NoError(t, proj.Load(testDir))

		graph, err := proj.ResourceGraph()
		require.NoError(t, err)
		s, err := syncer.New(applyProvider, workspace)
		require.NoError(t, err)
		require.NoError(t, s.Sync(context.Background(), graph))
	}

	// The real value, injected via substitution, reached the backend.
	books := b.AllBooks()
	require.Len(t, books, 1)
	assert.Equal(t, "filled-in-access-key", books[0].AccessKey)
	assert.Equal(t, "the-hobbit", books[0].ExternalID)
}
