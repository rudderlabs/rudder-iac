package migrate

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mapResolver map[string]string

func (m mapResolver) Resolve(name string) (string, bool) {
	value, ok := m[name]
	return value, ok
}

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
	raw := []byte(`version: rudder/0.1
kind: events
metadata:
  name: api_tracking
spec:
  name: "{{ .API_TRACKING_NAME }}"
  description: "prefix-{{ .API_TRACKING_DESCRIPTION }}"
  token: "{{ .API_TOKEN }}"
  unchanged: "literal"
`)

	sub := varsubst.NewSubstitutor(mapResolver{
		"API_TRACKING_NAME":        "API Tracking",
		"API_TRACKING_DESCRIPTION": "description from file",
		"API_TOKEN":                "secret-token",
	})
	replacements, err := collectPlaceholderReplacements(raw, sub)
	require.NoError(t, err)

	formatted := []byte(`version: "rudder/v1"
kind: "events"
metadata:
  name: "api_tracking"
spec:
  name: "API Tracking"
  description: "prefix-description from file"
  renamed_token: "secret-token"
  unchanged: "literal"
`)

	got := restorePlaceholderScalars(formatted, replacements)
	assert.Contains(t, string(got), `name: "{{ .API_TRACKING_NAME }}"`)
	assert.Contains(t, string(got), `description: "prefix-{{ .API_TRACKING_DESCRIPTION }}"`)
	assert.Contains(t, string(got), `renamed_token: "{{ .API_TOKEN }}"`)
	assert.Contains(t, string(got), `unchanged: "literal"`)
}
