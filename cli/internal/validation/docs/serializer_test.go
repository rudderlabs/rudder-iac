package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func sampleDoc() *RulesDoc {
	return &RulesDoc{
		SchemaVersion: 1,
		ToolMetadata:  ToolMetadata{CLIVersion: "test-1.0"},
		Rules: []ResolvedRule{
			{
				RuleID:      "rule-a",
				Phase:       "syntactic",
				Severity:    "error",
				Description: "the rule",
				AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
				MatchBehavior: []MatchBehaviorEntry{
					{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
				},
			},
		},
	}
}

func TestEmitYAML_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	doc := sampleDoc()

	path, err := EmitYAML(doc, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "rules.yaml"), path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var back RulesDoc
	require.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, doc, &back)
}

func TestEmitJSON_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	doc := sampleDoc()

	path, err := EmitJSON(doc, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "rules.json"), path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var back RulesDoc
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, doc, &back)
}

func TestEmitYAML_CreatesOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	_, err := EmitYAML(sampleDoc(), dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "rules.yaml"))
	require.NoError(t, err)
}
