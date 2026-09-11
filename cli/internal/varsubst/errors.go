package varsubst

import (
	"errors"
	"fmt"
)

var (
	ErrUndefinedVariable = errors.New("undefined variable")
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
