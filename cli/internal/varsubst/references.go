package varsubst

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Reference is a {{ .NAME }} token as written in spec bytes.
type Reference struct {
	// Name is the variable name, without the leading dot.
	Name string
	// HasDefault reports a `| default` part, which substitution uses when the
	// variable is undefined.
	HasDefault bool
}

// Replaced is spec bytes with every {{ .NAME }} reference swapped for a
// numbered placeholder, as ReplaceReferences returns them.
type Replaced struct {
	// Data is the spec with each reference replaced by its placeholder.
	Data []byte
	// References lists the references in placeholder order.
	References []Reference
	// prefix starts every placeholder. It does not occur in the spec as
	// written, so a literal is never read as a placeholder.
	prefix string
}

// placeholderPrefix starts a placeholder. The form is a plain YAML string that
// a spec is unlikely to hold, and ReplaceReferences lengthens it when the spec
// holds it anyway.
const placeholderPrefix = "__rudder_variable_reference_"

func (r Replaced) placeholder(index int) string {
	return fmt.Sprintf("%s%d__", r.prefix, index)
}

// ReplaceReferences returns a copy of data with every {{ .NAME }} token that
// substitution would resolve swapped for a numbered placeholder. Tokens in YAML
// comments and tokens with a malformed name are left as they are, as
// substitution leaves them.
//
// It lets a caller read a spec as written: an unquoted token parses as a YAML
// flow mapping, so a spec holding one may not parse until substitution runs,
// while a placeholder parses as a plain string wherever the token stood.
func ReplaceReferences(data []byte) Replaced {
	replaced := Replaced{prefix: placeholderPrefix}
	for bytes.Contains(data, []byte(replaced.prefix)) {
		replaced.prefix = "_" + replaced.prefix
	}

	var (
		out  = make([]byte, 0, len(data))
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
		out = append(out, replaced.placeholder(len(replaced.References))...)
		replaced.References = append(replaced.References, Reference{Name: name, HasDefault: match[4] != -1})
		last = end
	}
	replaced.Data = append(out, data[last:]...)
	return replaced
}

// Reference returns the reference that value stands for when value is exactly
// one placeholder, with nothing around it. value comes from parsing Data.
func (r Replaced) Reference(value string) (Reference, bool) {
	digits, ok := strings.CutPrefix(value, r.prefix)
	if !ok {
		return Reference{}, false
	}
	digits, ok = strings.CutSuffix(digits, "__")
	if !ok {
		return Reference{}, false
	}
	index, err := strconv.Atoi(digits)
	if err != nil || index < 0 || index >= len(r.References) || r.placeholder(index) != value {
		return Reference{}, false
	}
	return r.References[index], true
}

// HoldsReference reports whether value holds a placeholder, alone or with
// text around it. value comes from parsing Data.
func (r Replaced) HoldsReference(value string) bool {
	return len(r.References) > 0 && strings.Contains(value, r.prefix)
}
