package project_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
)

// TestProject_Load_JSONRenderer pins the machine-readable contract of
// `validate --format json`: on a spec with a rule violation, the JSON renderer
// emits a diagnostic carrying the stable code, severity, kind, file and source
// position. The code set is asserted explicitly so it cannot drift silently —
// codes are an API for downstream tooling.
func TestProject_Load_JSONRenderer(t *testing.T) {
	t.Parallel()

	// kind Destination + version rudder/v1 is an unsupported pair (Destination is
	// only known at rudder/0.1), so the project/resource-kind-version-valid
	// gatekeeper flags it at /kind — a deterministic syntactic violation.
	const spec = "kind: Destination\nversion: rudder/v1\nmetadata:\n  name: my_dest\nspec:\n  k: v"

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockProvider.MatchPatterns = fixtureMatchPatterns

	mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
		return map[string]*specs.RawSpec{
			"specs/dest.yaml": {Data: []byte(spec)},
		}, nil
	}}

	var buf bytes.Buffer
	proj := project.New(mockProvider,
		project.WithLoader(mockLoader),
		project.WithRenderer(renderer.NewJSONRenderer(&buf)),
	)

	err := proj.Load("test_dir")
	require.Error(t, err) // validation failed, but diagnostics were rendered as JSON

	var out struct {
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Kind     string `json:"kind"`
			File     string `json:"file"`
			Line     int    `json:"line"`
			Col      int    `json:"col"`
			RuleDoc  string `json:"ruleDoc"`
		} `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	require.Len(t, out.Diagnostics, 1)

	d := out.Diagnostics[0]
	assert.Equal(t, "project/resource-kind-version-valid", d.Code)
	assert.Equal(t, "error", d.Severity)
	assert.Equal(t, "Destination", d.Kind)
	assert.Equal(t, "specs/dest.yaml", d.File)
	assert.Equal(t, 1, d.Line) // /kind is on line 1
	assert.Greater(t, d.Col, 0)
	assert.Equal(t, "docs/generated/rules.yaml#project/resource-kind-version-valid", d.RuleDoc)

	// Pin the emitted code set so new/renamed rule codes surface as a test change.
	codes := make([]string, 0, len(out.Diagnostics))
	for _, diag := range out.Diagnostics {
		codes = append(codes, diag.Code)
	}
	assert.Equal(t, []string{"project/resource-kind-version-valid"}, codes)
}
