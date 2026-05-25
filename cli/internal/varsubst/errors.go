package varsubst

import "errors"

var (
	ErrUndefinedVariable  = errors.New("undefined variable")
	ErrInvalidVarSyntax   = errors.New("invalid variable syntax")
	ErrVarFileNotFound    = errors.New("variable file not found")
	ErrVarFileParseFailed = errors.New("variable file parse failed")
)

// SubstitutionError holds context for variable substitution failures.
type SubstitutionError struct {
	Name     string
	Line     int
	Column   int
	LineText string
	Err      error
}

func (e *SubstitutionError) Error() string {
	if e == nil {
		return ""
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return "substitution error"
}

func (e *SubstitutionError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}
