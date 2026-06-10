package varsubst

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
