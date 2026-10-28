package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureStdout runs fn with os.Stdout pointed at a file and returns what was
// written there, by fmt.Println or by a writer that reads os.Stdout while fn
// runs, such as a renderer fn builds. A writer that kept os.Stdout before fn
// runs, such as one a package sets at init, still writes to the real stdout.
// Tests using it must not run in parallel, since os.Stdout is process state.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stdout, fn)
}

// CaptureStderr is CaptureStdout for os.Stderr.
func CaptureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stderr, fn)
}

// capture runs fn with *stream pointed at a file and returns what was written
// to it.
func capture(t *testing.T, stream **os.File, fn func()) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "output")
	require.NoError(t, err)

	original := *stream
	*stream = f
	func() {
		defer func() { *stream = original }()
		fn()
	}()

	require.NoError(t, f.Close())
	out, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	return string(out)
}
