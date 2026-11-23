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
	// Only ```go fences are checked, so a Go sample under any other label,
	// or none, would hide from this test. Other fences must name a language
	// that is clearly not Go.
	notGo := map[string]bool{"bash": true, "sh": true, "yaml": true, "json": true, "text": true}
	require.Zero(t, len(fences)%2, "a README fence is never closed")
	for i := 0; i < len(fences); i += 2 {
		label := strings.TrimSpace(fences[i][1])
		assert.Truef(t, label == "go" || notGo[label],
			"README fence %d is labelled %q: use go for a Go sample, or one of bash, sh, yaml, json, text", i/2+1, label)
	}
	for _, s := range samples {
		assert.Truef(t, strings.Contains(source, "\n"+s[1]) || strings.Contains(source, "\n"+indent.ReplaceAllString(s[1], "\t$1")),
			"README sample is not quoted from examples_test.go:\n%s", s[1])
	}
}
