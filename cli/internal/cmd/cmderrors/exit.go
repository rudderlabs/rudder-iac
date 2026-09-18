package cmderrors

import (
	"errors"
	"net/http"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
)

// Exit codes. A caller — a CI step, or an agent driving the CLI — needs to know
// whether to fix the specs and retry, re-authenticate, or back off and try
// again later. Prose on stderr cannot be branched on; these can.
//
// 1 stays the unclassified failure so nothing that already treats "non-zero" or
// "exactly 1" as failure regresses; the classified codes are additive.
const (
	ExitOK         = 0
	ExitError      = 1 // unclassified failure
	ExitValidation = 2 // the project's specs are wrong — fix them and retry
	ExitAuth       = 3 // credentials missing, expired, or lacking permission
	ExitUpstream   = 4 // the API rejected or could not serve the request
)

// ExitCode classifies err into one of the codes above.
//
// Classification is by error type and HTTP status only, never by matching
// message text: messages get reworded, and a caller branching on a code we
// derived from prose would break silently when they do.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	if errors.Is(err, project.ErrValidationFailed) {
		return ExitValidation
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return ExitAuth
		default:
			return ExitUpstream
		}
	}

	// The pre-flight "no token configured" failure never reaches the API, so it
	// never produces an APIError to classify.
	if errors.Is(err, app.ErrNotAuthenticated) {
		return ExitAuth
	}

	return ExitError
}
