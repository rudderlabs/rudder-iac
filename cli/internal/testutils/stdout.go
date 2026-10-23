package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureStdout runs fn with os.Stdout pointed at a file and returns what was
// written there, including by code that captured os.Stdout or prints with
// fmt.Println. Tests using it must not run in parallel, since os.Stdout is
// process state.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)

	original := os.Stdout
	os.Stdout = f
	func() {
		defer func() { os.Stdout = original }()
		fn()
	}()

	require.NoError(t, f.Close())
	out, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	return string(out)
}
