package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Serialize writes the RulesDoc to outputDir as rules.yaml and rules.json.
// outputDir is created (mkdir -p) if it does not exist.
func Serialize(doc *RulesDoc, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir %s: %w", outputDir, err)
	}

	yamlBytes, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "rules.yaml"), yamlBytes, 0o644); err != nil {
		return fmt.Errorf("writing rules.yaml: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "rules.json"), jsonBytes, 0o644); err != nil {
		return fmt.Errorf("writing rules.json: %w", err)
	}
	return nil
}
