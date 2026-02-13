package funcs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func init() {
	rules.RegisterDefaultValidator(GetPatternValidator())
	rules.RegisterDefaultValidator(GetArrayItemTypesValidator())

	// Register the default patterns for the pattern validator
	// These patterns can be used by callers downstream to validate fields in struct
	NewPattern("letter_start", "^[a-zA-Z]", "must start with a letter [a-zA-Z]")
}
