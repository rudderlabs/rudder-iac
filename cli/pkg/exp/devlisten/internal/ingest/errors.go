package ingest

import (
	"net/http"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// gwError is one oracle error response: the message comes from rudder-server
// gateway/response/response.go v1.85.0, the status from SVC
// internal/gateway/response/response.go. stage names the rejection stage of
// the capture record (contract section 3).
type gwError struct {
	status  int
	message string
	stage   string
}

var (
	errUncompress            = gwError{http.StatusBadRequest, "failed to uncompress request body", "decode"}
	errRequestBodyTooLarge   = gwError{http.StatusRequestEntityTooLarge, "request size exceeds max limit", "size"}
	errNoWriteKeyInBasicAuth = gwError{http.StatusUnauthorized, "failed to read writekey from header", "auth"}
	errNoWriteKeyInQuery     = gwError{http.StatusUnauthorized, "failed to read writekey from query params", "auth"}
	errInvalidWriteKey       = gwError{http.StatusUnauthorized, "invalid write key", "auth"}
	errRequestBodyNil        = gwError{http.StatusBadRequest, "request body is nil", "body"}
	errRequestBodyReadFailed = gwError{http.StatusBadRequest, "failed to read body from request", "body"}
	errInvalidJSON           = gwError{http.StatusBadRequest, "invalid json", "parse"}
	errNotRudderEvent        = gwError{http.StatusBadRequest, "event is not a valid rudder event", "parse"}
	errEmptyBatch            = gwError{http.StatusBadRequest, "empty batch payload", "batch"}
	errTooManyEventsInBatch  = gwError{http.StatusRequestEntityTooLarge, "batch exceeds the maximum allowed number of events", "batch"}
	errNonIdentifiable       = gwError{http.StatusBadRequest, "request neither has anonymousId nor userId", "identity"}
	errUnknownPath           = gwError{http.StatusNotFound, "unknown path", ""}
)

// reject writes the error the way the oracle does, with http.Error
// (SVC internal/gateway/gateway.go:660-667).
func (rep reply) reject(e gwError) reply {
	rep.status = e.status
	rep.header.Set("Content-Type", "text/plain; charset=utf-8")
	rep.header.Set("X-Content-Type-Options", "nosniff")
	rep.body = []byte(e.message + "\n")
	if e.stage != "" {
		rep.rejection = &store.Rejection{Stage: e.stage, Reason: e.message}
	}
	return rep
}

func (rep reply) rejectEvent(e gwError, idx int) reply {
	rep = rep.reject(e)
	rep.rejection.Idx = &idx
	return rep
}
