package api

import (
	"fmt"
	"net/http"
)

// apiError is the contract section 4.8 error object. Next is set on every
// error the server returns.
type apiError struct {
	status  int
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Param   *string        `json:"param"`
	Details map[string]any `json:"details"`
	Next    *string        `json:"next"`
}

const (
	nextInfo   = "rudder-cli dev info --json"
	nextListen = "rudder-cli dev listen --detach"
	nextShell  = "rudder-cli dev summary --json"
)

func strp(s string) *string { return &s }

var errShuttingDown = apiError{status: http.StatusServiceUnavailable, Code: "shutting_down",
	Message: "the server is shutting down", Next: strp(nextListen)}

// routeCommands maps a query route to the CLI command that calls it.
var routeCommands = map[string]string{
	"events":         "rudder-cli dev events list",
	"requests":       "rudder-cli dev requests list",
	"requests/{seq}": "rudder-cli dev requests show",
	"summary":        "rudder-cli dev summary",
	"info":           "rudder-cli dev info",
}

func helpNext(route string) *string {
	cmd, ok := routeCommands[route]
	if !ok {
		cmd = "rudder-cli dev"
	}
	return strp(cmd + " --help")
}

func invalidParam(route, param, format string, args ...any) *apiError {
	return &apiError{status: http.StatusBadRequest, Code: "invalid_parameter",
		Message: param + ": " + fmt.Sprintf(format, args...), Param: &param, Next: helpNext(route)}
}

func writeError(w http.ResponseWriter, e apiError) {
	writeJSON(w, e.status, map[string]apiError{"error": e})
}
