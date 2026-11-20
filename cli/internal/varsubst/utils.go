package varsubst

import "regexp"

// quotedVarRegex matches a {{ .VAR }} token directly enclosed in double
// quotes, capturing the token. Built from varRegex so the token grammar lives
// in one place.
var quotedVarRegex = regexp.MustCompile(`"(` + varRegex.String() + `)"`)

// SingleQuoteTokens rewrites every double-quoted "{{ .VAR }}" token in data to
// its single-quoted form. Generated specs emit secrets as variable references,
// and the encoder double-quotes them like any string. A bare slot is not safe:
// substitution rewrites raw bytes before YAML parsing, so a JSON-blob secret
// (a GCP service account key) substituted into an unquoted slot parses as a
// flow mapping and the field silently becomes a map. A double-quoted slot also
// breaks, because the JSON's own quotes end the scalar. Single quotes keep the
// substituted JSON a string because they do not interpret backslashes or
// double quotes.
//
// Only a whole scalar is rewritten. Tokens embedded in longer strings keep
// their double quotes, including a token that ends a string after an escaped
// quote (`"x\"{{ .X }}"`): the quote before it is part of the string, not the
// start of a scalar, and rewriting it would produce invalid YAML.
func SingleQuoteTokens(data []byte) []byte {
	var (
		out  = make([]byte, 0, len(data))
		last int
	)
	for _, m := range quotedVarRegex.FindAllSubmatchIndex(data, -1) {
		start, end := m[0], m[1]
		if escapedQuote(data, start) {
			continue
		}
		out = append(out, data[last:start]...)
		out = append(out, '\'')
		out = append(out, data[m[2]:m[3]]...)
		out = append(out, '\'')
		last = end
	}
	return append(out, data[last:]...)
}

// escapedQuote reports whether the byte at index is preceded by an odd number
// of backslashes, which makes it an escape sequence inside a double-quoted
// string and not the start of one.
func escapedQuote(data []byte, index int) bool {
	slashes := 0
	for i := index - 1; i >= 0 && data[i] == '\\'; i-- {
		slashes++
	}
	return slashes%2 == 1
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
