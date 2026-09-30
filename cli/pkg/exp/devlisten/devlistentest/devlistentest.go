// Package devlistentest starts a devlisten server for a Go test.
package devlistentest

import (
	"context"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const closeTimeout = 5 * time.Second

// Start starts a server on a free loopback port and closes it in
// t.Cleanup. It uses context.Background because t.Context is cancelled
// before cleanups run, which would close the server early.
func Start(t testing.TB, opts ...devlisten.Option) *devlisten.Server {
	t.Helper()
	s, err := devlisten.Start(context.Background(), opts...)
	if err != nil {
		t.Fatalf("starting dev listen: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if err := s.Close(ctx); err != nil {
			t.Errorf("closing dev listen: %v", err)
		}
	})
	return s
}
