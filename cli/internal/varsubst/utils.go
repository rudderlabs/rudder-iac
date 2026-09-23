package varsubst

import "regexp"

// quotedVarRegex matches a {{ .VAR }} token directly enclosed in double
// quotes, capturing the token. Built from varRegex so the token grammar lives
// in one place.
var quotedVarRegex = regexp.MustCompile(`"(` + varRegex.String() + `)"`)

// UnquoteTokens replaces every double-quoted "{{ .VAR }}" token in data with
// its unquoted form, so generated specs read as template references rather
// than string literals. YAML encoders cannot emit a scalar starting with '{'
// unquoted (it reads as a flow mapping), so generators that emit references
// post-process their output with this. Substitution rewrites raw bytes before
// YAML parsing, so the unquoted form never reaches a parser. Tokens embedded
// in longer strings keep their quotes.
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
