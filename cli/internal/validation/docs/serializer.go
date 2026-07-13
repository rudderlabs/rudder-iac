package docs

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Serialize writes the resolved rules catalog as rules.yaml to outDir. YAML is
// the canonical artifact consumed by the Hugo translation layer downstream. The
// file is overwritten on each run.
//
// JSON output (for LLM tooling) was intentionally dropped: there is no consumer
// today, and it can be re-derived from the canonical YAML as a quick incremental
// lift when one appears.
func Serialize(doc DocumentedRules, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir %s: %w", outDir, err)
	}

	yamlBytes, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling yaml: %w", err)
	}

	if err := writeFile(filepath.Join(outDir, "rules.yaml"), yamlBytes); err != nil {
		return err
	}

	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
