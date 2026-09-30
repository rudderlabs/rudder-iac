package eventstream

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	connection "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecSchemasCoverSupportedKinds(t *testing.T) {
	t.Parallel()

	p := New(source.NewMockSourceClient())

	assert.ElementsMatch(t, []string{source.ResourceKind, connection.EventStreamConnectionResourceKind}, p.SupportedKinds())
	assert.ElementsMatch(t, p.SupportedKinds(), schema.Kinds(p.SpecSchemas()))
}

// Connection rules are registered unconditionally now that connection support
// is GA, so NewConnectionSemanticValidRule always captures destinationRegistry.
// It constructs fine with a nil registry and only panics later, inside
// registry.Get during validate. Pin both halves of the guard that keeps that
// unreachable through the constructor: New defaults the field, and
// WithDestinationRegistry refuses to clear it.
func TestNewAlwaysHasDestinationRegistry(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()

	cases := []struct {
		name string
		opts []Option
		want *definitions.Registry
	}{
		{name: "no options uses the default", opts: nil},
		{name: "nil registry is ignored", opts: []Option{WithDestinationRegistry(nil)}},
		{name: "supplied registry wins", opts: []Option{WithDestinationRegistry(registry)}, want: registry},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := New(source.NewMockSourceClient(), tc.opts...)

			require.NotNil(t, p.destinationRegistry)
			if tc.want != nil {
				assert.Same(t, tc.want, p.destinationRegistry)
			}
		})
	}
}
