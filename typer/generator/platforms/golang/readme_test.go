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
	ex := string(examples)
	for _, s := range samples {
		assert.Truef(t, strings.Contains(ex, s[1]) || strings.Contains(ex, indent.ReplaceAllString(s[1], "\t$1")),
			"README sample is not quoted from examples_test.go:\n%s", s[1])
	}
}
