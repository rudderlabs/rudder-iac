package importer

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
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

const varFileHeader = `# Variables referenced by the imported specs. Fill in every value before
# applying, and keep this file out of version control.
#
# An unfilled (null) entry makes apply fail rather than silently sending an
# empty secret; to deliberately send an empty value, use KEY: "".
`

// scaffoldSecretsVarFile writes a fill-in-the-blanks var file for every
// variable referenced by the generated entities. Only active under the
// enableVarSubstitution experimental gate — without substitution the
// references could never be resolved on apply.
func scaffoldSecretsVarFile(ctx context.Context, baseDir string, entities []writer.FormattableEntity) (string, error) {
	if !config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
		return "", nil
	}

	names := collectVariableNames(entities)
	if len(names) == 0 {
		return "", nil
	}

	vars := make(map[string]any, len(names))
	for _, name := range names {
		// nil scaffolds a "KEY:" line; the var-file resolver rejects null
		// values, so an unfilled placeholder fails apply loudly.
		vars[name] = nil
	}

	entity := writer.FormattableEntity{
		Content:      vars,
		RelativePath: SecretsVarFileName,
	}
	// writer.Write fails if the file already exists: a var file may hold real
	// secret values the user filled in, so it is never overwritten or merged.
	if err := writer.Write(ctx, baseDir, formatter.Setup(varFileFormatter{}), []writer.FormattableEntity{entity}); err != nil {
		return "", fmt.Errorf("writing var file: %w", err)
	}

	return filepath.Join(baseDir, SecretsVarFileName), nil
}

// varFileFormatter renders the flat name→value var-file map — one sorted
// "KEY: value" entry per line under an explanatory header. It exists because
// yaml.Marshal can emit neither comments nor the bare "KEY:" placeholder form,
// and implementing formatter.Formatter lets the var file flow through the same
// writer machinery as every other generated file.
type varFileFormatter struct{}

func (varFileFormatter) Extension() []string {
	return []string{loader.ExtensionYAML, loader.ExtensionYML}
}

func (varFileFormatter) Format(data any) ([]byte, error) {
	vars, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", data)
	}

	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(varFileHeader)
	for _, name := range names {
		if vars[name] == nil {
			fmt.Fprintf(&b, "%s:\n", name)
			continue
		}
		entry, err := yaml.Marshal(map[string]any{name: vars[name]})
		if err != nil {
			return nil, fmt.Errorf("marshaling var file entry %q: %w", name, err)
		}
		b.Write(entry)
	}
	return []byte(b.String()), nil
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

// extractVariableNames recursively walks an entity's content — spec wrappers,
// maps, and slices — and records the name of every "{{ .VAR }}" reference found
// in the string values it reaches.
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
