package core

import (
	"fmt"
	"slices"
)

// CollisionHandler defines a function type that handles name collisions.
// It takes the name that caused the collision and a list of existing names, and returns a new name.
type CollisionHandler func(name string, existingNames []string) string

// NameRegistry keeps track of all names that generators create for different kinds of code constructs.
// It handles name collisions gracefully by using collision handlers and scoped name tracking.
type NameRegistry struct {
	collisionHandler CollisionHandler
	// scopes maps scope -> name -> id
	scopes map[string]map[string]string
	// registrations maps scope -> id -> name
	registrations map[string]map[string]string
}

// NewNameRegistry creates a new NameRegistry with the provided collision handler.
func NewNameRegistry(handler CollisionHandler) *NameRegistry {
	return &NameRegistry{
		collisionHandler: handler,
		scopes:           make(map[string]map[string]string),
		registrations:    make(map[string]map[string]string),
	}
}

// RegisterName registers a name in the registry under a given scope/id, applying the collision handler if necessary.
// If a name is already registered for the given scope/id, the registered name is returned.
// If the id is not registered, and there is a collision in the given scope, the collision handler is applied to generate a new name.
// The function returns the final registered name.
func (nr *NameRegistry) RegisterName(id string, scope string, name string) (string, error) {
	if id == "" || scope == "" || name == "" {
		return "", fmt.Errorf("id, scope, and name must not be empty")
	}

	// Initialize scope maps if they don't exist
	if nr.scopes[scope] == nil {
		nr.scopes[scope] = make(map[string]string)
	}
	if nr.registrations[scope] == nil {
		nr.registrations[scope] = make(map[string]string)
	}

	// If this id is already registered in this scope, return the existing name
	if existingName, exists := nr.registrations[scope][id]; exists {
		return existingName, nil
	}

	// Check if the name already exists in this scope
	if existingId, exists := nr.scopes[scope][name]; exists && existingId != id {
		// Name collision - apply collision handler
		existingNames := nr.getExistingNames(scope)
		name = nr.collisionHandler(name, existingNames)
	}

	// Register the name
	nr.scopes[scope][name] = id
	nr.registrations[scope][id] = name

	return name, nil
}

// getExistingNames returns all existing names in the given scope
func (nr *NameRegistry) getExistingNames(scope string) []string {
	if scopeMap, exists := nr.scopes[scope]; exists {
		names := make([]string, 0, len(scopeMap))
		for name := range scopeMap {
			names = append(names, name)
		}
		return names
	}
	return []string{}
}

// DefaultCollisionHandler provides a simple collision handler that appends a number to the name
func DefaultCollisionHandler(name string, existingNames []string) string {
	baseName := name
	counter := 1

	for {
		candidate := fmt.Sprintf("%s%d", baseName, counter)
		if !slices.Contains(existingNames, candidate) {
			return candidate
		}
		counter++
	}
}
