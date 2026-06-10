package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"gopkg.in/yaml.v3"
)

// SecretsVarFileName is the var file scaffolded next to imported specs, holding
// one placeholder per variable the specs reference (secret fields are exported
// as "{{ .VAR }}" references). The ".vars.yaml" suffix keeps it out of spec
// loading. Users fill in the real values and pass it to apply via --var-file.
const SecretsVarFileName = "secrets.vars.yaml" //nolint:gosec // a file name, not a credential

// scaffoldSecretsVarFile writes a fill-in-the-blanks var file for every
// variable referenced by the generated entities. Only active under the
// enableVarSubstitution experimental gate — without substitution the
// references could never be resolved on apply. Re-imports merge: values the
// user already filled in are kept, only missing variables gain a placeholder.
func scaffoldSecretsVarFile(baseDir string, entities []writer.FormattableEntity) (string, error) {
	if !config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
		return "", nil
	}

	names := collectVariableNames(entities)
	if len(names) == 0 {
		return "", nil
	}

	path := filepath.Join(baseDir, SecretsVarFileName)
	vars, err := loadExistingVars(path)
	if err != nil {
		return "", err
	}

	for _, name := range names {
		if _, ok := vars[name]; !ok {
			vars[name] = ""
		}
	}

	data, err := yaml.Marshal(vars)
	if err != nil {
		return "", fmt.Errorf("marshaling var file content: %w", err)
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", baseDir, err)
	}

	// 0600: the user is expected to put real secrets in this file.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("writing var file %s: %w", path, err)
	}

	return path, nil
}

func loadExistingVars(path string) (map[string]any, error) {
	vars := make(map[string]any)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return vars, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading existing var file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &vars); err != nil {
		return nil, fmt.Errorf("parsing existing var file %s: %w", path, err)
	}
	return vars, nil
}

// collectVariableNames extracts the names of all "{{ .VAR }}" references in the
// entities' content, sorted and de-duplicated. Scanning the generated content —
// rather than threading names through every export strategy — keeps the var
// file in sync with what the specs actually reference, regardless of which
// provider or strategy produced them.
func collectVariableNames(entities []writer.FormattableEntity) []string {
	found := make(map[string]struct{})
	for _, entity := range entities {
		extractVariableNames(entity.Content, found)
	}

	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func extractVariableNames(content any, found map[string]struct{}) {
	switch v := content.(type) {
	case *specs.Spec:
		if v == nil {
			return
		}
		extractVariableNames(v.Metadata, found)
		extractVariableNames(v.Spec, found)
	case map[string]any:
		for _, item := range v {
			extractVariableNames(item, found)
		}
	case []any:
		for _, item := range v {
			extractVariableNames(item, found)
		}
	case string:
		for _, name := range varsubst.ExtractVariableNames([]byte(v)) {
			found[name] = struct{}{}
		}
	}
}
