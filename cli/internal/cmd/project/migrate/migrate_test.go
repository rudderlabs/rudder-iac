package migrate

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateTelemetryExtras(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		confirm  bool
		want     map[string]any
	}{
		{
			name:     "default location and confirm true",
			location: ".",
			confirm:  true,
			want: map[string]any{
				"location":     ".",
				"confirm":      true,
				"from_version": specs.SpecVersionV0_1,
				"to_version":   specs.SpecVersionV1,
			},
		},
		{
			name:     "custom path and confirm false",
			location: "/tmp/project",
			confirm:  false,
			want: map[string]any{
				"location":     "/tmp/project",
				"confirm":      false,
				"from_version": specs.SpecVersionV0_1,
				"to_version":   specs.SpecVersionV1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := migrateTelemetryExtras(tt.location, tt.confirm)
			require.Len(t, got, len(tt.want))

			byKey := make(map[string]any, len(got))
			for _, kv := range got {
				byKey[kv.K] = kv.V
			}
			assert.Equal(t, tt.want, byKey)
		})
	}
}
