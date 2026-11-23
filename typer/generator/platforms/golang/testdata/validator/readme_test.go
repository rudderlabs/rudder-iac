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
		fences  = regexp.MustCompile("(?m)^[ \t]*```(.*)$").FindAllStringSubmatch(string(readme), -1)
		indent  = regexp.MustCompile(`(?m)^(.)`)
		source  = string(examples)
	)
	require.NotEmpty(t, samples)
	// Only ```go fences are checked, so an unlabelled fence would hide a Go
	// sample from this test. Other languages, such as bash, are allowed.
	require.Zero(t, len(fences)%2, "a README fence is never closed")
	for i := 0; i < len(fences); i += 2 {
		assert.NotEmptyf(t, strings.TrimSpace(fences[i][1]), "README fence %d must name its language", i/2+1)
	}
	for _, s := range samples {
		assert.Truef(t, strings.Contains(source, "\n"+s[1]) || strings.Contains(source, "\n"+indent.ReplaceAllString(s[1], "\t$1")),
			"README sample is not quoted from examples_test.go:\n%s", s[1])
	}
}
