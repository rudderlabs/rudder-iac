package funcs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	LegacyReferenceTag   = "legacy_reference"
	LegacyReferenceRegex = `^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`

	ReferenceTag   = "reference"
	ReferenceRegex = `^#(%s):([a-zA-Z0-9_-]+)$`
)

// NewLegacyReferenceValidateFunc creates a validator for legacy reference format: #/<kind>/<group>/<id>
// Example:
//
//	#/properties/user-traits/email
//	#/events/tracking/page_viewed
//	#/custom-types/abc
func NewLegacyReferenceValidateFunc(allowedKinds []string) rules.CustomValidateFunc {
	pattern := BuildLegacyReferenceRegex(allowedKinds)

	var fn validator.Func = func(fl validator.FieldLevel) bool {
		return pattern.MatchString(fl.Field().String())
	}

	return rules.CustomValidateFunc{
		Tag:  "reference",
		Func: fn,
	}
}

// NewReferenceValidateFunc creates a validator for reference format: #kind:id
// Example:
//
//	#properties:email
//	#events:page_viewed
//	#custom-types:abc
func NewReferenceValidateFunc(allowedKinds []string) rules.CustomValidateFunc {
	pattern := BuildReferenceRegex(allowedKinds)

	var fn validator.Func = func(fl validator.FieldLevel) bool {
		return pattern.MatchString(fl.Field().String())
	}

	return rules.CustomValidateFunc{
		Tag:  "reference",
		Func: fn,
	}
}

// BuildLegacyReferenceRegex creates regex for legacy format: #/<kind>/<group>/<id>
func BuildLegacyReferenceRegex(kinds []string) *regexp.Regexp {
	if len(kinds) == 0 {
		return regexp.MustCompile(`^$`)
	}

	escapedKinds := make([]string, len(kinds))
	for i, kind := range kinds {
		escapedKinds[i] = regexp.QuoteMeta(kind)
	}
	kindsPattern := strings.Join(escapedKinds, "|")

	pattern := fmt.Sprintf(`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`, kindsPattern)
	return regexp.MustCompile(pattern)
}

// BuildReferenceRegex creates regex for reference format: #kind:id
func BuildReferenceRegex(kinds []string) *regexp.Regexp {
	if len(kinds) == 0 {
		return regexp.MustCompile(`^$`)
	}

	escapedKinds := make([]string, len(kinds))
	for i, kind := range kinds {
		escapedKinds[i] = regexp.QuoteMeta(kind)
	}
	kindsPattern := strings.Join(escapedKinds, "|")

	pattern := fmt.Sprintf(`^#(%s):([a-zA-Z0-9_-]+)$`, kindsPattern)
	return regexp.MustCompile(pattern)
}
