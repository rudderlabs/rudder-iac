package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeparator(t *testing.T) {
	assert.Equal(t, "===", Separator("=", 3))
	assert.Equal(t, "---", Separator("-", 3))
	assert.Equal(t, "", Separator("=", 0))
	assert.Equal(t, "", Separator("=", -1))
}

func TestPadColumns(t *testing.T) {
	// Basic padding
	result := PadColumns("hello", "world", 20)
	assert.Equal(t, "hello               world", result)

	// When left text exceeds rightCol, minimum 1 space padding
	result = PadColumns("a very long left text", "right", 5)
	assert.Equal(t, "a very long left text right", result)

	// ANSI-colored left text uses visual width, not byte length
	colored := Color("hello", ColorRed)
	result = PadColumns(colored, "world", 20)
	// Visual width of "hello" is 5, so padding should be 15 spaces
	assert.Contains(t, result, "world")
	assert.Equal(t, colored+"               world", result)
}

func TestSectionHeader(t *testing.T) {
	result := SectionHeader("TITLE", "=", 10)
	assert.Equal(t, Bold("TITLE")+"\n"+"==========", result)
}
