package renderer

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONRenderer_Render(t *testing.T) {
	t.Run("empty diagnostics renders empty array", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewJSONRenderer(&buf)

		require.NoError(t, r.Render(validation.Diagnostics{}))
		assert.JSONEq(t, `{"diagnostics":[]}`, buf.String())
	})

	t.Run("renders diagnostics with code, severity, positions", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewJSONRenderer(&buf)

		diags := validation.Diagnostics{
			{
				RuleID:   "datacatalog/properties/spec-syntax-valid",
				Severity: rules.Error,
				Message:  "missing required field 'id'",
				File:     "specs/props.yaml",
				Kind:     "properties",
				Position: pathindex.Position{Line: 4, Column: 7, LineText: "name: Foo"},
			},
			{
				RuleID:   "datacatalog/properties/deprecated",
				Severity: rules.Warning,
				Message:  "property is deprecated",
				File:     "specs/props.yaml",
				Kind:     "properties",
				Position: pathindex.Position{Line: 10, Column: 3},
			},
		}

		require.NoError(t, r.Render(diags))

		var out struct {
			Diagnostics []map[string]any `json:"diagnostics"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
		require.Len(t, out.Diagnostics, 2)

		first := out.Diagnostics[0]
		assert.Equal(t, "datacatalog/properties/spec-syntax-valid", first["code"])
		assert.Equal(t, "error", first["severity"])
		assert.Equal(t, "missing required field 'id'", first["message"])
		assert.Equal(t, "properties", first["kind"])
		assert.Equal(t, "specs/props.yaml", first["file"])
		assert.Equal(t, float64(4), first["line"])
		assert.Equal(t, float64(7), first["col"])

		second := out.Diagnostics[1]
		assert.Equal(t, "warning", second["severity"])
		assert.Equal(t, float64(10), second["line"])
	})
}
