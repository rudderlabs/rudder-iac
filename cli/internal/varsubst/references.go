package varsubst

import (
	"bytes"
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
	// Text is the token as written, such as "{{ .NAME | fallback }}".
	Text string
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
	// byPlaceholder maps each placeholder in Data to its reference.
	byPlaceholder map[string]Reference
}

// A placeholder is placeholderPrefix, the reference's index and
// placeholderSuffix. The form is a plain YAML string that a spec is unlikely
// to hold, and ReplaceReferences lengthens the prefix when the spec holds it
// anyway.
const (
	placeholderPrefix = "__rudder_variable_reference_"
	placeholderSuffix = "__"
)

// ReplaceReferences returns a copy of data with every {{ .NAME }} token that
// substitution would resolve swapped for a numbered placeholder. Tokens in YAML
// comments and tokens with a malformed name are left as they are, as
// substitution leaves them.
//
// It lets a caller read a spec as written: an unquoted token parses as a YAML
// flow mapping, so a spec holding one may not parse until substitution runs,
// while a placeholder parses as a plain string wherever the token stood.
func ReplaceReferences(data []byte) Replaced {
	replaced := Replaced{prefix: placeholderPrefix, byPlaceholder: make(map[string]Reference)}
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

		ref := Reference{Name: name, HasDefault: match[4] != -1, Text: string(data[start:end])}
		placeholder := replaced.prefix + strconv.Itoa(len(replaced.References)) + placeholderSuffix
		replaced.byPlaceholder[placeholder] = ref
		replaced.References = append(replaced.References, ref)

		out = append(out, data[last:start]...)
		out = append(out, placeholder...)
		last = end
	}
	replaced.Data = append(out, data[last:]...)
	return replaced
}

// Reference returns the reference that value stands for when value is exactly
// one placeholder, with nothing around it. value comes from parsing Data.
func (r Replaced) Reference(value string) (Reference, bool) {
	ref, ok := r.byPlaceholder[value]
	return ref, ok
}

// HoldsReference reports whether value holds a placeholder, alone or with
// text around it. value comes from parsing Data.
func (r Replaced) HoldsReference(value string) bool {
	return len(r.References) > 0 && strings.Contains(value, r.prefix)
}

// Restore swaps each placeholder in text for its reference as written. text
// comes from Data, such as a value parsed from it or a line of it, and a
// caller restores it before showing it, since a placeholder means nothing to
// the person who wrote the spec.
func (r Replaced) Restore(text string) string {
	if len(r.byPlaceholder) == 0 || !strings.Contains(text, r.prefix) {
		return text
	}
	for placeholder, ref := range r.byPlaceholder {
		text = strings.ReplaceAll(text, placeholder, ref.Text)
	}
	return text
}
