package parser

import "strings"

// importCandidate is a cleaned, single-statement line that contains the word "import".
type importCandidate struct {
	text string
}

// scanImportCandidates walks Python source code character-by-character, safely skipping
// string literals (including all prefix variants: r, f, b, u, rb, br, etc.) and # comments,
// then emits every logical statement that contains the word "import" as a candidate.
//
// It handles:
//   - Triple-quoted strings (""" / ''') with any string prefix
//   - Single/double-quoted strings with escape sequences
//   - Inline # comments
//   - Implicit line continuation inside parentheses/brackets/braces
//   - Explicit backslash line continuation
//   - Semicolon-separated statements on a single line
func scanImportCandidates(code string) []importCandidate {
	var candidates []importCandidate

	runes := []rune(code)
	n := len(runes)

	var buf strings.Builder
	parenDepth := 0
	i := 0

	flushStatement := func() {
		stmt := strings.TrimSpace(buf.String())
		buf.Reset()
		if strings.Contains(stmt, "import") {
			candidates = append(candidates, importCandidate{text: stmt})
		}
	}

	for i < n {
		ch := runes[i]

		// --- String prefix detection (r, f, b, u, rb, br, rf, fr, ...) ---
		// A string prefix is one or two letters immediately before a quote character.
		// We detect it here so we can skip into the string body correctly.
		if isStringPrefixStart(runes, i, n) {
			// Consume prefix letters without adding them to buf
			for i < n && isStringPrefixChar(runes[i]) {
				i++
			}
			// Fall through to quote handling below
			ch = runes[i]
		}

		switch {
		// Triple-quoted string
		case (ch == '"' || ch == '\'') && i+2 < n && runes[i+1] == ch && runes[i+2] == ch:
			quote := ch
			i += 3 // skip opening triple quote
			for i < n {
				if runes[i] == '\\' {
					i += 2 // skip escaped char
					continue
				}
				if runes[i] == quote && i+2 < n && runes[i+1] == quote && runes[i+2] == quote {
					i += 3 // skip closing triple quote
					break
				}
				i++
			}

		// Single-quoted string
		case ch == '"' || ch == '\'':
			quote := ch
			i++ // skip opening quote
			for i < n {
				if runes[i] == '\\' {
					i += 2 // skip escaped char
					continue
				}
				if runes[i] == quote {
					i++ // skip closing quote
					break
				}
				i++
			}

		// Comment: skip to end of line
		case ch == '#':
			for i < n && runes[i] != '\n' {
				i++
			}

		// Backslash continuation: join with next line
		case ch == '\\' && i+1 < n && runes[i+1] == '\n':
			i += 2 // skip \ and newline — logical line continues
			buf.WriteRune(' ')

		// Open paren/bracket/brace: implicit continuation
		case ch == '(' || ch == '[' || ch == '{':
			parenDepth++
			buf.WriteRune(ch)
			i++

		case ch == ')' || ch == ']' || ch == '}':
			if parenDepth > 0 {
				parenDepth--
			}
			buf.WriteRune(ch)
			i++

		// Semicolon at depth 0: statement boundary
		case ch == ';' && parenDepth == 0:
			flushStatement()
			i++

		// Newline at depth 0: statement boundary
		case ch == '\n' && parenDepth == 0:
			flushStatement()
			i++

		// Newline inside parens: implicit continuation — replace with space
		case ch == '\n' && parenDepth > 0:
			buf.WriteRune(' ')
			i++

		default:
			buf.WriteRune(ch)
			i++
		}
	}

	// Flush any remaining content
	flushStatement()

	return candidates
}

// isStringPrefixStart returns true when position i is the start of a string prefix
// (e.g. r, f, b, u, rb, br, rf, fr) immediately followed by a quote character,
// but is NOT itself already inside a keyword or identifier (preceded by an alnum/_).
func isStringPrefixStart(runes []rune, i, n int) bool {
	ch := runes[i]
	if !isStringPrefixChar(ch) {
		return false
	}
	// Must not be mid-identifier
	if i > 0 && isIdentChar(runes[i-1]) {
		return false
	}
	// Look ahead: up to 2 prefix chars followed by a quote
	j := i
	for j < n && j-i < 2 && isStringPrefixChar(runes[j]) {
		j++
	}
	return j < n && (runes[j] == '"' || runes[j] == '\'')
}

func isStringPrefixChar(ch rune) bool {
	return ch == 'r' || ch == 'R' || ch == 'f' || ch == 'F' ||
		ch == 'b' || ch == 'B' || ch == 'u' || ch == 'U'
}

func isIdentChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}
