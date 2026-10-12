package golang_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The README's samples must stay compiled: each is quoted from the validator's
// examples_test.go, de-indented by one tab when it comes from inside a
// function.
func TestREADMESamplesAreQuotedFromExamples(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	require.NoError(t, err)
	examples, err := os.ReadFile("testdata/validator/examples_test.go")
	require.NoError(t, err)

	samples := regexp.MustCompile("(?s)```go\n(.*?)```").FindAllStringSubmatch(string(readme), -1)
	require.NotEmpty(t, samples)
	indent := regexp.MustCompile(`(?m)^(.)`)
	for _, s := range samples {
		var (
			sample   = s[1]
			indented = indent.ReplaceAllString(sample, "\t$1")
		)
		assert.Truef(t, strings.Contains(string(examples), sample) || strings.Contains(string(examples), indented),
			"README sample is not quoted from examples_test.go:\n%s", sample)
	}
}
