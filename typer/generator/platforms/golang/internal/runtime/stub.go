// Package runtime holds the part of every generated Go file that does not
// depend on the tracking plan. The generator embeds runtime.go after its
// import block verbatim, so these tests exercise exactly the emitted code.
//
// This file stands in for the declarations runtime.go needs from the rest of
// the generated file (the templates); it is not embedded.
package runtime

import (
	"errors"
	"time"

	analytics "github.com/rudderlabs/analytics-go/v4"
)

var ErrInvalidValue = errors.New("ruddertyper: invalid value")

type Option func(*callOptions)

type callOptions struct {
	context      *analytics.Context
	integrations analytics.Integrations
	timestamp    time.Time
}

func rudderTyperContext() map[string]any {
	return map[string]any{"platform": "go"}
}
