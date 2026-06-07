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
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// fakeComposite implements get.Composite (the seam interface) with a single
// event-stream provider backed by a mock client.
type fakeComposite struct {
	prov           provider.Provider
	supportedTypes []string
}

func (f *fakeComposite) ProviderForType(resourceType string) (get.GetProvider, error) {
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

// ---------------------------------------------------------------------------
// Selector filter tests
// ---------------------------------------------------------------------------

func TestRunGet_Selector_ExternalID_Table(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:   "table",
		Selector: map[string]string{"external-id": "my-managed-source"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "my-managed-source", "selector must keep the matching row")
	// The unmanaged source has no external-id so it must be absent.
	assert.NotContains(t, out, "src-remote-2", "selector must exclude non-matching rows")
}

func TestRunGet_Selector_ExternalID_JSON(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:   "json",
		Selector: map[string]string{"external-id": "my-managed-source"},
	})
	require.NoError(t, err)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rows))
	require.Len(t, rows, 1, "exactly one row must survive the selector filter")
	assert.Equal(t, "my-managed-source", rows[0]["external_id"])
}

// newMixedComposite creates a composite backed by a managedOnlyProv with both
// a managed and an unmanaged row so selector tests for managed=true/false work
// with predictable Managed fields (unaffected by the event-stream LoadImportable
// namer which would assign ExternalIDs to unmanaged rows).
func newMixedComposite() *managedOnlyComposite {
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src-m": {ID: "src-m", ExternalID: "ext-managed", Data: map[string]any{"name": "Managed Source"}},
		"src-u": {ID: "src-u", ExternalID: "", Data: map[string]any{"name": "Unmanaged Source"}},
	})
	return &managedOnlyComposite{
		prov:           &managedOnlyProv{rc: rc},
		supportedTypes: []string{"event-stream-source"},
	}
}

func TestRunGet_Selector_Managed_True(t *testing.T) {
	t.Parallel()

	cp := newMixedComposite()
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:   "table",
		Selector: map[string]string{"managed": "true"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "ext-managed", "managed=true must keep the managed row")
	assert.NotContains(t, out, "src-u", "managed=true must exclude the unmanaged row")
}

func TestRunGet_Selector_Managed_False(t *testing.T) {
	t.Parallel()

	cp := newMixedComposite()
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:   "table",
		Selector: map[string]string{"managed": "false"},
	})
	require.NoError(t, err)

	out := buf.String()
	assert.NotContains(t, out, "ext-managed", "managed=false must exclude the managed row")
	assert.Contains(t, out, "src-u", "managed=false must keep the unmanaged row")
}

func TestRunGet_Selector_UnknownKey_Error(t *testing.T) {
	t.Parallel()

	cp := newFakeComposite(testSources())
	var buf bytes.Buffer

	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output:   "table",
		Selector: map[string]string{"bogus": "x"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus", "error must name the offending key")
	assert.Contains(t, err.Error(), "external-id", "error must list supported keys")
}

// ---------------------------------------------------------------------------
// Degraded-note test
// ---------------------------------------------------------------------------

// managedOnlyProv implements only provider.ManagedRemoteResourceLoader (= get.GetProvider).
// It does NOT implement provider.UnmanagedRemoteResourceLoader, so
// resourceops.SupportsUnmanaged returns false and RunGet prints the degraded note.
type managedOnlyProv struct {
	rc *resources.RemoteResources
}

func (p *managedOnlyProv) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return p.rc, nil
}

// managedOnlyComposite is a fakeComposite variant that returns a managedOnlyProv
// so that the degraded-mode note path in RunGet can be exercised.
type managedOnlyComposite struct {
	prov           *managedOnlyProv
	supportedTypes []string
}

func (c *managedOnlyComposite) ProviderForType(resourceType string) (get.GetProvider, error) {
	for _, t := range c.supportedTypes {
		if t == resourceType {
			return c.prov, nil
		}
	}
	return nil, provider.ErrUnsupportedType
}

func (c *managedOnlyComposite) SupportedTypes() []string { return c.supportedTypes }

func TestRunGet_DegradedNote_ManagedOnlyProvider(t *testing.T) {
	t.Parallel()

	// Build a RemoteResources with one managed source so there is something to list.
	rc := resources.NewRemoteResources()
	rc.Set("event-stream-source", map[string]*resources.RemoteResource{
		"src-1": {ID: "src-1", ExternalID: "managed-src", Data: map[string]any{"name": "Managed Source"}},
	})

	cp := &managedOnlyComposite{
		prov:           &managedOnlyProv{rc: rc},
		supportedTypes: []string{"event-stream-source"},
	}

	var buf bytes.Buffer
	// Default scope (ScopeAll) triggers the unmanaged check.
	err := get.RunGet(context.Background(), &buf, cp, []string{"event-stream-source"}, get.GetOptions{
		Output: "table",
	})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "note:", "degraded note must appear when provider cannot enumerate unmanaged resources")
	assert.Contains(t, out, "unmanaged", "note must mention unmanaged resources")
	// The managed row must still be present.
	assert.Contains(t, out, "managed-src")
}
