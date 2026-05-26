package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EmitYAML marshals the RulesDoc to YAML and writes it to <dir>/rules.yaml.
// Returns the path written. Creates dir if it doesn't exist.
func EmitYAML(doc *RulesDoc, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "rules.yaml")
	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshalling YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}

// EmitJSON marshals the RulesDoc to indented JSON and writes it to
// <dir>/rules.json. Returns the path written.
func EmitJSON(doc *RulesDoc, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "rules.json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}
