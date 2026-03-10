package results

import (
	"testing"

	"github.com/stretchr/testify/assert"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

func TestTestResults_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		results  *TestResults
		expected bool
	}{
		{
			name: "empty results - no libraries or transformations",
			results: &TestResults{
				Libraries:       []transformations.LibraryTestResult{},
				Transformations: []*TransformationTestWithDefinitions{},
			},
			expected: true,
		},
		{
			name: "has libraries only",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{
					{HandleName: "lib1", Pass: true},
				},
				Transformations: []*TransformationTestWithDefinitions{},
			},
			expected: false,
		},
		{
			name: "has transformations only",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{},
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							Name: "tr1",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "has both libraries and transformations",
			results: &TestResults{
				Libraries: []transformations.LibraryTestResult{
					{HandleName: "lib1", Pass: true},
				},
				Transformations: []*TransformationTestWithDefinitions{
					{
						Result: &transformations.TransformationTestResult{
							Name: "tr1",
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.results.IsEmpty())
		})
	}
}
