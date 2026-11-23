package varsubst

import (
	"errors"
	"fmt"
)

var (
	ErrUndefinedVariable = errors.New("undefined variable")
	ErrInvalidVarSyntax  = errors.New("invalid variable syntax")

	// ErrUnescapedSingleQuote reports a value that would end a single-quoted
	// YAML scalar early. Without this check the failure is a YAML parse error
	// far from the variable, or, for a doubled pair, a silently changed value.
	ErrUnescapedSingleQuote = errors.New("value contains a single quote inside a single-quoted slot; write each ' as '' in the variable")
)

type SubstitutionError struct {
	Name     string
	Line     int
	Column   int
	LineText string
	Err      error
}

func (e *SubstitutionError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("line %d, column %d: %s: %s", e.Line, e.Column, e.Err, e.Name)
	}
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Err)
}

func (e *SubstitutionError) Unwrap() error {
	return e.Err
}
