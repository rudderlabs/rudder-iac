package project

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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
