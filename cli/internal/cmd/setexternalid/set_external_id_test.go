package setexternalid_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/setexternalid"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
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

func TestRunSetExternalID_Success(t *testing.T) {
	t.Parallel()

	mockClient := esSource.NewMockSourceClient()
	prov := eventstream.New(mockClient)
	router := &fakeRouter{
		prov:           prov,
		supportedTypes: prov.SupportedTypes(),
	}

	var buf bytes.Buffer
	err := setexternalid.RunSetExternalID(context.Background(), &buf, router, "event-stream-source", "src_remote_1", "my-source")

	require.NoError(t, err)
	assert.True(t, mockClient.SetExternalIDCalled(), "SetExternalID must be called on the client")
	assert.Contains(t, buf.String(), "my-source", "output must mention the external id")
}

func TestRunSetExternalID_UnsupportedCapability(t *testing.T) {
	t.Parallel()

	// MockProvider does NOT implement provider.ExternalIDSetter
	mockProv := testutils.NewMockProvider(nil, []string{"some-type"})
	router := &fakeRouter{
		prov:           mockProv,
		supportedTypes: []string{"some-type"},
	}

	var buf bytes.Buffer
	err := setexternalid.RunSetExternalID(context.Background(), &buf, router, "some-type", "remote-id-1", "my-external-id")

	require.Error(t, err)
	assert.ErrorIs(t, err, resourceops.ErrVerbNotSupported)
}

func TestRunSetExternalID_UnknownType(t *testing.T) {
	t.Parallel()

	mockClient := esSource.NewMockSourceClient()
	prov := eventstream.New(mockClient)
	router := &fakeRouter{
		prov:           prov,
		supportedTypes: prov.SupportedTypes(),
	}

	var buf bytes.Buffer
	err := setexternalid.RunSetExternalID(context.Background(), &buf, router, "no-such-type", "remote-id-1", "my-external-id")

	require.Error(t, err)
	assert.ErrorIs(t, err, provider.ErrUnsupportedType)
}

func TestNewCmdSetExternalID_Registered(t *testing.T) {
	t.Parallel()

	cmd := setexternalid.NewCmdSetExternalID()
	require.NotNil(t, cmd)
	assert.Equal(t, "set-external-id", cmd.Name())
	// Exactly 3 args required — verify arity boundary cases.
	assert.Error(t, cmd.Args(cmd, []string{"a", "b"}), "two args must be rejected")
	assert.NoError(t, cmd.Args(cmd, []string{"a", "b", "c"}), "three args must be accepted")
	assert.Error(t, cmd.Args(cmd, []string{"a", "b", "c", "d"}), "four args must be rejected")
}
