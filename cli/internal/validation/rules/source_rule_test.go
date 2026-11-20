package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

func TestNewSourceContext(t *testing.T) {
	t.Parallel()

	t.Run("spec is parsed once with placeholders", func(t *testing.T) {
		t.Parallel()

		// "/hook" after the unquoted reference is not YAML until substitution.
		source := []byte(`version: rudder/v1
kind: destination
metadata:
  name: shop-webhook
spec:
  id: shop-webhook
  config:
    webhook_url: {{ .HOOK_BASE }}/hook
    api_key: '{{ .API_KEY }}'
`)

		ctx := NewSourceContext("specs/webhook.yaml", &specs.RawSpec{Data: source})

		references := varsubst.ReplaceReferences(source)
		assert.Equal(t, &ValidationContext{
			FilePath: "specs/webhook.yaml",
			FileName: "webhook.yaml",
			Kind:     "destination",
			Version:  "rudder/v1",
			Metadata: map[string]any{"name": "shop-webhook"},
			Spec: map[string]any{
				"id": "shop-webhook",
				"config": map[string]any{
					"webhook_url": "__rudder_variable_reference_0__/hook",
					"api_key":     "__rudder_variable_reference_1__",
				},
			},
			Source:     source,
			References: references,
		}, ctx)

		ref, ok := ctx.References.Reference("__rudder_variable_reference_1__")
		assert.True(t, ok)
		assert.Equal(t, varsubst.Reference{Name: "API_KEY", Text: "{{ .API_KEY }}"}, ref)
	})

	// A spec that does not parse strictly gets no Spec, as the syntax rule
	// reports it, but still names its kind and version.
	t.Run("spec with an unknown top-level field", func(t *testing.T) {
		t.Parallel()

		source := []byte("version: rudder/v1\nkind: destination\nmetdata:\n  name: typo\nspec:\n  id: x\n")

		assert.Equal(t, &ValidationContext{
			FilePath:   "x.yaml",
			FileName:   "x.yaml",
			Kind:       "destination",
			Version:    "rudder/v1",
			Source:     source,
			References: varsubst.ReplaceReferences(source),
		}, NewSourceContext("x.yaml", &specs.RawSpec{Data: source}))
	})

	t.Run("source that does not parse", func(t *testing.T) {
		t.Parallel()

		source := []byte("spec: [unclosed\n")

		assert.Equal(t, &ValidationContext{
			FilePath:   "broken.yaml",
			FileName:   "broken.yaml",
			Source:     source,
			References: varsubst.ReplaceReferences(source),
		}, NewSourceContext("broken.yaml", &specs.RawSpec{Data: source}))
	})

	// A source with no reference is parsed through the RawSpec, so the load
	// reuses that parse when substitution leaves the source unchanged.
	t.Run("source with no reference keeps its parse", func(t *testing.T) {
		t.Parallel()

		source := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: destination\nspec:\n  id: x\n")}

		ctx := NewSourceContext("x.yaml", source)

		require.NotNil(t, source.Parsed(), "the parse is kept on the RawSpec")
		assert.Equal(t, map[string]any{"id": "x"}, ctx.Spec)
		assert.Equal(t, source.Parsed().Spec, ctx.Spec)
	})
}
