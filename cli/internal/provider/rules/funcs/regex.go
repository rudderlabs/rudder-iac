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

// Register adds a named pattern to the registry
func (r *patternRegistry) Register(name, pattern, errorMsg string) {
	r.RegisterWithReject(name, pattern, "", errorMsg)
}

// RegisterWithReject adds a named pattern and optional reject pattern to the registry.
func (r *patternRegistry) RegisterWithReject(name, pattern, rejectPattern, errorMsg string) {
	compiled := regexp.MustCompile(pattern)

	var reject *regexp.Regexp
	if rejectPattern != "" {
		reject = regexp.MustCompile(rejectPattern)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.patterns == nil {
		r.patterns = make(map[string]*regexp.Regexp)
	}
	if r.rejects == nil {
		r.rejects = make(map[string]*regexp.Regexp)
	}
	if r.errors == nil {
		r.errors = make(map[string]string)
	}

	r.patterns[name] = compiled
	r.errors[name] = errorMsg

	if reject == nil {
		delete(r.rejects, name)
		return
	}
	r.rejects[name] = reject
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

	if reject, ok := r.rejects[name]; ok && reject.MatchString(value) {
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

// NewPatternWithReject registers a named pattern with an optional reject pattern in the global pattern registry.
// Usage: NewPatternWithReject("url", "^https?://", "localhost", "must be a valid public URL")
// Then use in struct tags: validate:"pattern=url"
func NewPatternWithReject(name, pattern, rejectPattern, errorMessage string) {
	registry.RegisterWithReject(name, pattern, rejectPattern, errorMessage)
}

// GetPatternValidator returns the global pattern validator that should be registered
// in ValidateStruct. This validator handles all validate:"pattern=name" tags.
func GetPatternValidator() rules.CustomValidateFunc {
	fn := validator.Func(func(fl validator.FieldLevel) bool {
		patternName := fl.Param()

		return registry.match(patternName, fl.Field().String())
	})

	return rules.CustomValidateFunc{
		Tag:  "pattern",
		Func: fn,
	}
}
