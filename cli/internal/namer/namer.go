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

var ErrDuplicateNameException = errors.New("duplicate name exception")

const ExternalIdNamerScope = "externalIds"

// Namer interface provides methods to generate unique names based on strategy and load existing names.
type Namer interface {
	Name(input string) (string, error)
	Load([]string) error
}

// NamingStrategy interface defines how to transform input into a base name.
type NamingStrategy interface {
	Name(input string) string
}

// ExternalIdNamer implements Namer by composing NameRegistry.
type ExternalIdNamer struct {
	*core.NameRegistry
	strategy NamingStrategy
	scope    string
	mu       sync.Mutex
}

// NewNamer creates a new Namer instance using the provided strategy.
func NewExternalIdNamer(strategy NamingStrategy) Namer {
	registry := core.NewNameRegistry(collisionHandler)
	return &ExternalIdNamer{
		NameRegistry: registry,
		strategy:     strategy,
		scope:        ExternalIdNamerScope,
	}
}

// Name generates a unique name using the strategy and handles collisions.
func (p *ExternalIdNamer) Name(input string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	baseName := p.strategy.Name(input)

	registered, err := p.RegisterName(uuid.New().String(), p.scope, baseName)
	if err != nil {
		return "", fmt.Errorf("registering name: %s errored with: %w", baseName, err)
	}

	return registered, nil
}

// Load adds existing names to the registry, returning error on duplicates.
func (p *ExternalIdNamer) Load(names []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, name := range names {
		registered, err := p.RegisterName(uuid.New().String(), p.scope, name)
		if err != nil {
			return err
		}
		if registered != name {
			return fmt.Errorf("loading name: %s errored with: %w", name, ErrDuplicateNameException)
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

// KebabCase implements NamingStrategy for kebab-case naming.
type KebabCase struct{}

// NewKebabCase returns a new KebabCase strategy.
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
