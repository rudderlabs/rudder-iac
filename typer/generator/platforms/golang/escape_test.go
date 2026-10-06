package golang

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", "some string property", "// some string property"},
		{"empty", "", "//"},
		{"lines", "first line\nsecond line", "// first line\n// second line"},
		{"blank line", "first\n\nthird", "// first\n//\n// third"},
		{"trailing whitespace", "padded \t\nCRLF\r\nend", "// padded\n// CRLF\n// end"},
		{"leading whitespace is kept", "  indented", "//   indented"},
		{"quotes, backslashes and block comments need no escaping", `"quotes", backslash\path, /* comment */`, `// "quotes", backslash\path, /* comment */`},
		{"dollar signs", "$variable and ${expression}", "// $variable and ${expression}"},
		{"unicode", "café, naïve, 日本語 🎯", "// café, naïve, 日本語 🎯"},
		{"NUL is removed", "nul\x00byte", "// nulbyte"},
		{"byte order mark is removed", "\xef\xbb\xbfbom", "// bom"},
		{"invalid UTF-8 is replaced", "bad\xffbyte", "// bad\xef\xbf\xbdbyte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, comment(tt.input))
		})
	}
}

func TestInlineComment(t *testing.T) {
	assert.Equal(t, "https://example.com/plan 1", inlineComment("https://example.com/plan\n1\x00"))
}

// Event names reach the file twice: as a string literal sent on the wire and
// quoted in a doc comment. Both must survive go/format whatever the name holds.
func TestEventNameLiterals(t *testing.T) {
	tests := []struct {
		name    string
		event   string
		literal string
	}{
		{"quotes", `Product "Premium" Clicked`, `"Product \"Premium\" Clicked"`},
		{"backslash", `Path\To\Event`, `"Path\\To\\Event"`},
		{"newline and tab", "Line 1\nLine\t2", `"Line 1\nLine\t2"`},
		{"dollar signs", "$Variable$String", `"$Variable$String"`},
		{"unicode", "Café 日本語 🎯", `"Café 日本語 🎯"`},
		{"block comment", "Event */ Name", `"Event */ Name"`},
		{"NUL", "Event\x00Name", `"Event\x00Name"`},
		{"byte order mark", "Event\xef\xbb\xbfName", `"Event\ufeffName"`},
		{"invalid UTF-8", "Event\xffName", `"Event\xffName"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &plan.TrackingPlan{Rules: []plan.EventRule{{
				Event:   plan.Event{EventType: plan.EventTypeTrack, Name: tt.event},
				Section: plan.IdentitySectionProperties,
			}}}

			files, err := (&Generator{}).Generate(p, core.GenerateOptions{}, nil)
			require.NoError(t, err)
			require.Len(t, files, 1)

			assert.Contains(t, files[0].Content, "sends the track event "+tt.literal+", which has no properties.\n")
			assert.Contains(t, files[0].Content, "\treturn r.track(identity, "+tt.literal+", nil, opts)\n")
		})
	}
}
