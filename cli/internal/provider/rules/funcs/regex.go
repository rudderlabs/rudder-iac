package funcs

import (
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// patternRegistry holds pre-compiled regex patterns and their error messages.
// Optional reject patterns fail validation when matched (e.g. ngrok hosts).
type patternRegistry struct {
	patterns map[string]*regexp.Regexp
	rejects  map[string]*regexp.Regexp
	errors   map[string]string
	mu       sync.RWMutex
}

// registry is the global pattern registry for named patterns
var registry = &patternRegistry{
	patterns: make(map[string]*regexp.Regexp),
	rejects:  make(map[string]*regexp.Regexp),
	errors:   make(map[string]string),
}

// Register adds a named allow pattern to the registry.
func (r *patternRegistry) Register(name, pattern, errorMsg string) {
	r.RegisterWithReject(name, pattern, "", errorMsg)
}

// RegisterWithReject adds a named allow pattern and optional reject pattern.
// When rejectPattern is non-empty, values matching it fail even if they match pattern.
func (r *patternRegistry) RegisterWithReject(name, pattern, rejectPattern, errorMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.patterns[name] = regexp.MustCompile(pattern)
	r.errors[name] = errorMsg
	if rejectPattern == "" {
		delete(r.rejects, name)
		return
	}
	r.rejects[name] = regexp.MustCompile(rejectPattern)
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

func (r *patternRegistry) match(name, value string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pattern, ok := r.patterns[name]
	if !ok {
		return false
	}
	if reject, hasReject := r.rejects[name]; hasReject && reject.MatchString(value) {
		return false
	}
	return pattern.MatchString(value)
}

// getPatternErrorMessage retrieves the custom error message for a registered pattern
// from the registry and returns if the pattern is registered.
func getPatternErrorMessage(patternName string) (string, bool) {
	_, msg, ok := registry.Get(patternName)
	return msg, ok
}

// NewPattern registers a named pattern in the global pattern registry
// Usage: NewPattern("customtype_name", "^[a-zA-Z_][a-zA-Z0-9_]*$", "must be valid identifier")
// Then use in struct tags: validate:"pattern=customtype_name"
func NewPattern(name string, pattern string, errorMessage string) {
	registry.Register(name, pattern, errorMessage)
}

// NewPatternWithReject registers an allow pattern plus a reject pattern.
// Values matching rejectPattern fail validation even when they match pattern.
// Usage: validate:"pattern=name"
func NewPatternWithReject(name, pattern, rejectPattern, errorMessage string) {
	registry.RegisterWithReject(name, pattern, rejectPattern, errorMessage)
}

// GetPatternValidator returns the global pattern validator that should be registered
// in ValidateStruct. This validator handles all validate:"pattern=name" tags.
func GetPatternValidator() rules.CustomValidateFunc {
	fn := validator.Func(func(fl validator.FieldLevel) bool {
		return registry.match(fl.Param(), fl.Field().String())
	})

	return rules.CustomValidateFunc{
		Tag:  "pattern",
		Func: fn,
	}
}
