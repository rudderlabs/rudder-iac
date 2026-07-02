package resourceops

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
)

// EncodeYAML serializes v to YAML using the project's canonical formatter,
// matching the format used for spec files on disk (double-quoted strings, 2-space indent).
func EncodeYAML(v any) (string, error) {
	b, err := formatter.DefaultYAML.Format(v)
	if err != nil {
		return "", fmt.Errorf("encoding YAML: %w", err)
	}
	return string(b), nil
}

// EncodeJSON serializes v to indented JSON. Because the spec struct uses yaml struct tags
// (not json tags), we round-trip through YAML first — marshal to YAML bytes, unmarshal
// into a generic map, then marshal that map to JSON — so the resulting keys are the
// lowercase YAML field names (e.g. "kind", "version") rather than Go field names.
func EncodeJSON(v any) (string, error) {
	yamlBytes, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshaling to YAML for JSON round-trip: %w", err)
	}

	var m map[string]any
	if err := yaml.Unmarshal(yamlBytes, &m); err != nil {
		return "", fmt.Errorf("unmarshaling YAML for JSON round-trip: %w", err)
	}

	jsonBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding JSON: %w", err)
	}
	return string(jsonBytes), nil
}
