//go:build unix

package providers_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The umask is process wide, so these subtests must not run in parallel.
func TestWriteSnapshotNewFileFollowsTheUmask(t *testing.T) {
	for name, tc := range map[string]struct {
		umask int
		want  os.FileMode
	}{
		"umask 077 keeps the snapshot private": {umask: 0o077, want: 0o600},
		"umask 022 makes it world readable":    {umask: 0o022, want: 0o644},
	} {
		t.Run(name, func(t *testing.T) {
			previous := syscall.Umask(tc.umask)
			t.Cleanup(func() { syscall.Umask(previous) })

			var (
				store = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
				path  = filepath.Join(t.TempDir(), "plan.snapshot.json")
			)

			_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, path)
			require.NoError(t, err)

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, info.Mode().Perm())
		})
	}
}
