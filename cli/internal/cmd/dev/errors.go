package dev

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

// cliError is a failure the CLI detects itself. It prints as the same
// error object the server returns.
type cliError struct {
	Code    string
	Message string
	Param   string
	Details map[string]any
	Next    string
}

func (e *cliError) Error() string { return e.Code + ": " + e.Message }

func usageError(next, format string, args ...any) error {
	return &cliError{Code: "usage", Message: fmt.Sprintf(format, args...), Next: next}
}

const nextListen = "rudder-cli dev listen --detach"

// asAPIError maps any error to the section 4.8 object.
func asAPIError(err error) *devlisten.APIError {
	var apiErr *devlisten.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	var cliErr *cliError
	if errors.As(err, &cliErr) {
		return &devlisten.APIError{Code: cliErr.Code, Message: cliErr.Message, Param: cliErr.Param,
			Details: cliErr.Details, Next: cliErr.Next}
	}
	return &devlisten.APIError{Code: "server_unreachable", Message: err.Error(), Next: nextListen}
}
