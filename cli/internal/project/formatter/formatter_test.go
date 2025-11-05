package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatters_Format(t *testing.T) {
	formatters := Setup(DefaultYAML)

	tests := []struct {
		input         any
		extension     string
		expected      []byte
		errorExpected error
	}{
		{input: map[string]string{"key": "value"}, extension: "yaml", expected: []byte("key: \"value\"\n"), errorExpected: nil},
		{input: map[string]string{"key": "value"}, extension: "json", errorExpected: ErrUnsupportedExtension},
		{input: map[string]string{"key": "value"}, extension: "", errorExpected: ErrEmptyExtension},
	}
	for _, test := range tests {
		t.Run(test.extension, func(t *testing.T) {
			output, err := formatters.Format(test.input, test.extension)
			if test.errorExpected != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, test.errorExpected)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, string(test.expected), string(output))
		})
	}
}

func TestNormalizeExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: ".yaml", expected: "yaml"},
		{input: ".yml", expected: "yml"},
		{input: "YAMl", expected: "yaml"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expected, normalizeExtension(test.input))
		})
	}
}
