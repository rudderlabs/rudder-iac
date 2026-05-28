package resolver

import "errors"

var (
	// ErrNotFound is returned when a backing source for a resolver is missing
	// (e.g. the variable file does not exist on disk).
	ErrNotFound = errors.New("not found")

	// ErrIllegalArgument is returned when a resolver receives input it cannot
	// process — for example, a variable file that fails to parse or whose
	// contents do not match the expected scalar shape.
	ErrIllegalArgument = errors.New("illegal argument")
)
