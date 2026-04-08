package project

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func TestHasLegacySpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		specMap  map[string]*specs.Spec
		wantTrue bool
	}{
		{
			name:     "nil map",
			specMap:  nil,
			wantTrue: false,
		},
		{
			name:     "empty map",
			specMap:  map[string]*specs.Spec{},
			wantTrue: false,
		},
		{
			name: "all v1",
			specMap: map[string]*specs.Spec{
				"a.yaml": {Version: specs.SpecVersionV1},
				"b.yaml": {Version: specs.SpecVersionV1},
			},
			wantTrue: false,
		},
		{
			name: "rudder/0.1 only",
			specMap: map[string]*specs.Spec{
				"a.yaml": {Version: specs.SpecVersionV0_1},
			},
			wantTrue: true,
		},
		{
			name: "rudder/v0.1 variant",
			specMap: map[string]*specs.Spec{
				"a.yaml": {Version: specs.SpecVersionV0_1Variant},
			},
			wantTrue: true,
		},
		{
			name: "mix v1 and v0.1",
			specMap: map[string]*specs.Spec{
				"a.yaml": {Version: specs.SpecVersionV1},
				"b.yaml": {Version: specs.SpecVersionV0_1},
			},
			wantTrue: true,
		},
		{
			name: "nil spec entry skipped",
			specMap: map[string]*specs.Spec{
				"a.yaml": nil,
				"b.yaml": {Version: specs.SpecVersionV1},
			},
			wantTrue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := HasLegacySpecs(tt.specMap)
			assert.Equal(t, tt.wantTrue, got)
		})
	}
}

func TestPrintLegacySpecDeprecationIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("no legacy specs prints nothing", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		ui.SetWriter(&buf)
		t.Cleanup(ui.RestoreWriter)

		PrintLegacySpecDeprecationIfNeeded(map[string]*specs.Spec{
			"a.yaml": {Version: specs.SpecVersionV1},
		})

		assert.Empty(t, buf.String())
	})

	t.Run("legacy specs prints deprecation", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		ui.SetWriter(&buf)
		t.Cleanup(ui.RestoreWriter)

		PrintLegacySpecDeprecationIfNeeded(map[string]*specs.Spec{
			"a.yaml": {Version: specs.SpecVersionV0_1},
		})

		out := buf.String()
		require.Contains(t, out, "Warning:")
		require.Contains(t, out, "v0.1 spec format")
		require.Contains(t, out, "rudder-cli migrate")
		require.Contains(t, out, "<migration-guide-url>")
	})
}
