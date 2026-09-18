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

func TestJSONRenderer_Render(t *testing.T) {
	tests := []struct {
		name        string
		diagnostics validation.Diagnostics
		expected    string
	}{
		{
			// A clean run must still emit a document — a consumer cannot tell
			// "validated clean" from "crashed before rendering" out of silence.
			name:        "empty diagnostics still produce a document",
			diagnostics: validation.Diagnostics{},
			expected: `{
  "diagnostics": [],
  "summary": {
    "errors": 0,
    "warnings": 0
  }
}
`,
		},
		{
			name: "mixed severities are counted and emitted in order",
			diagnostics: validation.Diagnostics{
				{
					RuleID:   "project/version-valid",
					Severity: rules.Error,
					Message:  "version must be one of the supported versions",
					File:     "specs/malformed.yaml",
					Position: pathindex.Position{Line: 1, Column: 1, LineText: "version: rudder/v1.1"},
				},
				{
					RuleID:   "datacatalog/properties/deprecated",
					Severity: rules.Warning,
					Message:  "property 'user_id' is deprecated",
					File:     "specs/events.yaml",
					Position: pathindex.Position{Line: 15, Column: 3},
				},
			},
			expected: `{
  "diagnostics": [
    {
      "ruleId": "project/version-valid",
      "severity": "error",
      "message": "version must be one of the supported versions",
      "file": "specs/malformed.yaml",
      "line": 1,
      "column": 1
    },
    {
      "ruleId": "datacatalog/properties/deprecated",
      "severity": "warning",
      "message": "property 'user_id' is deprecated",
      "file": "specs/events.yaml",
      "line": 15,
      "column": 3
    }
  ],
  "summary": {
    "errors": 1,
    "warnings": 1
  }
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			require.NoError(t, NewJSONRenderer(&buf).Render(tt.diagnostics))
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
