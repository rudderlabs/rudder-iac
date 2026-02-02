package handler_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseHandler_LoadSpec_ImportMetadata(t *testing.T) {
	t.Parallel()

	t.Run("accepts URN-based import metadata", func(t *testing.T) {
		b := backend.NewBackend()
		provider := example.NewProvider(b)

		proj := project.New(provider, project.WithLoader(&mockLoader{specs: map[string]string{
			"writer/tolkien.yaml": `version: rudder/v0.1
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
`,
		}}))

		err := proj.Load("dummy/path")
		require.NoError(t, err, "URN-based import metadata should be accepted")
	})

	t.Run("rejects local_id-only import metadata", func(t *testing.T) {
		b := backend.NewBackend()
		provider := example.NewProvider(b)

		proj := project.New(provider, project.WithLoader(&mockLoader{specs: map[string]string{
			"writer/tolkien.yaml": `version: rudder/v0.1
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
`,
		}}))

		err := proj.Load("dummy/path")
		require.Error(t, err, "local_id-only import metadata should be rejected")
		assert.Contains(t, err.Error(), "urn field is required")
		assert.Contains(t, err.Error(), "local_id is not supported")
	})

	t.Run("rejects empty URN and local_id", func(t *testing.T) {
		b := backend.NewBackend()
		provider := example.NewProvider(b)

		proj := project.New(provider, project.WithLoader(&mockLoader{specs: map[string]string{
			"writer/tolkien.yaml": `version: rudder/v0.1
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
`,
		}}))

		err := proj.Load("dummy/path")
		require.Error(t, err, "empty URN and local_id should be rejected")
		assert.Contains(t, err.Error(), "either urn or local_id must be set")
	})

	t.Run("rejects both URN and local_id set", func(t *testing.T) {
		b := backend.NewBackend()
		provider := example.NewProvider(b)

		proj := project.New(provider, project.WithLoader(&mockLoader{specs: map[string]string{
			"writer/tolkien.yaml": `version: rudder/v0.1
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
`,
		}}))

		err := proj.Load("dummy/path")
		require.Error(t, err, "both URN and local_id set should be rejected")
		assert.Contains(t, err.Error(), "mutually exclusive")
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
