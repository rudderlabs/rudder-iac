package parser

import "strings"

// importCandidate is a cleaned, single-statement line that contains a real import statement.
// line is 1-based and refers to the first line of the original statement.
type importCandidate struct {
	text string
	line int
}

// scanImportCandidates walks Python source code character-by-character, safely skipping
// string literals (including all prefix variants: r, f, b, u, rb, br, rf, fr) and # comments,
// then emits every logical statement that starts with "import" or "from" as a candidate.
//
// It handles:
//   - Triple-quoted strings (""" / ''') with any string prefix
//   - Single/double-quoted strings with escape sequences
//   - Inline # comments
//   - Implicit line continuation inside parentheses/brackets/braces
//   - Explicit backslash line continuation
//   - Semicolon-separated statements on a single line
//   - One-line suites (if cond: import x → import x)
func scanImportCandidates(code string) []importCandidate {
	var candidates []importCandidate

	runes := []rune(code)
	n := len(runes)

	var buf strings.Builder
	parenDepth := 0
	line := 1
	stmtLine := 1 // line where the current statement started
	i := 0

	for i < n {
		ch := runes[i]

		// String prefix detection (r, f, b, u, rb, br, rf, fr, ...).
		// Must be checked before quote handling so the prefix chars are consumed
		// without entering the buffer, and the string body is properly skipped.
		if isStringStart(runes, i, n) {
			i = skipString(runes, i, n)
			continue
		}

		switch {
		case ch == '#':
			i = skipComment(runes, i, n)

		case ch == '\\' && i+1 < n && runes[i+1] == '\n':
			// Explicit backslash continuation: join with next line.
			i += 2
			line++
			buf.WriteRune(' ')

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

		case ch == ';' && parenDepth == 0:
			candidates = emitCandidate(candidates, buf.String(), stmtLine)
			buf.Reset()
			stmtLine = line
			i++

		case ch == '\n' && parenDepth == 0:
			candidates = emitCandidate(candidates, buf.String(), stmtLine)
			buf.Reset()
			line++
			stmtLine = line
			i++

		case ch == '\n' && parenDepth > 0:
			// Implicit continuation inside brackets: fold newline into space.
			buf.WriteRune(' ')
			line++
			i++

		default:
			buf.WriteRune(ch)
			i++
		}
	}

	// Flush any statement not terminated by a newline or semicolon.
	candidates = emitCandidate(candidates, buf.String(), stmtLine)

	return candidates
}

// emitCandidate converts a raw statement buffer into a candidate, if it is a valid
// import statement. It filters out false positives (identifiers or strings containing
// the word "import") and handles one-line suites (if cond: import x → import x).
func emitCandidate(candidates []importCandidate, raw string, line int) []importCandidate {
	text := strings.TrimSpace(raw)
	if text == "" || !strings.Contains(text, "import") {
		return candidates
	}

	// Extract the import statement from a one-line suite header (if cond: import x).
	if !strings.HasPrefix(text, "import ") && !strings.HasPrefix(text, "from ") {
		if idx := strings.LastIndex(text, ": "); idx != -1 {
			after := strings.TrimSpace(text[idx+2:])
			if strings.HasPrefix(after, "import ") || strings.HasPrefix(after, "from ") {
				text = after
			}
		}
	}

	// Only emit real import statements — prevents false positives from identifiers
	// like "importlib.load_module()" or assignments like "reimport = ...".
	if strings.HasPrefix(text, "import ") || strings.HasPrefix(text, "from ") {
		candidates = append(candidates, importCandidate{text: text, line: line})
	}

	return candidates
}

// isStringStart returns true when position i begins a Python string literal,
// with or without a prefix (r, f, b, u, rb, br, rf, fr and their uppercase variants).
//
// Invalid prefix combinations (bu, uf, fb, etc.) are rejected to avoid false-matching
// identifiers that happen to start with a prefix character (e.g. "buffer", "format").
func isStringStart(runes []rune, i, n int) bool {
	ch := runes[i]

	// Direct quote.
	if ch == '"' || ch == '\'' {
		return true
	}

	// Must not be mid-identifier — prefix chars preceded by an ident char are not prefixes.
	if !isStringPrefixChar(ch) || (i > 0 && isIdentChar(runes[i-1])) {
		return false
	}

	// Single prefix + quote (e.g. r", f', b").
	if i+1 < n && (runes[i+1] == '"' || runes[i+1] == '\'') {
		return true
	}

	// Double prefix + quote (e.g. rb", rf', br").
	if i+2 < n && isStringPrefixChar(runes[i+1]) && isValidDoublePrefix(ch, runes[i+1]) &&
		(runes[i+2] == '"' || runes[i+2] == '\'') {
		return true
	}

	return false
}

// skipString advances past an entire Python string literal starting at i,
// returning the position immediately after the closing quote.
// It handles triple-quoted and single-quoted strings, with or without prefix chars.
func skipString(runes []rune, i, n int) int {
	// Consume string prefix chars (r, f, b, u, etc.) without adding to buffer.
	for i < n && isStringPrefixChar(runes[i]) {
		i++
	}

	if i >= n {
		return i
	}

	quote := runes[i]

	// Triple-quoted string.
	if i+2 < n && runes[i+1] == quote && runes[i+2] == quote {
		i += 3 // skip opening triple quote
		for i < n {
			if runes[i] == '\\' {
				i += 2 // skip escape sequence (works for raw strings too)
				continue
			}
			if runes[i] == quote && i+2 < n && runes[i+1] == quote && runes[i+2] == quote {
				return i + 3 // skip closing triple quote
			}
			i++
		}
		return i
	}

	// Single-quoted string.
	i++ // skip opening quote
	for i < n {
		if runes[i] == '\\' {
			i += 2 // skip escape sequence
			continue
		}
		if runes[i] == quote {
			return i + 1 // skip closing quote
		}
		i++
	}
	return i
}

// skipComment advances past a # comment to the end of the line (leaving the \n for the
// main loop to handle as a statement boundary).
func skipComment(runes []rune, i, n int) int {
	for i < n && runes[i] != '\n' {
		i++
	}
	return i
}

// isValidDoublePrefix reports whether (a, b) form a valid Python 3 two-char string prefix.
// Valid pairs are: rb, br, rf, fr (and their uppercase variants).
// Invalid combos like bu, uf, fb are rejected to avoid false-matching identifiers.
func isValidDoublePrefix(a, b rune) bool {
	al := toLower(a)
	bl := toLower(b)
	return (al == 'r' && bl == 'b') || (al == 'b' && bl == 'r') ||
		(al == 'r' && bl == 'f') || (al == 'f' && bl == 'r')
}

func toLower(ch rune) rune {
	if ch >= 'A' && ch <= 'Z' {
		return ch + 32
	}
	return ch
}

func isStringPrefixChar(ch rune) bool {
	return ch == 'r' || ch == 'R' || ch == 'f' || ch == 'F' ||
		ch == 'b' || ch == 'B' || ch == 'u' || ch == 'U'
}

func isIdentChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}
