package secret

import (
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// WithVariableName attaches the substitution variable that stands in for the
// secret during import scaffolding, making the marshals emit a quoted
// "{{ .name }}" reference instead of a masked literal. The provider building
// the export spec chooses the name from what it knows about the resource
// (e.g. resource type, external ID, field), which is what keeps names
// deterministic and stable across re-imports; the name is normalized to the
// substitution grammar (UPPER_SNAKE, invalid runes folded to "_").
//
// Scaffolding only works under the enableVarSubstitution experimental gate —
// without substitution the reference could never be resolved on apply — so
// with the gate off this option is a no-op and the secret exports as a masked
// literal, the pre-scaffolding behaviour.
func WithVariableName(name string) Option {
	return func(s *String) {
		if !config.GetConfig().ExperimentalFlags.EnableVarSubstitution {
			return
		}
		s.varName = normalizeVarName(name)
	}
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
