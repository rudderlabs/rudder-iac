package funcs

import (
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// patternRegistry holds pre-compiled regex patterns and their error messages
type patternRegistry struct {
	patterns map[string]*regexp.Regexp
	errors   map[string]string
	mu       sync.RWMutex
}

// registry is the global pattern registry for named patterns
var registry = &patternRegistry{
	patterns: make(map[string]*regexp.Regexp),
	errors:   make(map[string]string),
}

// Register adds a named pattern to the registry
func (r *patternRegistry) Register(name, pattern, errorMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	compiled := regexp.MustCompile(pattern)

	r.patterns[name] = compiled
	r.errors[name] = errorMsg
}

// Get retrieves a pattern and its error message from the registry
func (r *patternRegistry) Get(name string) (*regexp.Regexp, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pattern, ok := r.patterns[name]
	if !ok {
		return nil, "", false
	}
	return pattern, r.errors[name], true
}

// GetPatternErrorMessage retrieves the custom error message for a registered pattern
func getPatternErrorMessage(patternName string) (string, bool) {
	_, msg, ok := registry.Get(patternName)
	return msg, ok
}

// NewPattern registers a named pattern in the global pattern registry
// Usage: NewPattern("customtype_name", "^[a-zA-Z_][a-zA-Z0-9_]*$", "must be valid identifier")
// Then use in struct tags: validate:"pattern:customtype_name"
func NewPattern(name string, pattern string, errorMessage string) {
	registry.Register(name, pattern, errorMessage)
}

// GetPatternValidator returns the global pattern validator that should be registered
// in ValidateStruct. This validator handles all validate:"pattern:name" tags.
func GetPatternValidator() rules.CustomValidateFunc {
	fn := validator.Func(func(fl validator.FieldLevel) bool {
		patternName := fl.Param()

		regex, _, ok := registry.Get(patternName)
		if !ok {
			return false
		}

		return regex.MatchString(fl.Field().String())
	})

	return rules.CustomValidateFunc{
		Tag:  "pattern",
		Func: fn,
	}
}

func init() {
	// Register the pattern validator as a default validator
	rules.RegisterDefaultValidator(GetPatternValidator())
}
