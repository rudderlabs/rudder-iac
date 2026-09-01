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

func TestRestorePlaceholderScalars(t *testing.T) {
	t.Parallel()

	input := []byte(`spec:
  port: 5432
  enabled: true
  default_port: 5432
  name: "warehouse"
  description: "keep me"
`)

	got := restorePlaceholderScalars(input, []placeholderReplacement{
		{token: "{{ .DB_PORT }}", key: "port", scalarForms: []string{"5432", `"5432"`}},
		{token: "{{ .ENABLED }}", key: "enabled", scalarForms: []string{"true", `"true"`}},
		{token: "{{ .NAME }}", key: "name", scalarForms: []string{"warehouse", `"warehouse"`}},
	})

	assert.Equal(t, `spec:
  port: {{ .DB_PORT }}
  enabled: {{ .ENABLED }}
  default_port: 5432
  name: {{ .NAME }}
  description: "keep me"
`, string(got))
}
