package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/spf13/viper"
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

// Not parallel: toggles the global experimental config used by secret.WithVariableName.
func TestBaseHandler_SecretFieldsScrubStateAndExportVariables(t *testing.T) {
	enableVarSubstitution(t)

	h := newSecretFixtureHandler()
	remote := &secretFixtureRemote{
		ID:          "remote-secret-1",
		ExternalID:  "alpha-beta",
		WorkspaceID: "workspace-1",
		Name:        "Alpha Beta",
		APIKey:      "literal-remote-secret",
	}
	collection := resources.NewRemoteResources()
	collection.Set(secretFixtureMetadata.ResourceType, map[string]*resources.RemoteResource{
		remote.ID: {
			ID:         remote.ID,
			ExternalID: remote.ExternalID,
			Data:       remote,
		},
	})

	mappedState, err := h.MapRemoteToState(collection)
	require.NoError(t, err)
	resourceState := mappedState.GetResource(resources.URN(remote.ExternalID, secretFixtureMetadata.ResourceType))
	require.NotNil(t, resourceState)

	inputRaw, ok := resourceState.InputRaw.(*secretFixtureResource)
	require.True(t, ok, "expected *secretFixtureResource, got %T", resourceState.InputRaw)
	require.NotNil(t, inputRaw.APIKey)
	assert.True(t, inputRaw.APIKey.IsUnknown())
	assert.Empty(t, inputRaw.APIKey.Reveal())

	outputRaw, ok := resourceState.OutputRaw.(*secretFixtureState)
	require.True(t, ok, "expected *secretFixtureState, got %T", resourceState.OutputRaw)
	assert.True(t, outputRaw.APIKey.IsUnknown())
	assert.Empty(t, outputRaw.APIKey.Reveal())

	entities, entries, err := h.FormatForExport(collection, namer.NewExternalIdNamer(namer.StrategyKebabCase), nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	require.Len(t, entries, 1)

	exported, err := json.Marshal(entities[0].Content)
	require.NoError(t, err)
	exportedContent := string(exported)
	assert.Contains(t, exportedContent, "{{ .SECRET_FIXTURE_ALPHA_BETA_API_KEY }}")
	assert.NotContains(t, exportedContent, remote.APIKey)
}

func enableVarSubstitution(t *testing.T) {
	t.Helper()

	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

var secretFixtureMetadata = handler.HandlerMetadata{
	ResourceType:     "secret-fixture",
	SpecKind:         "secret-fixtures",
	SpecMetadataName: "secretFixtures",
}

type secretFixtureSpec struct {
	Items []secretFixtureItem `json:"items"`
}

type secretFixtureItem struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	APIKey secret.String `json:"apiKey"`
}

type secretFixtureResource struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	APIKey *secret.String `json:"apiKey" secret:"true"`
}

type secretFixtureState struct {
	RemoteID string        `json:"remoteId"`
	APIKey   secret.String `json:"apiKey" secret:"true"`
}

type secretFixtureRemote struct {
	ID          string
	ExternalID  string
	WorkspaceID string
	Name        string
	APIKey      string
}

func (r secretFixtureRemote) Metadata() handler.RemoteResourceMetadata {
	return handler.RemoteResourceMetadata{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		WorkspaceID: r.WorkspaceID,
		Name:        r.Name,
	}
}

type secretFixtureImpl struct {
	*export.SingleSpecExportStrategy[secretFixtureSpec, secretFixtureRemote]
}

func newSecretFixtureHandler() *handler.BaseHandler[secretFixtureSpec, secretFixtureResource, secretFixtureState, secretFixtureRemote] {
	impl := &secretFixtureImpl{}
	impl.SingleSpecExportStrategy = &export.SingleSpecExportStrategy[secretFixtureSpec, secretFixtureRemote]{Handler: impl}
	return handler.NewHandler[secretFixtureSpec, secretFixtureResource, secretFixtureState, secretFixtureRemote](impl)
}

func (h *secretFixtureImpl) Metadata() handler.HandlerMetadata {
	return secretFixtureMetadata
}

func (h *secretFixtureImpl) NewSpec() *secretFixtureSpec {
	return &secretFixtureSpec{}
}

func (h *secretFixtureImpl) ExtractResourcesFromSpec(_ string, spec *secretFixtureSpec) (map[string]*secretFixtureResource, error) {
	resourcesByID := make(map[string]*secretFixtureResource, len(spec.Items))
	for _, item := range spec.Items {
		resourcesByID[item.ID] = &secretFixtureResource{
			ID:     item.ID,
			Name:   item.Name,
			APIKey: &item.APIKey,
		}
	}
	return resourcesByID, nil
}

func (h *secretFixtureImpl) LoadRemoteResources(context.Context) ([]*secretFixtureRemote, error) {
	return nil, nil
}

func (h *secretFixtureImpl) LoadImportableResources(context.Context) ([]*secretFixtureRemote, error) {
	return nil, nil
}

func (h *secretFixtureImpl) MapRemoteToState(remote *secretFixtureRemote, _ handler.URNResolver) (*secretFixtureResource, *secretFixtureState, error) {
	inputSecret := secret.New(remote.APIKey)
	return &secretFixtureResource{
			ID:     remote.ExternalID,
			Name:   remote.Name,
			APIKey: &inputSecret,
		}, &secretFixtureState{
			RemoteID: remote.ID,
			APIKey:   secret.New(remote.APIKey),
		}, nil
}

func (h *secretFixtureImpl) Create(context.Context, *secretFixtureResource) (*secretFixtureState, error) {
	return nil, nil
}

func (h *secretFixtureImpl) Update(context.Context, *secretFixtureResource, *secretFixtureResource, *secretFixtureState) (*secretFixtureState, error) {
	return nil, nil
}

func (h *secretFixtureImpl) Import(context.Context, *secretFixtureResource, string) (*secretFixtureState, error) {
	return nil, nil
}

func (h *secretFixtureImpl) Delete(context.Context, string, *secretFixtureResource, *secretFixtureState) error {
	return nil
}

func (h *secretFixtureImpl) MapRemoteToSpec(data map[string]*secretFixtureRemote, _ resolver.ReferenceResolver) (*export.SpecExportData[secretFixtureSpec], error) {
	items := make([]secretFixtureItem, 0, len(data))
	for externalID, remote := range data {
		items = append(items, secretFixtureItem{
			ID:     externalID,
			Name:   remote.Name,
			APIKey: secret.New(remote.APIKey),
		})
	}
	return &export.SpecExportData[secretFixtureSpec]{
		RelativePath: "secret-fixtures.yaml",
		Data:         &secretFixtureSpec{Items: items},
	}, nil
}
