package golang

import (
	"strings"
	"unicode"
)

// commentLines splits text into lines that are safe after "//". go/scanner
// rejects invalid UTF-8, NUL and U+FEFF even inside comments, which would
// fail the go/format pass; quotes, backslashes and "*/" need no escaping.
func commentLines(text string) []string {
	text = strings.ToValidUTF8(text, "\uFFFD")
	text = strings.NewReplacer("\x00", "", "\uFEFF", "").Replace(text)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
	}
	return lines
}

// comment renders text as "//" line comments, one per line of text.
func comment(text string) string {
	lines := commentLines(text)
	for i, line := range lines {
		if line == "" {
			lines[i] = "//"
			continue
		}
		lines[i] = "// " + line
	}
	return strings.Join(lines, "\n")
}

// inlineComment renders text for the middle of a single comment line.
func inlineComment(text string) string {
	return strings.Join(commentLines(text), " ")
}
