package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextFormatter_Format_Success(t *testing.T) {
	t.Parallel()

	formatter := TextFormatter{}

	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "simple javascript code",
			input:    "export function transformEvent(event) { return event; }",
			expected: []byte("export function transformEvent(event) { return event; }"),
		},
		{
			name:     "python code with newlines",
			input:    "def transform_event(event):\n    return event",
			expected: []byte("def transform_event(event):\n    return event"),
		},
		{
			name:     "empty string",
			input:    "",
			expected: []byte(""),
		},
		{
			name:     "code with special characters",
			input:    "const regex = /\\w+/; // comment\nconst str = \"hello\\nworld\";",
			expected: []byte("const regex = /\\w+/; // comment\nconst str = \"hello\\nworld\";"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := formatter.Format(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextFormatter_Format_InvalidType(t *testing.T) {
	t.Parallel()

	formatter := TextFormatter{}

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "integer input",
			input: 42,
		},
		{
			name:  "map input",
			input: map[string]string{"key": "value"},
		},
		{
			name:  "slice input",
			input: []string{"a", "b", "c"},
		},
		{
			name:  "nil input",
			input: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := formatter.Format(tt.input)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "expected string")
		})
	}
}

func TestTextFormatter_Extension(t *testing.T) {
	t.Parallel()

	formatter := TextFormatter{}
	extensions := formatter.Extension()

	assert.Equal(t, []string{"js", "py"}, extensions)
	assert.Len(t, extensions, 2)
	assert.Contains(t, extensions, "js")
	assert.Contains(t, extensions, "py")
}
