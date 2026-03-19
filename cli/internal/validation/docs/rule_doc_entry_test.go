package docs

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRuleDocEntries_MalformedYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/bad.yaml": &fstest.MapFile{Data: []byte(":\tinvalid: [yaml")},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parsing bad.yaml")
}

func TestLoadRuleDocEntries_SkipsSubdirectories(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/rule1.yaml":     &fstest.MapFile{Data: []byte("rule_id: \"rule-1\"\n")},
		"rules/subdir/ignored": &fstest.MapFile{Data: []byte("")},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.NoError(t, err)
	require.Len(t, result, 1)
}

func TestLoadRuleDocEntries_SkipsNonYAMLFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/rule1.yaml":  &fstest.MapFile{Data: []byte("rule_id: \"rule-1\"\n")},
		"rules/readme.txt":  &fstest.MapFile{Data: []byte("some text")},
		"rules/config.json": &fstest.MapFile{Data: []byte(`{"key":"val"}`)},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "rule-1", result[0].RuleID)
}

func TestLoadRuleDocEntries_MultipleYAMLFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/rule1.yaml": &fstest.MapFile{Data: []byte("rule_id: \"rule-1\"\n")},
		"rules/rule2.yml":  &fstest.MapFile{Data: []byte("rule_id: \"rule-2\"\n")},
		"rules/rule3.yaml": &fstest.MapFile{Data: []byte("rule_id: \"rule-3\"\n")},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.NoError(t, err)
	require.Len(t, result, 3)
}

func TestLoadRuleDocEntries_SingleValidYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/rule1.yaml": &fstest.MapFile{Data: []byte("rule_id: \"rs.source.missing-name\"\n")},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, RuleDocEntry{RuleID: "rs.source.missing-name"}, result[0])
}

func TestLoadRuleDocEntries_EmptyDir(t *testing.T) {
	fsys := fstest.MapFS{
		"rules/.keep": &fstest.MapFile{Data: []byte("")},
	}
	result, err := LoadRuleDocEntries(fsys, "rules")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadRuleDocEntries_NonexistentDir(t *testing.T) {
	fsys := fstest.MapFS{}
	result, err := LoadRuleDocEntries(fsys, "no-such-dir")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reading rule docs dir no-such-dir")
}
