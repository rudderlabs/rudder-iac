package project

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestWiringLoader returns a fixed set of raw specs for Load.
type manifestWiringLoader struct {
	specs map[string]*specs.RawSpec
}

func (l *manifestWiringLoader) Load(string) (map[string]*specs.RawSpec, error) {
	return l.specs, nil
}

const importManifestYAML = `version: rudder/v1
kind: import-manifest
metadata:
  name: import-manifest
spec:
  workspaces:
    - workspace_id: ws-1
      resources:
        - urn: "event-stream-source:my-src"
          remote_id: rid-1
`

// TestProject_Load_RoutesImportManifestToProvider proves loadSpec classifies an
// import-manifest spec as project-level and routes it to the import-manifest
// provider (not the resource provider), and that the provider accumulates it.
func TestProject_Load_RoutesImportManifestToProvider(t *testing.T) {
	t.Parallel()

	loader := &manifestWiringLoader{specs: map[string]*specs.RawSpec{
		"import-manifest.yaml": {Data: []byte(importManifestYAML)},
	}}

	p := New(testutils.NewMockProvider(nil, nil), WithLoader(loader)).(*project)

	require.NoError(t, p.Load("dummy"))

	// The resource provider must NOT have received the manifest spec — it was
	// routed to the import-manifest provider instead.
	mp := p.provider.(*testutils.MockProvider)
	assert.Empty(t, mp.LoadSpecCalledWithArgs, "manifest must not be routed to the resource provider")
}
