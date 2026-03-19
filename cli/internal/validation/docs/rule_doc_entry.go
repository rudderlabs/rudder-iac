package docs

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadRuleDocEntries(fsys fs.FS, dir string) ([]RuleDocEntry, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading rule docs dir %s: %w", dir, err)
	}

	var result []RuleDocEntry
	for _, entry := range entries {
		if shouldSkipEntry(entry) {
			continue
		}

		ruleDoc, err := loadYAMLEntry(fsys, dir, entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, ruleDoc)
	}
	return result, nil
}

func shouldSkipEntry(entry fs.DirEntry) bool {
	return entry.IsDir() || !isYAMLFile(entry.Name())
}

func isYAMLFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

func loadYAMLEntry(fsys fs.FS, dir, name string) (RuleDocEntry, error) {
	data, err := fs.ReadFile(fsys, filepath.Join(dir, name))
	if err != nil {
		return RuleDocEntry{}, fmt.Errorf("reading %s: %w", name, err)
	}

	var ruleDoc RuleDocEntry
	if err := yaml.Unmarshal(data, &ruleDoc); err != nil {
		return RuleDocEntry{}, fmt.Errorf("parsing %s: %w", name, err)
	}
	return ruleDoc, nil
}
