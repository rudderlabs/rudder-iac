package funcs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// urlPattern mirrors destination URL field validators in terraform-provider /
// integrations-config: optional http(s) scheme, hostname with at least one
// dot, then path/query/fragment characters.
const urlPattern = `^(?:https?:\/\/)?[\w.-]+(?:\.[\w.-]+)+[\w\-\._~:/?#\[\]@!$&'()*+,;=.]+$`

func init() {
	rules.RegisterDefaultValidator(GetPatternValidator())
	rules.RegisterDefaultValidator(GetArrayItemTypesValidator())

	// Register the default patterns for the pattern validator
	// These patterns can be used by callers downstream to validate fields in struct
	NewPattern("letter_start", "^[a-zA-Z]", "must start with a letter [a-zA-Z]")
	NewPattern("url", urlPattern, "must be a valid URL")
}
