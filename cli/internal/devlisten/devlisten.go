// Package devlisten runs a local listener that answers RudderStack SDKs and
// keeps what they send for review.
package devlisten

import (
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
