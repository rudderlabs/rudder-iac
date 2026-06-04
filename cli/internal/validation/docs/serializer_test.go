package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureDocumentedRules(generatedAt string) DocumentedRules {
	return DocumentedRules{
		SchemaVersion: 1,
		ToolMetadata: ToolMetadata{
			CLIVersion:  "1.2.3",
			GeneratedAt: generatedAt,
		},
		Rules: []DocumentedRule{
			{
				RuleID:      "datacatalog/categories/spec-syntax-valid",
				Provider:    "datacatalog",
				Phase:       "syntactic",
				Severity:    "error",
				Description: "Spec syntax must be valid.",
				AppliesTo: []MatchPatternDoc{
					{Kind: "categories", Version: "v1"},
				},
				MatchBehavior: []MatchBehaviorEntry{
					{
						AppliesTo: []MatchPatternDoc{
							{Kind: "categories", Version: "v1"},
						},
						Valid: []ValidExample{
							{
								ExampleID: "ok-1",
								Title:     "A valid spec",
								Files:     map[string]string{"a.yaml": "kind: categories"},
							},
						},
						Invalid: []InvalidExample{
							{
								ExampleID: "bad-1",
								Title:     "A broken spec",
								Files:     map[string]string{"b.yaml": "kind: ???"},
								ExpectedDiagnostics: []ExpectedDiagnostic{
									{
										File:            "b.yaml",
										Reference:       "$.kind",
										Severity:        "error",
										MessageContains: "invalid",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// stripGeneratedAt removes the generated_at line so two runs with different
// timestamps can be compared for byte-stability of everything else.
func stripGeneratedAt(s string) string {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "generated_at") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func TestSerializeYAMLStableIgnoringGeneratedAt(t *testing.T) {
	var (
		dir1 = t.TempDir()
		dir2 = t.TempDir()
	)

	require.NoError(t, Serialize(fixtureDocumentedRules("2026-01-01T00:00:00Z"), dir1, FormatBoth))
	require.NoError(t, Serialize(fixtureDocumentedRules("2026-12-31T23:59:59Z"), dir2, FormatBoth))

	yaml1, err := os.ReadFile(filepath.Join(dir1, "rules.yaml"))
	require.NoError(t, err)
	yaml2, err := os.ReadFile(filepath.Join(dir2, "rules.yaml"))
	require.NoError(t, err)

	assert.Contains(t, string(yaml1), "generated_at")
	assert.Equal(t, stripGeneratedAt(string(yaml1)), stripGeneratedAt(string(yaml2)))
}

func TestSerializeJSONSnakeCase(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Serialize(fixtureDocumentedRules("2026-01-01T00:00:00Z"), dir, FormatBoth))

	data, err := os.ReadFile(filepath.Join(dir, "rules.json"))
	require.NoError(t, err)

	var generic interface{}
	require.NoError(t, json.Unmarshal(data, &generic), "rules.json must be valid JSON")

	content := string(data)
	assert.Contains(t, content, `"rule_id"`)
	assert.Contains(t, content, `"schema_version"`)
	assert.NotContains(t, content, `"RuleID"`)
	assert.NotContains(t, content, `"SchemaVersion"`)
}

func TestSerializeWritesAndOverwritesBothFiles(t *testing.T) {
	dir := t.TempDir()

	var (
		yamlPath = filepath.Join(dir, "rules.yaml")
		jsonPath = filepath.Join(dir, "rules.json")
	)

	require.NoError(t, Serialize(fixtureDocumentedRules("2026-01-01T00:00:00Z"), dir, FormatBoth))
	assert.FileExists(t, yamlPath)
	assert.FileExists(t, jsonPath)

	// Second run with different content must overwrite cleanly.
	require.NoError(t, Serialize(fixtureDocumentedRules("2026-06-01T00:00:00Z"), dir, FormatBoth))

	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Contains(t, string(yamlData), "2026-06-01T00:00:00Z")
	assert.NotContains(t, string(yamlData), "2026-01-01T00:00:00Z")
}

func TestSerializeFormatGatesFiles(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		wantYAMLFile bool
		wantJSONFile bool
	}{
		{name: "yaml only", format: FormatYAML, wantYAMLFile: true, wantJSONFile: false},
		{name: "json only", format: FormatJSON, wantYAMLFile: false, wantJSONFile: true},
		{name: "both", format: FormatBoth, wantYAMLFile: true, wantJSONFile: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, Serialize(fixtureDocumentedRules("2026-01-01T00:00:00Z"), dir, tc.format))

			_, yamlErr := os.Stat(filepath.Join(dir, "rules.yaml"))
			_, jsonErr := os.Stat(filepath.Join(dir, "rules.json"))

			assert.Equal(t, tc.wantYAMLFile, yamlErr == nil, "rules.yaml presence")
			assert.Equal(t, tc.wantJSONFile, jsonErr == nil, "rules.json presence")
		})
	}
}
