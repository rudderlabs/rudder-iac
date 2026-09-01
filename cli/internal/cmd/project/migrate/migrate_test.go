package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
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

func TestLoadRawMigrationProjectPreservesPlaceholders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`version: rudder/0.1
kind: source
metadata:
  name: test_source
spec:
  label: "prod-{{ .ENV }}"
  port: {{ .DB_PORT | 5432 }}
  hosts:
    - {{ .DB_HOST }}
`), 0644))

	proj, err := loadRawMigrationProject(dir)
	require.NoError(t, err)

	require.Contains(t, proj.Specs(), path)

	spec := proj.Specs()[path]
	assert.Equal(t, "prod-{{ .ENV }}", spec.Spec["label"])
	assert.Equal(t, "{{ .DB_PORT | 5432 }}", spec.Spec["port"])
	assert.Equal(t, []any{"{{ .DB_HOST }}"}, spec.Spec["hosts"])
}

func TestLoadRawMigrationProjectParsesFormatterOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.yaml")

	formatted, err := formatter.YAMLFormatter{}.Format(&specs.Spec{
		Version: specs.SpecVersionV0_1,
		Kind:    "source",
		Metadata: map[string]any{
			"name": "test_source",
		},
		Spec: map[string]any{
			"name": "{{ .API_TRACKING_NAME }}",
		},
	})
	require.NoError(t, err)
	require.Contains(t, string(formatted), "name: {{ .API_TRACKING_NAME }}")

	require.NoError(t, os.WriteFile(path, formatted, 0644))

	proj, err := loadRawMigrationProject(dir)
	require.NoError(t, err)

	require.Contains(t, proj.Specs(), path)
	assert.Equal(t, "{{ .API_TRACKING_NAME }}", proj.Specs()[path].Spec["name"])
}
