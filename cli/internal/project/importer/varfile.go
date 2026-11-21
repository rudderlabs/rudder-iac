package importer

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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
#
# Generated specs write each reference as '{{ .VAR }}', in single quotes, and
# a one-line value is inserted as is.
#
# For a JSON value (such as a service account key), wrap it in single quotes
# here:
#   KEY: '{"private_key": "...", ...}'
# Double quotes turn the \n escapes inside the key into real newlines, which
# breaks the private key.
#
# A value that contains a single quote needs each one doubled, and is written
# without quotes in this file. For the password it's-a-secret, write:
#   KEY: it''s-a-secret
#
# For a multi-line value such as a PEM certificate or key, use a YAML literal
# block so its line breaks stay intact:
#   KEY: |
#     -----BEGIN CERTIFICATE-----
#     ...
#     -----END CERTIFICATE-----
# The generated '{{ .VAR }}' references keep those line breaks.
`

// scaffoldSecretsVarFile writes a fill-in-the-blanks var file for every
// variable referenced by the generated entities.
func scaffoldSecretsVarFile(ctx context.Context, baseDir string, entities []writer.FormattableEntity) (string, error) {
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
//
// Raw code files are skipped: they are loaded verbatim, never substituted, so a
// "{{ .NAME }}" in them needs no value, and an unfilled entry would fail every
// run that loads the var file.
func collectVariableNames(entities []writer.FormattableEntity) []string {
	found := make(map[string]struct{})
	for _, entity := range entities {
		if !isSubstitutedSpec(entity.Content) {
			continue
		}
		extractVariableNames(entity.Content, found)
	}

	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// checkVariableOwners refuses an import in which one secret variable would
// stand for two different secrets. Every provider names a secret variable from
// the resource id and key path joined with "_" (secret.VariableName), so two
// different secrets can share a name: destination "hook" with key bearer_token
// and destination "hook-bearer" with key token, or a destination and an
// account both imported as "pg" with a password. A slice-by-slice adoption
// imports into a project that already holds earlier slices, so the specs
// already under location count as well as the new entities. One var file entry
// would otherwise fill both.
//
// A reference's owner is the spec key it fills: kind, spec id and key path (or
// the file, for content that is not a spec). A new reference is refused only
// when it adds an owner the variable does not have yet next to another one: a
// re-import of a key that already uses the variable is fine, and so are specs
// already in the project that share a variable among themselves. Raw code
// files, such as a transformation's script, are loaded verbatim rather than
// substituted and do not count.
func checkVariableOwners(location string, entities []writer.FormattableEntity) error {
	type use struct {
		owner, file string
		isNew       bool
	}
	uses := map[string][]use{}
	existingOwners := map[string]map[string]bool{}

	if location != "" {
		raw, err := (&loader.Loader{}).Load(location)
		if err != nil {
			return fmt.Errorf("reading project specs for secret variables: %w", err)
		}
		for path, rawSpec := range raw {
			rel, err := filepath.Rel(location, path)
			if err != nil {
				rel = path
			}
			// References are found as substitution finds them, in the bytes as
			// written, and each is swapped for a placeholder so the spec parses
			// even where a reference is unquoted.
			replaced := varsubst.ReplaceReferences(rawSpec.Data)
			if len(replaced.References) == 0 {
				continue
			}
			spec, err := specs.New(replaced.Data)
			if err != nil {
				return fmt.Errorf("reading project spec %s for secret variables: %w", rel, err)
			}
			// Substitution runs over the whole document, so every field and
			// mapping key counts, not only the spec body.
			var document map[string]any
			if err := yaml.Unmarshal(replaced.Data, &document); err != nil {
				return fmt.Errorf("reading project spec %s for secret variables: %w", rel, err)
			}
			id := specOwner(spec, rel)
			walkStrings(document, "", func(keyPath, value string) {
				for _, ref := range replaced.ReferencesIn(value) {
					owner := id + ":" + keyPath
					uses[ref.Name] = append(uses[ref.Name], use{owner: owner, file: rel})
					if existingOwners[ref.Name] == nil {
						existingOwners[ref.Name] = map[string]bool{}
					}
					existingOwners[ref.Name][owner] = true
				}
			})
		}
	}
	for _, entity := range entities {
		if !isSubstitutedSpec(entity.Content) {
			continue
		}
		// Named from the project root, as the existing specs are, since that is
		// where the import writes it.
		newFile := filepath.Join(ImportedDir, entity.RelativePath)
		id, content := entity.RelativePath, entity.Content
		if spec, ok := entity.Content.(*specs.Spec); ok && spec != nil {
			id = specOwner(spec, entity.RelativePath)
			content = map[string]any{"version": spec.Version, "kind": spec.Kind, "metadata": spec.Metadata, "spec": spec.Spec}
		}
		walkStrings(content, "", func(keyPath, value string) {
			for _, name := range varsubst.ExtractVariableNames([]byte(value)) {
				uses[name] = append(uses[name], use{owner: id + ":" + keyPath, file: newFile, isNew: true})
			}
		})
	}

	var clashes []string
	for name, all := range uses {
	clash:
		for _, added := range all {
			if !added.isNew || existingOwners[name][added.owner] {
				continue
			}
			for _, other := range all {
				if other.owner != added.owner {
					clashes = append(clashes, fmt.Sprintf("%s (in %s and %s)", name, other.file, added.file))
					break clash
				}
			}
		}
	}
	if len(clashes) == 0 {
		return nil
	}
	sort.Strings(clashes)
	return fmt.Errorf("imported specs would share secret variables: %s; the names come from each resource's id, so rename one of the resources whose ids collide and import again",
		strings.Join(clashes, ", "))
}

// walkStrings calls visit with the document path of every string under value,
// mapping keys included: a key is visited at its own path with a "(key)"
// suffix, since substitution replaces a reference wherever it is written.
func walkStrings(value any, path string, visit func(keyPath, value string)) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			keyPath := key
			if path != "" {
				keyPath = path + "." + key
			}
			visit(keyPath+" (key)", key)
			walkStrings(item, keyPath, visit)
		}
	case []any:
		for i, item := range v {
			walkStrings(item, fmt.Sprintf("%s[%d]", path, i), visit)
		}
	case string:
		visit(path, v)
	}
}

// specOwner names the resource a spec declares: kind and spec id, or the file
// when the spec has no id.
func specOwner(spec *specs.Spec, file string) string {
	if id, ok := spec.Spec["id"].(string); ok && id != "" {
		return spec.Kind + "/" + id
	}
	return file
}

// isSubstitutedSpec reports whether content is a spec the loader runs
// variable substitution over, as opposed to a raw file written as is.
func isSubstitutedSpec(content any) bool {
	switch content.(type) {
	case *specs.Spec, map[string]any:
		return true
	default:
		return false
	}
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
