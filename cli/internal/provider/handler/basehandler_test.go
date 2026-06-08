package handler_test

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadImportMetadataSpec runs a single-spec project through the full project
// pipeline (the same path the apply/validate commands use) and returns the
// load error plus the captured renderer output.
//
// Import-metadata validation is enforced by the project-level gatekeeper rule
// project/metadata-syntax-valid. It runs in the syntax phase and renders its
// diagnostics; Load itself returns a generic "syntax validation failed", so
// assertions target the rendered output rather than err.Error(). The example
// provider declares no SupportedMatchPatterns, but the gatekeeper rule matches
// every kind (MatchAll), so it governs import metadata here exactly as it does
// for real providers — the handler-phase ImportIds validation is shadowed by it.
func loadImportMetadataSpec(t *testing.T, spec string) (error, string) {
	t.Helper()

	var buf bytes.Buffer
	provider := example.NewProvider(backend.NewBackend())
	proj := project.New(provider,
		project.WithLoader(&mockLoader{specs: map[string]string{"writer/tolkien.yaml": spec}}),
		project.WithRenderer(renderer.NewTextRenderer(&buf)),
	)

	return proj.Load("dummy/path"), buf.String()
}

func TestBaseHandler_LoadSpec_ImportMetadata(t *testing.T) {
	t.Parallel()

	t.Run("accepts URN-based import metadata", func(t *testing.T) {
		err, _ := loadImportMetadataSpec(t, `version: rudder/v0.1
kind: writer
metadata:
  name: common
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - urn: "example-writer:tolkien"
            remote_id: "remote-writer-tolkien"
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)
		require.NoError(t, err, "URN-based import metadata should be accepted")
	})

	t.Run("rejects local_id-only import metadata", func(t *testing.T) {
		err, out := loadImportMetadataSpec(t, `version: rudder/v0.1
kind: writer
metadata:
  name: common
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - local_id: tolkien
            remote_id: "remote-writer-tolkien"
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)
		require.Error(t, err, "local_id-only import metadata should be rejected")
		assert.Contains(t, out, "project/metadata-syntax-valid")
		assert.Contains(t, out, "local_id")
		assert.Contains(t, out, "not supported")
	})

	t.Run("rejects empty URN and local_id", func(t *testing.T) {
		err, out := loadImportMetadataSpec(t, `version: rudder/v0.1
kind: writer
metadata:
  name: common
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - remote_id: "remote-writer-tolkien"
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)
		require.Error(t, err, "empty URN and local_id should be rejected")
		assert.Contains(t, out, "project/metadata-syntax-valid")
		assert.Contains(t, out, "is required when")
	})

	t.Run("rejects both URN and local_id set", func(t *testing.T) {
		err, out := loadImportMetadataSpec(t, `version: rudder/v0.1
kind: writer
metadata:
  name: common
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - urn: "example-writer:tolkien"
            local_id: tolkien
            remote_id: "remote-writer-tolkien"
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)
		require.Error(t, err, "both URN and local_id set should be rejected")
		assert.Contains(t, out, "project/metadata-syntax-valid")
		assert.Contains(t, out, "cannot be specified together")
	})
}

type mockLoader struct {
	specs map[string]string
}

func (m *mockLoader) Load(_ string) (map[string]*specs.RawSpec, error) {
	s := make(map[string]*specs.RawSpec, len(m.specs))
	for p, specStr := range m.specs {
		s[p] = &specs.RawSpec{Data: []byte(specStr)}
	}
	return s, nil
}
