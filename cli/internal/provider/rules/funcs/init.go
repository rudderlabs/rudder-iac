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

	// Upstream integrations-config spells the common "short text" constraint as
	// `^(.{0,100})$`, which bounds length *and* forbids line breaks — a plain
	// max=100 would let newlines through.
	NewPattern("single_line_100", `^(.{0,100})$`, "must be at most 100 characters and must not contain line breaks")
}
