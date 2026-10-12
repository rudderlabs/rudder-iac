package validator

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The README's samples must stay compiled: each is quoted from
// examples_test.go, from the start of a line, de-indented by one tab when it
// comes from inside a function. The test lives here, not in the generator
// package, so make typer-go-validate runs it on README-only changes.
func TestREADMESamplesAreQuotedFromExamples(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	require.NoError(t, err)
	examples, err := os.ReadFile("examples_test.go")
	require.NoError(t, err)

	var (
		samples = regexp.MustCompile("(?s)```go\n(.*?)```").FindAllStringSubmatch(string(readme), -1)
		indent  = regexp.MustCompile(`(?m)^(.)`)
		source  = string(examples)
	)
	require.NotEmpty(t, samples)
	// A fence not written as ```go would otherwise be skipped silently.
	assert.Equal(t, 2*len(samples), strings.Count(string(readme), "```"), "every README fence must open a ```go sample")
	for _, s := range samples {
		assert.Truef(t, strings.Contains(source, "\n"+s[1]) || strings.Contains(source, "\n"+indent.ReplaceAllString(s[1], "\t$1")),
			"README sample is not quoted from examples_test.go:\n%s", s[1])
	}
}
