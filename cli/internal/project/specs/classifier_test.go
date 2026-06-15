package specs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     *Spec
		expected SpecLevel
	}{
		{
			name:     "import-manifest is project-level",
			spec:     &Spec{Kind: KindImportManifest, Version: SpecVersionV1},
			expected: ProjectSpec,
		},
		{
			name:     "resource kind is resource-level",
			spec:     &Spec{Kind: "source", Version: SpecVersionV1},
			expected: ResourceSpec,
		},
		{
			name:     "empty kind is resource-level",
			spec:     &Spec{Kind: "", Version: SpecVersionV1},
			expected: ResourceSpec,
		},
		{
			name:     "legacy resource kind is resource-level",
			spec:     &Spec{Kind: "destination", Version: SpecVersionV0_1},
			expected: ResourceSpec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, Classify(tt.spec))
		})
	}
}
