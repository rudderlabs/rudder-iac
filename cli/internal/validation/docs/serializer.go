package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Output formats for the rules catalog. YAML is consumed by Hugo / humans,
// JSON by LLMs / tooling. Both carry identical snake_case keys.
const (
	FormatYAML = "yaml"
	FormatJSON = "json"
	FormatBoth = "both"
)

// Serialize writes the resolved rules catalog to outDir. The format selects
// which artifacts are written: rules.yaml, rules.json, or both. Files are
// overwritten on each run.
func Serialize(doc DocumentedRules, outDir, format string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir %s: %w", outDir, err)
	}

	yamlBytes, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling yaml: %w", err)
	}

	if format == FormatYAML || format == FormatBoth {
		if err := writeFile(filepath.Join(outDir, "rules.yaml"), yamlBytes); err != nil {
			return err
		}
	}

	if format == FormatJSON || format == FormatBoth {
		// JSON is intentionally derived from the YAML bytes (not marshaled from
		// the struct directly) — see yamlToJSON for the rationale.
		jsonBytes, err := yamlToJSON(yamlBytes)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outDir, "rules.json"), jsonBytes); err != nil {
			return err
		}
	}

	return nil
}

// yamlToJSON derives the JSON artifact from the already-marshaled YAML bytes
// instead of marshaling DocumentedRules to JSON directly.
//
// Design decision: the DocumentedRules types intentionally carry only `yaml:` tags.
// Adding a parallel set of `json:` tags would double the struct-tag maintenance
// burden and risk the two encodings silently drifting apart. So the YAML form is
// the single source of truth for field naming, and JSON is produced from it:
// unmarshaling into a generic map preserves the snake_case yaml keys, and
// encoding/json sorts map keys for byte-stable output.
//
// Trade-off: this relies on a yaml -> generic -> json round-trip, faithful for
// the current schema (strings, one int, string maps). If a field is later added
// whose YAML and JSON scalar representations would differ, revisit this (e.g.
// add `json` tags for that type and marshal it directly).
func yamlToJSON(yamlBytes []byte) ([]byte, error) {
	var generic interface{}
	if err := yaml.Unmarshal(yamlBytes, &generic); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml for json conversion: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling json: %w", err)
	}
	return jsonBytes, nil
}

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
