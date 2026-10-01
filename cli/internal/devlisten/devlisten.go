// Package devlisten runs a local listener that answers RudderStack SDKs and
// keeps what they send for review.
package devlisten

import (
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/api"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

// NewHandler returns the handler that answers SDKs and the store it captures
// into. An empty writeKeys accepts every key. version is what /version
// reports.
func NewHandler(writeKeys []string, version string) (*ingest.Handler, *store.Store) {
	s := store.New()
	return ingest.New(s, writeKeys, version), s
}

// IsLoopback reports whether a bind address takes local connections only.
func IsLoopback(bind string) bool {
	return api.IsLoopback(bind)
}

// The read commands check their flags with the rules of the query API, so a
// bad value fails before any request.
const MaxLimit = api.MaxLimit

// ParseSince reads a since value: a cursor, a duration back from now, or an
// RFC 3339 time.
func ParseSince(value string, now time.Time) error {
	_, err := api.ParseSince(value, now)
	return err
}

// ParseWait reads a wait value with the bounds of the query API.
func ParseWait(value string) (time.Duration, error) {
	return api.ParseWait(value)
}

// CheckFieldPath rejects a comma list or an empty path segment.
func CheckFieldPath(path string) error {
	return api.CheckFieldPath(path)
}
