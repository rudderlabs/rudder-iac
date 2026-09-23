package client

import (
	"net/http"
	"strings"
)

// accountInUseMessages are the refusals the control plane raises when an
// account still backs something. Matching on prose is fragile, but it is the
// only signal available: ConflictError carries a message and a status and
// nothing else, so APIError.ErrorCode is empty on this path.
//
//	AccountService.deleteAccountByIdAndWorkspaceId   the public accounts API
//	AccountService.deleteAccountByIdAndWSLegacy      the webapp's legacy routes
//
// The two services word the same refusal differently ("used by" vs "used in"),
// so the shared prefix is what is matched.
var accountInUseMessages = []string{
	"can't be removed because it is being used",
}

// BlockedByAccountUsage reports whether this is the control plane refusing to
// delete an account because sources, destinations or data graphs still use it.
func (e *APIError) BlockedByAccountUsage() bool {
	if e.HTTPStatusCode != http.StatusConflict && e.HTTPStatusCode != http.StatusBadRequest {
		return false
	}

	msg := strings.ToLower(e.Msg())
	for _, want := range accountInUseMessages {
		if strings.Contains(msg, want) {
			return true
		}
	}

	return false
}
