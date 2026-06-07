package get_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/get"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

// fakeComposite implements get.Composite (the seam interface) with a single
// event-stream provider backed by a mock client.
type fakeComposite struct {
	prov           provider.Provider
	supportedTypes []string
}

func (f *fakeComposite) ProviderForType(resourceType string) (provider.Provider, error) {
	for _, t := range f.supportedTypes {
		if t == resourceType {
			return f.prov, nil
		}
	}
	return nil, provider.ErrUnsupportedType
}

func (f *fakeComposite) SupportedTypes() []string {
	return f.supportedTypes
}

// newFakeComposite builds a fakeComposite backed by a real event-stream provider
// whose mock client returns the given sources.
func newFakeComposite(sources []sourceClient.EventStreamSource) *fakeComposite {
	mockClient := esSource.NewMockSourceClient()
	mockClient.SetGetSourcesFunc(func(_ context.Context) ([]sourceClient.EventStreamSource, error) {
		return sources, nil
	})
	prov := eventstream.New(mockClient)
	return &fakeComposite{
		prov:           prov,
		supportedTypes: prov.SupportedTypes(),
	}
}

// testSources returns a consistent set of test sources: one managed, one unmanaged-ish.
func testSources() []sourceClient.EventStreamSource {
	return []sourceClient.EventStreamSource{
		{
			ID:          "src-remote-1",
			ExternalID:  "my-managed-source",
			Name:        "My Managed Source",
			Type:        "javascript",
			Enabled:     true,
			WorkspaceID: "ws-123",
		},
		// second source has no ExternalID — not yet managed
		{
			ID:          "src-remote-2",
			ExternalID:  "",
			Name:        "Unmanaged Source",
			Type:        "node",
			Enabled:     false,
			WorkspaceID: "ws-123",
		},
	}
}

func TestRunGet_List_ContainsManagedExternalID(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output: "table",
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "my-managed-source", "table must include the external ID")
	assert.Contains(t, out, "src-remote-1", "table must include the remote ID")
}

func TestRunGet_List_JSON_ContainsManagedSource(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output: "json",
	})
	require.NoError(t, err)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rows))
	require.NotEmpty(t, rows)

	// at least one row should have external_id = "my-managed-source"
	found := false
	for _, r := range rows {
		if r["external_id"] == "my-managed-source" {
			found = true
			break
		}
	}
	assert.True(t, found, "JSON list must contain the managed source")
}

func TestRunGet_List_ScopedManaged_ContainsManagedSource(t *testing.T) {
	t.Parallel()

	// Use only a managed source (ExternalID set). With ScopeManaged, this row
	// must appear; with ScopeUnmanaged it must not.
	managedOnly := []sourceClient.EventStreamSource{
		{
			ID:          "src-remote-1",
			ExternalID:  "my-managed-source",
			Name:        "My Managed Source",
			Type:        "javascript",
			Enabled:     true,
			WorkspaceID: "ws-123",
		},
	}
	cp := newFakeComposite(managedOnly)

	var buf bytes.Buffer
	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:  "table",
		Managed: true,
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "my-managed-source")
	assert.Contains(t, out, "src-remote-1")
}

func TestRunGet_Single_YAML_RoundTrips(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp,
		[]string{"event-stream-source", "my-managed-source"},
		get.GetOptions{Output: "yaml"},
	)
	require.NoError(t, err)

	out := buf.String()
	require.NotEmpty(t, out)

	spec, err := specs.New([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, esSource.ResourceKind, spec.Kind)
}

func TestRunGet_Single_JSON_HasKind(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp,
		[]string{"event-stream-source", "my-managed-source"},
		get.GetOptions{Output: "json"},
	)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))

	kind, ok := m["kind"].(string)
	require.True(t, ok, "expected 'kind' key in JSON output")
	assert.Equal(t, esSource.ResourceKind, kind)
}

func TestRunGet_Single_Table_ContainsRemoteID(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	// default output (table) for a single resource should show the row
	err := get.RunGet(context.Background(), &buf, cp,
		[]string{"event-stream-source", "my-managed-source"},
		get.GetOptions{Output: "table"},
	)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "src-remote-1")
}

func TestRunGet_UnknownType_ErrorMentionsValidTypes(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"nope"}, get.GetOptions{
		Output: "table",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event-stream-source",
		"error message should list valid types")
}

func TestRunGet_MutualExclusion_ManagedAndUnmanaged(t *testing.T) {
	t.Parallel()

	// We pass a nil composite — the mutual-exclusion check must fire before any
	// provider interaction so the composite is never called.
	err := get.RunGet(context.Background(), nil, nil, []string{"event-stream-source"}, get.GetOptions{
		Output:    "table",
		Managed:   true,
		Unmanaged: true,
	})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "mutually exclusive")
}

func TestNewCmdGet_Registered(t *testing.T) {
	t.Parallel()

	cmd := get.NewCmdGet()
	require.NotNil(t, cmd)
	assert.Equal(t, "get", cmd.Name())

	// Flags should be present
	assert.NotNil(t, cmd.Flags().Lookup("output"))
	assert.NotNil(t, cmd.Flags().Lookup("managed"))
	assert.NotNil(t, cmd.Flags().Lookup("unmanaged"))
	assert.NotNil(t, cmd.Flags().Lookup("selector"))
}
