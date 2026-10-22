// Package specdocs builds the reference documents `rudder-cli docs` prints.
// Every document is read from the code that validates specs, never from
// hand-written prose, so it describes exactly what the installed binary
// accepts. The JSON shapes are a contract: version 1 only gains optional
// fields, and removing or renaming a field bumps SchemaVersion.
package specdocs

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// SchemaVersion is the version of every JSON document this package emits.
const SchemaVersion = 1

// Error codes of an ErrorDoc. A caller branches on the code; a new code is an
// optional addition under SchemaVersion 1.
const (
	// CodeUnknownType: the destination type is not compiled in.
	CodeUnknownType = "unknown_type"
	// CodeUnknownSource: --source is not a source spec type.
	CodeUnknownSource = "unknown_source"
	// CodeUnsupportedSource: the destination type does not accept --source.
	CodeUnsupportedSource = "unsupported_source"
	// CodeInvalidID: --id would write an invalid display_name or variable name.
	CodeInvalidID = "invalid_id"
	// CodeUnknownDefinitionVersion: the type has no such definition version.
	CodeUnknownDefinitionVersion = "unknown_definition_version"
	// CodeMissingType: --id or --definition-version was given without a type,
	// or the type argument is empty.
	CodeMissingType = "missing_type"
	// CodeInvalidArguments: the command got more arguments than it takes, a
	// --format other than md or json, or --format and --json that disagree.
	CodeInvalidArguments = "invalid_arguments"
)

// Error is an input the documents cannot be built for. It carries what a
// caller needs to correct the input without a second lookup.
type Error struct {
	Code          string
	Message       string
	Suggestions   []string
	ValidTypes    []string
	ValidVersions []int64
}

func (e *Error) Error() string {
	return e.Message
}

// ErrorDoc is the JSON document printed instead of the requested one when the
// input is wrong.
type ErrorDoc struct {
	SchemaVersion int       `json:"schemaVersion"`
	CLIVersion    string    `json:"cliVersion"`
	Error         ErrorBody `json:"error"`
}

// ErrorBody says what was wrong and how to fix it. ValidTypes lists the
// values accepted for the input the code names: destination types for
// unknown_type and missing_type, source spec types for unknown_source, and
// the source spec types the destination accepts for unsupported_source.
// ValidVersions lists the type's definition versions for
// unknown_definition_version.
type ErrorBody struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Suggestions   []string `json:"suggestions"`
	ValidTypes    []string `json:"validTypes,omitempty"`
	ValidVersions []int64  `json:"validVersions,omitempty"`
}

// NewErrorDoc wraps e for JSON output.
func NewErrorDoc(cliVersion string, e *Error) ErrorDoc {
	suggestions := e.Suggestions
	if suggestions == nil {
		suggestions = []string{}
	}
	return ErrorDoc{
		SchemaVersion: SchemaVersion,
		CLIVersion:    cliVersion,
		Error: ErrorBody{
			Code:          e.Code,
			Message:       e.Message,
			Suggestions:   suggestions,
			ValidTypes:    e.ValidTypes,
			ValidVersions: e.ValidVersions,
		},
	}
}

// WriteJSON writes v as indented JSON. HTML escaping is off because variable
// names carry "<N>" and patterns carry "&" and "<", which must read as written.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

// unverifiedEnable lists the environment variables that register the
// unverified destination definitions.
func unverifiedEnable() map[string]string {
	return map[string]string{
		config.ExperimentalEnvVar:                                   "true",
		config.GetEnvironmentVariableName("unverifiedDestinations"): "true",
	}
}
