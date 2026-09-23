package resolver

import "errors"

var (
	// ErrVarFileNotFound is returned when the variable file does not exist on disk.
	ErrVarFileNotFound = errors.New("variable file not found")

	// ErrVarFileParseFailed is returned when the variable file cannot be parsed
	// as a flat YAML map of scalar values (e.g. invalid YAML, nested map/array,
	// or a nil value).
	ErrVarFileParseFailed = errors.New("variable file parse failed")

	// ErrVarFileInvalidName is returned when a variable file path does not end in
	// the required .vars.yaml or .vars.yml suffix. The suffix is mandatory for
	// every var file, including those passed via --var-file from outside the
	// project directory.
	ErrVarFileInvalidName = errors.New("variable file must use the .vars.yaml or .vars.yml suffix")
)
