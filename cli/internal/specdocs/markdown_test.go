package specdocs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeSpanFencesBackticks(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "`plain`", code("plain"))
	assert.Equal(t, "``a`b``", code("a`b"))
	assert.Equal(t, "`` `edge ``", code("`edge"))
}

// A pipe inside a cell, even inside a code span, ends the cell unless escaped.
func TestMarkdownRowEscapesPipes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := &printer{w: &buf}
	p.row(code("a|b"), "c")
	require.NoError(t, p.err)
	assert.Equal(t, "| `a\\|b` | c |\n", buf.String())
}

func TestRequiredCellReadsEveryRule(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "yes", requiredCell(Field{Required: true}))
	assert.Equal(t, "yes, once its parent is written", requiredCell(Field{RequiredWhenParentPresent: true}))
	assert.Equal(t,
		"if `a` is `true` and `b` is `x`; unless `c` is `y`; if `d` or `e` is set; not with `f`; checked by `g_required`",
		requiredCell(Field{
			RequiredIf:     []Condition{{Key: "a", Value: true}, {Key: "b", Value: "x"}},
			RequiredUnless: []Condition{{Key: "c", Value: "y"}},
			RequiredWith:   []string{"d", "e"},
			ExcludedWith:   []string{"f"},
			CheckedBy:      []string{"g_required"},
		}),
	)
}
