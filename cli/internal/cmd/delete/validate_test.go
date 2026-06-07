package delete

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

// stubRouter is a minimal deleteRouter for validateType tests.
type stubRouter struct {
	types []string
}

func (s *stubRouter) SupportedTypes() []string { return s.types }

func (s *stubRouter) ProviderForType(_ string) (provider.Provider, error) { return nil, nil }

func TestValidateType_KnownType_ReturnsNil(t *testing.T) {
	t.Parallel()

	r := &stubRouter{types: []string{"event-stream-source", "event-stream-destination"}}

	require.NoError(t, validateType(r, "event-stream-source"))
	require.NoError(t, validateType(r, "event-stream-destination"))
}

func TestValidateType_UnknownType_ListsValidTypes(t *testing.T) {
	t.Parallel()

	r := &stubRouter{types: []string{"event-stream-source", "event-stream-destination"}}

	err := validateType(r, "bogus-type")
	require.Error(t, err)

	msg := err.Error()
	assert.Contains(t, msg, "bogus-type")
	assert.True(t, strings.Contains(msg, "event-stream-source") || strings.Contains(msg, "event-stream-destination"),
		"error should list at least one valid type, got: %s", msg)
}
