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

func TestSerializer_EmitsYAMLAndJSON(t *testing.T) {
	dir := t.TempDir()
	doc := &RulesDoc{
		SchemaVersion: 1,
		ToolMetadata:  ToolMetadata{CLIVersion: "v0.0.0"},
		Rules: []ResolvedRule{{
			RuleID:      "r1",
			Phase:       "syntactic",
			Severity:    "error",
			Description: "test",
			AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
			},
		}},
	}

	require.NoError(t, Serialize(doc, dir))

	yamlBytes, err := os.ReadFile(filepath.Join(dir, "rules.yaml"))
	require.NoError(t, err)
	var roundTripYAML RulesDoc
	require.NoError(t, yaml.Unmarshal(yamlBytes, &roundTripYAML))
	assert.Equal(t, doc, &roundTripYAML)

	jsonBytes, err := os.ReadFile(filepath.Join(dir, "rules.json"))
	require.NoError(t, err)
	var roundTripJSON RulesDoc
	require.NoError(t, json.Unmarshal(jsonBytes, &roundTripJSON))
	assert.Equal(t, doc, &roundTripJSON)
}

func TestSerializer_CreatesOutputDirIfMissing(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "doesnt", "exist")
	doc := &RulesDoc{SchemaVersion: 1, ToolMetadata: ToolMetadata{CLIVersion: "v"}}
	require.NoError(t, Serialize(doc, nested))
	_, err := os.Stat(filepath.Join(nested, "rules.yaml"))
	require.NoError(t, err)
}
