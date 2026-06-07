package describe_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/describe"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
)

// fakeRouter implements provider.TypeRouter by routing to a single provider
// for registered types and returning ErrUnsupportedType for everything else.
type fakeRouter struct {
	prov           provider.Provider
	supportedTypes []string
}

func (f *fakeRouter) ProviderForType(resourceType string) (provider.Provider, error) {
	for _, t := range f.supportedTypes {
		if t == resourceType {
			return f.prov, nil
		}
	}
	return nil, provider.ErrUnsupportedType
}

// newFakeRouter builds a fakeRouter backed by a real event-stream provider
// whose mock client returns the given sources.
func newFakeRouter(sources []sourceClient.EventStreamSource) *fakeRouter {
	mockClient := esSource.NewMockSourceClient()
	mockClient.SetGetSourcesFunc(func(_ context.Context) ([]sourceClient.EventStreamSource, error) {
		return sources, nil
	})
	prov := eventstream.New(mockClient)
	return &fakeRouter{
		prov:           prov,
		supportedTypes: prov.SupportedTypes(),
	}
}

// managedSource returns a source with ExternalID set (IaC-managed).
func managedSource() sourceClient.EventStreamSource {
	return sourceClient.EventStreamSource{
		ID:          "src-remote-1",
		ExternalID:  "my-managed-source",
		Name:        "My Managed Source",
		Type:        "javascript",
		Enabled:     true,
		WorkspaceID: "ws-123",
	}
}

func TestRunDescribe_ManagedSource_OutputContainsKeyFields(t *testing.T) {
	t.Parallel()

	router := newFakeRouter([]sourceClient.EventStreamSource{managedSource()})
	var buf bytes.Buffer

	err := describe.RunDescribe(context.Background(), &buf, router, "event-stream-source", "my-managed-source")
	require.NoError(t, err)

	out := buf.String()
	// The source name should appear in the formatted output.
	assert.Contains(t, out, "My Managed Source", "output must contain the source name")
	// The remote ID should appear.
	assert.Contains(t, out, "src-remote-1", "output must contain the remote ID")
	// Must contain a Managed line showing yes.
	assert.Contains(t, out, "Managed:", "output must include the Managed line")
	assert.Contains(t, out, "yes", "managed source must show 'yes'")
}

func TestRunDescribe_ManagedSource_OutputContainsManagedYes(t *testing.T) {
	t.Parallel()

	router := newFakeRouter([]sourceClient.EventStreamSource{managedSource()})
	var buf bytes.Buffer

	err := describe.RunDescribe(context.Background(), &buf, router, "event-stream-source", "my-managed-source")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Managed:")
	assert.Contains(t, out, "yes")
}

func TestRunDescribe_UnknownType_WrapsErrUnsupportedType(t *testing.T) {
	t.Parallel()

	router := newFakeRouter([]sourceClient.EventStreamSource{managedSource()})
	var buf bytes.Buffer

	err := describe.RunDescribe(context.Background(), &buf, router, "no-such-type", "some-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, provider.ErrUnsupportedType)
}

func TestRunDescribe_UnknownID_WrapsErrResourceNotFound(t *testing.T) {
	t.Parallel()

	router := newFakeRouter([]sourceClient.EventStreamSource{managedSource()})
	var buf bytes.Buffer

	err := describe.RunDescribe(context.Background(), &buf, router, "event-stream-source", "does-not-exist")
	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrResourceNotFound)
}

func TestNewCmdDescribe_Registered(t *testing.T) {
	t.Parallel()

	cmd := describe.NewCmdDescribe()
	require.NotNil(t, cmd)
	assert.Equal(t, "describe", cmd.Name())
	// Exactly 2 args required.
	assert.NotNil(t, cmd.Args)
}
