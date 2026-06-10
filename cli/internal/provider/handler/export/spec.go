package export

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

type SpecExportData[Spec any] struct {
	RelativePath string
	Data         *Spec
}

func (s *SpecExportData[Spec]) ToMap() (map[string]any, error) {
	// With variable substitution enabled, secret fields are exported as
	// "{{ .VAR }}" references instead of masked literals, so the generated spec
	// stays applyable; the importer scaffolds a var file for the references.
	if config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
		if err := scaffoldSecretRefs(s.Data, varPathPrefix(s.RelativePath)); err != nil {
			return nil, fmt.Errorf("scaffolding secret references: %w", err)
		}
	}

	bytes, err := json.Marshal(s.Data)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
