package funcs

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	UpperCaseAlphaNumericWithUnderscoresAndHyphensRegex = "^[A-Z][A-Za-z0-9_-]*$"
)

var PatternUserInfoLookup = map[string]string{
	UpperCaseAlphaNumericWithUnderscoresAndHyphensRegex: "must start with uppercase and contain only alphanumeric, underscores, or hyphens",
}

// customTagErrorMessages is a map of custom tag names to error messages
// which are used mainly for custom regex patterns
var customTagErrorMessages = make(map[string]string)

// NewPattern creates a new regex pattern validator for a given tag name, pattern and error message
func NewPattern(tagName string, pattern string, errorMessage string) rules.CustomValidateFunc {
	// this will be used when we need to display the error
	// message if the validation fails for the given tagname
	customTagErrorMessages[tagName] = errorMessage

	regex := regexp.MustCompile(pattern)
	fn := validator.Func(func(fl validator.FieldLevel) bool {
		return regex.MatchString(fl.Field().String())
	})

	return rules.CustomValidateFunc{
		Tag:  tagName,
		Func: fn,
	}
}
