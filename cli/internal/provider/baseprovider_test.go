package provider

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureHandler is a no-op Handler that records the import manifest it receives.
type captureHandler struct {
	resourceType string
	got          *specs.WorkspacesImportMetadata
}

func (h *captureHandler) ResourceType() string              { return h.resourceType }
func (h *captureHandler) SpecKind() string                  { return h.resourceType }
func (h *captureHandler) LoadSpec(string, *specs.Spec) error { return nil }
func (h *captureHandler) LoadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	h.got = m
	return nil
}
func (h *captureHandler) ParseSpec(string, *specs.Spec) (*specs.ParsedSpec, error) {
	return &specs.ParsedSpec{}, nil
}
func (h *captureHandler) Resources() ([]*resources.Resource, error)              { return nil, nil }
func (h *captureHandler) Create(context.Context, any) (any, error)               { return nil, nil }
func (h *captureHandler) Update(context.Context, any, any, any) (any, error)     { return nil, nil }
func (h *captureHandler) Delete(context.Context, string, any, any) error         { return nil }
func (h *captureHandler) Import(context.Context, any, string) (any, error)       { return nil, nil }
func (h *captureHandler) LoadResourcesFromRemote(context.Context) (*resources.RemoteResources, error) {
	return nil, nil
}
func (h *captureHandler) MapRemoteToState(*resources.RemoteResources) (*state.State, error) {
	return nil, nil
}
func (h *captureHandler) LoadImportable(context.Context, namer.Namer) (*resources.RemoteResources, error) {
	return nil, nil
}
func (h *captureHandler) FormatForExport(*resources.RemoteResources, namer.Namer, resolver.ReferenceResolver) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	return nil, nil, nil
}

func TestBaseProvider_LoadImportManifest(t *testing.T) {
	t.Run("fans the active workspace out to every handler", func(t *testing.T) {
		h1 := &captureHandler{resourceType: "a"}
		h2 := &captureHandler{resourceType: "b"}
		p := NewBaseProvider([]Handler{h1, h2})
		ws := &specs.WorkspaceImportMetadata{
			WorkspaceID: "ws-a",
			Resources:   []specs.ImportIds{{URN: "a:1", RemoteID: "r1"}},
		}
		require.NoError(t, p.LoadImportManifest(ws))

		// Handlers receive the workspace wrapped in the shared plural type.
		want := &specs.WorkspacesImportMetadata{Workspaces: []specs.WorkspaceImportMetadata{*ws}}
		assert.Equal(t, want, h1.got)
		assert.Equal(t, want, h2.got)
	})
}
