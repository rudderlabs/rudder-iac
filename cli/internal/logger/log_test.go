package logger

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// An unwritable HOME must not stop the CLI: --help and usage errors run
// before any command needs the log (DEX-1017).
func TestOpenLogDiscardsWhenHomeIsNotWritable(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	require.NoError(t, os.Chmod(home, 0o500))
	t.Cleanup(func() { _ = os.Chmod(home, 0o700) })

	w := openLog(home)

	require.Equal(t, io.Discard, w)
	_, err := os.Stat(filepath.Join(home, ".rudder"))
	require.True(t, os.IsNotExist(err))
}
