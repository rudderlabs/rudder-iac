package varsubst

import (
	"fmt"
	"regexp"
	"strconv"
)

// Reference is a {{ .NAME }} token as written in spec bytes.
type Reference struct {
	// Name is the variable name, without the leading dot.
	Name string
	// HasDefault reports a `| default` part, which substitution uses when the
	// variable is undefined.
	HasDefault bool
}

// placeholderRegex matches a whole placeholder that ReplaceReferences writes.
// The form is a plain YAML string that a spec is unlikely to hold by chance.
var placeholderRegex = regexp.MustCompile(`^__rudder_variable_reference_(\d+)__$`)

func placeholder(index int) string {
	return fmt.Sprintf("__rudder_variable_reference_%d__", index)
}

// ReplaceReferences returns a copy of data with every {{ .NAME }} token that
// substitution would resolve swapped for a numbered placeholder, and the
// references in placeholder order. Tokens in YAML comments and tokens with a
// malformed name are left as they are, as substitution leaves them.
//
// It lets a caller read a spec as written: an unquoted token parses as a YAML
// flow mapping, so a spec holding one may not parse until substitution runs,
// while a placeholder parses as a plain string wherever the token stood.
func ReplaceReferences(data []byte) ([]byte, []Reference) {
	var (
		out  = make([]byte, 0, len(data))
		refs []Reference
		last int
	)
	for _, match := range varRegex.FindAllSubmatchIndex(data, -1) {
		start, end := match[0], match[1]
		if isInComment(data, start) {
			continue
		}
		name, err := parseVarName(string(data[match[2]:match[3]]))
		if err != nil {
			continue
		}

		out = append(out, data[last:start]...)
		out = append(out, placeholder(len(refs))...)
		refs = append(refs, Reference{Name: name, HasDefault: match[4] != -1})
		last = end
	}
	return append(out, data[last:]...), refs
}

// ReplacedReference returns the reference that value stands for when value is
// exactly one placeholder written by ReplaceReferences, with nothing around it.
// refs is the slice ReplaceReferences returned for the bytes value came from.
func ReplacedReference(value string, refs []Reference) (Reference, bool) {
	match := placeholderRegex.FindStringSubmatch(value)
	if match == nil {
		return Reference{}, false
	}
	index, err := strconv.Atoi(match[1])
	if err != nil || index >= len(refs) {
		return Reference{}, false
	}
	return refs[index], true
}
