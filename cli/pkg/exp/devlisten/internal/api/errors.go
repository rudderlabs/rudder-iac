package api

import (
	"fmt"
	"net/http"
)

// apiError is the contract section 4.8 error object.
type apiError struct {
	status  int
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Param   *string        `json:"param"`
	Details map[string]any `json:"details"`
	Next    *string        `json:"next"`
}

var errShuttingDown = apiError{status: http.StatusServiceUnavailable, Code: "shutting_down", Message: "the server is shutting down"}

func invalidParam(param, format string, args ...any) *apiError {
	return &apiError{status: http.StatusBadRequest, Code: "invalid_parameter",
		Message: param + ": " + fmt.Sprintf(format, args...), Param: &param}
}

func writeError(w http.ResponseWriter, e apiError) {
	writeJSON(w, e.status, map[string]apiError{"error": e})
}
