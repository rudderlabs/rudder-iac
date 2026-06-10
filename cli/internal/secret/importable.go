package secret

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// ImportableSecret is the spec-side flavour of a secret for resources that
// support import scaffolding. It embeds String — so loading, redaction, and
// every formatting surface behave exactly like a plain secret — and adds the
// name of the substitution variable to emit in its place during export.
//
// A remote never returns a secret's real value, so exporting one as a literal
// would only produce a useless mask. When VarName is set, the marshals emit a
// quoted "{{ .VarName }}" reference instead; the importer then scaffolds a var
// file with one placeholder per referenced variable, and variable substitution
// injects the real value on a later apply. When VarName is empty (every load
// path, or scaffolding disabled), it serializes exactly like String.
type ImportableSecret struct {
	String
	VarName string
}

// NewImportable wraps a secret with the substitution variable that stands in
// for it during import scaffolding. The provider chooses varName from what it
// knows about the resource (e.g. resource type, external ID, field), which is
// what keeps names deterministic and stable across re-imports; the name is
// normalized to the substitution grammar (UPPER_SNAKE, invalid runes folded to
// "_"). Scaffolding only works under the enableVarSubstitution experimental
// gate — without substitution the reference could never be resolved on apply —
// so with the gate off the name is dropped and the secret exports as a masked
// literal, the pre-scaffolding behaviour.
func NewImportable(s String, varName string) ImportableSecret {
	if !config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
		return ImportableSecret{String: s}
	}
	return ImportableSecret{String: s, VarName: normalizeVarName(varName)}
}

// MarshalJSON emits the variable reference when a name is set; otherwise the
// embedded String redacts as usual. Export runs through json.Marshal, so this
// is what places the "{{ .VAR }}" token in the generated spec.
func (s ImportableSecret) MarshalJSON() ([]byte, error) {
	if s.VarName == "" {
		return s.String.MarshalJSON()
	}
	return json.Marshal(s.token())
}

// MarshalYAML mirrors MarshalJSON for direct YAML serialization.
func (s ImportableSecret) MarshalYAML() (any, error) {
	if s.VarName == "" {
		return s.String.MarshalYAML()
	}
	return s.token(), nil
}

func (s ImportableSecret) token() string {
	return fmt.Sprintf("{{ .%s }}", s.VarName)
}

var (
	invalidVarRunes = regexp.MustCompile(`[^A-Za-z0-9_]+`)
	leadingDigit    = regexp.MustCompile(`^[0-9]`)
)

// normalizeVarName folds a provider-chosen name (often containing kebab-case
// external IDs) into the substitutor's variable grammar
// (^[A-Za-z_][A-Za-z0-9_]*$), uppercased by convention.
func normalizeVarName(name string) string {
	name = invalidVarRunes.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	name = strings.ToUpper(name)
	if leadingDigit.MatchString(name) {
		name = "_" + name
	}
	return name
}
