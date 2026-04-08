package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintDeprecationWarning(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	SetWriter(&buf)
	t.Cleanup(RestoreWriter)

	PrintDeprecationWarning("line one\nline two")

	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "\n"), "expected leading blank line")
	assert.Contains(t, out, "Warning:")
	assert.Contains(t, out, "line one")
	assert.Contains(t, out, "line two")
	assert.True(t, strings.HasSuffix(out, "\n\n") || strings.HasSuffix(out, "\n"), "expected trailing newline(s)")
}
