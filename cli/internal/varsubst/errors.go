package varsubst

import (
	"errors"
	"fmt"
)

var (
	// The hint rides on the error itself because the usual cause is a var file
	// that was never passed, not a bad spec.
	ErrUndefinedVariable = errors.New("make sure to pass this variable through --var-file. undefined variable")
	ErrInvalidVarSyntax  = errors.New("invalid variable syntax")
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
