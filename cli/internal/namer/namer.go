package namer

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"sync"

	"github.com/google/uuid"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core" // Import for NameRegistry
)

var (
	StrategyKebabCase = NewKebabCase()
	StrategySnakeCase = NewSnakeCase()
	StrategyCamelCase = NewCamelCase()
)

var ErrDuplicateNameException = errors.New("duplicate name exception")

type Namer interface {
	Name(input ScopeName) (string, error)
	Load([]ScopeName) error
}

type NamingStrategy interface {
	Name(input string) string
}

type ExternalIdNamer struct {
	*core.NameRegistry
	strategy NamingStrategy
	mu       sync.Mutex
}

func NewExternalIdNamer(strategy NamingStrategy) Namer {
	registry := core.NewNameRegistry(collisionHandler)
	return &ExternalIdNamer{
		NameRegistry: registry,
		strategy:     strategy,
	}
}

type NamingOptions struct {
	Strategy NamingStrategy
}

func WithStrategy(strategy NamingStrategy) NamingOptions {
	return NamingOptions{
		Strategy: strategy,
	}
}

func (p *ExternalIdNamer) Name(input ScopeName) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	baseName := p.strategy.Name(input.Name)
	// The reason we are generating id uniquely everytime we register name is
	// as we just need to make sure that the name is unique within the scope.
	registered, err := p.RegisterName(uuid.New().String(), input.Scope, baseName)
	if err != nil {
		return "", fmt.Errorf("registering name: %s errored with: %w", baseName, err)
	}

	return registered, nil
}

type ScopeName struct {
	Scope string
	Name  string
}

func (p *ExternalIdNamer) Load(names []ScopeName) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, name := range names {
		registered, err := p.RegisterName(uuid.New().String(), name.Scope, name.Name)
		if err != nil {
			return err
		}
		if registered != name.Name {
			return fmt.Errorf("loading name: %s errored with: %w", name.Name, ErrDuplicateNameException)
		}
	}
	return nil
}

func collisionHandler(name string, existingNames []string) string {
	baseName := name
	counter := 1
	for {
		candidate := fmt.Sprintf("%s-%d", baseName, counter)
		if !slices.Contains(existingNames, candidate) {
			return candidate
		}
		counter++
	}
}

type KebabCase struct{}

func NewKebabCase() NamingStrategy {
	return &KebabCase{}
}

// Name provides a safe repeatable way to convert input to kebab-case
// Example: "User Signed Up" -> "user-signed-up"
func (s *KebabCase) Name(input string) string {
	var result strings.Builder
	input = strings.ToLower(input)

	for i, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			continue
		}
		// replace any character that is not a letter or a digit with a hyphen
		// unless it's already a hyphen
		if i > 0 && result.Len() > 0 && result.String()[result.Len()-1] != '-' {
			result.WriteRune('-')
		}
	}

	return strings.Trim(result.String(), "-")
}

func NewSnakeCase() NamingStrategy {
	return &SnakeCase{}
}

type SnakeCase struct{}

func (s *SnakeCase) Name(input string) string {
	var result strings.Builder

	for i, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// Insert underscore before uppercase letters (except at the start)
			if i > 0 && unicode.IsUpper(r) && result.Len() > 0 {
				lastChar := result.String()[result.Len()-1]
				if lastChar != '_' {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
			continue
		}
		// Replace non-alphanumeric with underscore
		if i > 0 && result.Len() > 0 && result.String()[result.Len()-1] != '_' {
			result.WriteRune('_')
		}
	}
	return strings.Trim(result.String(), "_")
}

func NewCamelCase() NamingStrategy {
	return &CamelCase{}
}

type CamelCase struct{}

// Name provides a safe repeatable way to convert input to camelCase
// Example: "User Signed Up" -> "userSignedUp"
func (c *CamelCase) Name(input string) string {
	if input == "" {
		return ""
	}

	var result strings.Builder
	capitalizeNext := false
	isFirstWord := true

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capitalizeNext {
				result.WriteRune(unicode.ToUpper(r))
				capitalizeNext = false
			} else if isFirstWord {
				result.WriteRune(unicode.ToLower(r))
			} else {
				result.WriteRune(r)
			}
			isFirstWord = false
		} else {
			if result.Len() > 0 {
				capitalizeNext = true
			}
		}
	}

	return result.String()
}

// IDCandidate is one remote resource awaiting a local identifier.
type IDCandidate struct {
	// Key is the caller's handle for the resource, usually its remote ID. It
	// keys the returned map.
	Key string
	// Name is the display name a generated identifier is derived from.
	Name string
	// ExternalID is the identifier the resource already carries upstream. When
	// set it is kept verbatim: a resource already under management must not be
	// renamed, or a cloned project reads as a rename on the next apply.
	ExternalID string
}

// ResolveIDs assigns a local identifier to every candidate within one scope.
//
// Upstream ExternalIDs are reserved up front, before anything is generated, so
// a name derived from a display name can never collide with one — order
// matters here, which is why this lives in one place rather than in each
// handler.
func ResolveIDs(n Namer, scope string, candidates []IDCandidate) (map[string]string, error) {
	for _, c := range candidates {
		if c.ExternalID == "" {
			continue
		}
		// Reserved one at a time, tolerating a name the caller already loaded
		// — a project's own IDs are preloaded before import, and for a managed
		// resource those are these same ExternalIDs. Either way the identifier
		// is now taken, which is all the reservation is for: the collision
		// handler suffixes generated names, so none can land on it.
		err := n.Load([]ScopeName{{Name: c.ExternalID, Scope: scope}})
		if err != nil && !errors.Is(err, ErrDuplicateNameException) {
			return nil, fmt.Errorf("reserving upstream externalID %q in scope %q: %w", c.ExternalID, scope, err)
		}
	}

	ids := make(map[string]string, len(candidates))
	for _, c := range candidates {
		if c.ExternalID != "" {
			ids[c.Key] = c.ExternalID
			continue
		}
		generated, err := n.Name(ScopeName{Name: c.Name, Scope: scope})
		if err != nil {
			return nil, fmt.Errorf("generating externalID for %q in scope %q: %w", c.Name, scope, err)
		}
		ids[c.Key] = generated
	}

	return ids, nil
}
