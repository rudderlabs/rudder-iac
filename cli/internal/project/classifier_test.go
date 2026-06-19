package project

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     *specs.Spec
		expected SpecLevel
	}{
		{
			name:     "import-manifest is project-level",
			spec:     &specs.Spec{Kind: importmanifest.KindImportManifest, Version: specs.SpecVersionV1},
			expected: ProjectSpec,
		},
		{
			name:     "resource kind is resource-level",
			spec:     &specs.Spec{Kind: "source", Version: specs.SpecVersionV1},
			expected: ResourceSpec,
		},
		{
			name:     "empty kind is resource-level",
			spec:     &specs.Spec{Kind: "", Version: specs.SpecVersionV1},
			expected: ResourceSpec,
		},
		{
			name:     "legacy resource kind is resource-level",
			spec:     &specs.Spec{Kind: "destination", Version: specs.SpecVersionV0_1},
			expected: ResourceSpec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, classify(tt.spec))
		})
	}
}
