package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
)

// secretRuleRegistry registers real definitions, so the tests follow the
// secret keys the CLI ships with.
func secretRuleRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	for _, definition := range []*definitions.DestinationDefinition{
		am.NewDefinition(),
		braze.NewDefinition(),
		httpdest.NewDefinition(),
		webhook.NewDefinition(),
	} {
		require.NoError(t, registry.Register(definition))
	}
	return registry
}

func TestShowsSecret(t *testing.T) {
	t.Parallel()

	// http's secret keys include headers.from, the header value.
	secretKeys := []string{"password", "headers.from", "s3.access_key"}

	cases := []struct {
		path string
		want bool
	}{
		{path: "/password", want: true},
		{path: "/headers/0/from", want: true},
		// The list item's line shows its first property, which may be from.
		{path: "/headers/0", want: true},
		// A missing key inside a list item prints the item's line.
		{path: "/headers/0/to", want: true},
		{path: "/s3/access_key", want: true},
		{path: "/s3", want: true},
		{path: "/s3/bucket", want: false},
		{path: "/headers", want: true},
		{path: "/api_url", want: false},
		{path: "/connection_mode/web", want: false},
		{path: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, showsSecret(secretKeys, tc.path))
		})
	}
}
