package renderer

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextRenderer_Render(t *testing.T) {
	tests := []struct {
		name        string
		diagnostics validation.Diagnostics
		expected    string
	}{
		{
			name:        "empty diagnostics produces no output",
			diagnostics: validation.Diagnostics{},
			expected:    "",
		},
		{
			name: "single error with line text",
			diagnostics: validation.Diagnostics{
				{
					RuleID:   "project/version-valid",
					Severity: rules.Error,
					Message:  "version must be one of the supported versions",
					File:     "specs/malformed.yaml",
					Position: pathindex.Position{
						Line:     1,
						Column:   1,
						LineText: "version: rudder/v1.1",
					},
				},
			},
			expected: `
error[project/version-valid]: version must be one of the supported versions
  --> specs/malformed.yaml:1:1
     |
   1 | version: rudder/v1.1
     | ^^^^^^^^^^^^^^^^^^^^

Found 1 error(s), 0 warning(s)
`,
		},
		{
			name: "single warning",
			diagnostics: validation.Diagnostics{
				{
					RuleID:   "datacatalog/properties/deprecated",
					Severity: rules.Warning,
					Message:  "property 'user_id' is deprecated",
					File:     "specs/events.yaml",
					Position: pathindex.Position{
						Line:     15,
						Column:   3,
						LineText: "user_id: string",
					},
				},
			},
			expected: `
warning[datacatalog/properties/deprecated]: property 'user_id' is deprecated
  --> specs/events.yaml:15:3
      |
   15 | user_id: string
      | ^^^^^^^^^^^^^^^

Found 0 error(s), 1 warning(s)
`,
		},
		{
			name: "mixed errors and warnings",
			diagnostics: validation.Diagnostics{
				{
					RuleID:   "project/version-valid",
					Severity: rules.Error,
					Message:  "invalid version",
					File:     "specs/project.yaml",
					Position: pathindex.Position{
						Line:     1,
						Column:   1,
						LineText: "version: invalid",
					},
				},
				{
					RuleID:   "datacatalog/naming",
					Severity: rules.Warning,
					Message:  "name should be lowercase",
					File:     "specs/events.yaml",
					Position: pathindex.Position{
						Line:     5,
						Column:   3,
						LineText: "name: UserCreated",
					},
				},
				{
					RuleID:   "datacatalog/required",
					Severity: rules.Error,
					Message:  "missing required field 'id'",
					File:     "specs/events.yaml",
					Position: pathindex.Position{
						Line:     10,
						Column:   1,
						LineText: "spec: {...}",
					},
				},
			},
			expected: `
error[project/version-valid]: invalid version
  --> specs/project.yaml:1:1
     |
   1 | version: invalid
     | ^^^^^^^^^^^^^^^^

warning[datacatalog/naming]: name should be lowercase
  --> specs/events.yaml:5:3
     |
   5 | name: UserCreated
     | ^^^^^^^^^^^^^^^^^

error[datacatalog/required]: missing required field 'id'
  --> specs/events.yaml:10:1
      |
   10 | spec: {...}
      | ^^^^^^^^^^^

Found 2 error(s), 1 warning(s)
`,
		},
		{
			name: "info severity does not count in summary",
			diagnostics: validation.Diagnostics{
				{
					RuleID:   "datacatalog/hint",
					Severity: rules.Info,
					Message:  "consider adding a description",
					File:     "specs/events.yaml",
					Position: pathindex.Position{
						Line:     3,
						Column:   1,
						LineText: "name: Test",
					},
				},
			},
			expected: `
info[datacatalog/hint]: consider adding a description
  --> specs/events.yaml:3:1
     |
   3 | name: Test
     | ^^^^^^^^^^
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderer := NewTextRenderer(&buf)

			err := renderer.Render(tt.diagnostics)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestTextRenderer_RenderWithoutLineText(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewTextRenderer(&buf)

	diagnostics := validation.Diagnostics{
		{
			RuleID:   "project/parse-error",
			Severity: rules.Error,
			Message:  "failed to parse YAML",
			File:     "specs/broken.yaml",
			Position: pathindex.Position{
				Line:   1,
				Column: 1,
			},
		},
	}

	err := renderer.Render(diagnostics)

	require.NoError(t, err)
	expected := `
error[project/parse-error]: failed to parse YAML
  --> specs/broken.yaml:1:1

Found 1 error(s), 0 warning(s)
`
	assert.Equal(t, expected, buf.String())
}

func TestTextRenderer_MultiDigitLineNumbers(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewTextRenderer(&buf)

	diagnostics := validation.Diagnostics{
		{
			RuleID:   "datacatalog/property-type",
			Severity: rules.Error,
			Message:  "invalid property type",
			File:     "specs/large-file.yaml",
			Position: pathindex.Position{
				Line:     123,
				Column:   5,
				LineText: "type: invalid",
			},
		},
	}

	err := renderer.Render(diagnostics)

	require.NoError(t, err)
	expected := `
error[datacatalog/property-type]: invalid property type
  --> specs/large-file.yaml:123:5
       |
   123 | type: invalid
       | ^^^^^^^^^^^^^

Found 1 error(s), 0 warning(s)
`
	assert.Equal(t, expected, buf.String())
}
