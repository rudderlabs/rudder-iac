package renderer_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func diagnostic(file string, line int) validation.Diagnostic {
	return validation.Diagnostic{
		RuleID:   "test/rule",
		Severity: rules.Error,
		Message:  "boom",
		File:     file,
		Position: pathindex.Position{Line: line, Column: 1},
	}
}

// A single project load renders twice — syntax diagnostics before the graph is
// built, semantic ones after — so the second call must not discard the first.
func TestCaptureRenderer_AccumulatesAcrossCalls(t *testing.T) {
	r := renderer.NewCaptureRenderer()

	assert.NoError(t, r.Render(validation.Diagnostics{diagnostic("events.yaml", 3)}))
	assert.NoError(t, r.Render(validation.Diagnostics{diagnostic("properties.yaml", 9)}))

	assert.Equal(t, validation.Diagnostics{
		diagnostic("events.yaml", 3),
		diagnostic("properties.yaml", 9),
	}, r.Diagnostics())
}

func TestCaptureRenderer_SortsByFileThenPosition(t *testing.T) {
	r := renderer.NewCaptureRenderer()

	assert.NoError(t, r.Render(validation.Diagnostics{
		diagnostic("properties.yaml", 9),
		diagnostic("events.yaml", 20),
		diagnostic("events.yaml", 3),
	}))

	assert.Equal(t, validation.Diagnostics{
		diagnostic("events.yaml", 3),
		diagnostic("events.yaml", 20),
		diagnostic("properties.yaml", 9),
	}, r.Diagnostics())
}

func TestCaptureRenderer_EmptyByDefault(t *testing.T) {
	assert.Equal(t, validation.Diagnostics{}, renderer.NewCaptureRenderer().Diagnostics())
}
