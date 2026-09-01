package varsubst

import "regexp"

// quotedVarRegex matches a {{ .VAR }} token directly enclosed in double
// quotes, capturing the token. Built from varRegex so the token grammar lives
// in one place.
var quotedVarRegex = regexp.MustCompile(`"(` + varRegex.String() + `)"`)

// QuoteTokensForYAMLParse wraps whole-scalar {{ .VAR }} tokens in double
// quotes so YAML parsers see a string instead of a flow mapping. This is the
// inverse of UnquoteTokens for read paths that need to inspect raw specs
// without resolving variables.
func QuoteTokensForYAMLParse(data []byte) []byte {
	matches := varRegex.FindAllIndex(data, -1)
	if len(matches) == 0 {
		return data
	}

	for i := len(matches) - 1; i >= 0; i-- {
		matchStart, matchEnd := matches[i][0], matches[i][1]
		if isInComment(data, matchStart) || isInsideQuote(data, matchStart) || !isWholeScalarToken(data, matchStart, matchEnd) {
			continue
		}

		replacement := []byte(`"` + string(data[matchStart:matchEnd]) + `"`)
		data = replaceRange(data, matchStart, matchEnd, replacement)
	}

	return data
}

// UnquoteTokens replaces every double-quoted "{{ .VAR }}" token in data with
// its unquoted form, so generated specs read as template references rather
// than string literals. YAML encoders cannot emit a scalar starting with '{'
// unquoted (it reads as a flow mapping), so generators that emit references
// post-process their output with this. Read paths that parse raw specs without
// substitution should call QuoteTokensForYAMLParse first. Tokens embedded in
// longer strings keep their quotes.
func UnquoteTokens(data []byte) []byte {
	return quotedVarRegex.ReplaceAll(data, []byte("$1"))
}

// ExtractVariableNames returns the names of all well-formed {{ .VAR }}
// references in data, in order of appearance. Malformed tokens are skipped:
// extraction reports what the substitutor would resolve, not what it would
// reject. Import scaffolding uses this to discover which variables the
// generated specs reference so it can emit a placeholder for each.
func ExtractVariableNames(data []byte) []string {
	matches := varRegex.FindAllSubmatch(data, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		name, err := parseVarName(string(m[1]))
		if err != nil {
			continue
		}
		names = append(names, name)
	}
	return names
}

func isInsideQuote(data []byte, matchStart int) bool {
	lineStart := matchStart
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}

	var (
		inSingleQuote bool
		inDoubleQuote bool
	)

	for i := lineStart; i < matchStart; i++ {
		switch data[i] {
		case '\\':
			if inDoubleQuote && i+1 < matchStart {
				i++
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
	}

	return inSingleQuote || inDoubleQuote
}

func isWholeScalarToken(data []byte, matchStart, matchEnd int) bool {
	lineStart := matchStart
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}

	lineEnd := matchEnd
	for lineEnd < len(data) && data[lineEnd] != '\n' {
		lineEnd++
	}

	if hasTrailingScalarContent(data[matchEnd:lineEnd]) {
		return false
	}

	prefix := trimRightSpace(data[lineStart:matchStart])
	if len(prefix) == 0 {
		return true
	}

	last := prefix[len(prefix)-1]
	if last == ':' {
		return true
	}

	if last == '-' && (len(prefix) == 1 || prefix[len(prefix)-2] == ' ' || prefix[len(prefix)-2] == '\t') {
		return true
	}

	return false
}

func hasTrailingScalarContent(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t':
			continue
		case '#':
			return false
		default:
			return true
		}
	}
	return false
}

func trimRightSpace(data []byte) []byte {
	end := len(data)
	for end > 0 && (data[end-1] == ' ' || data[end-1] == '\t') {
		end--
	}
	return data[:end]
}
