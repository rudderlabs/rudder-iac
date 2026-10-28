package varsubst

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
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

	// written is the spec as written. spans holds where each placeholder sits
	// in Data and where its reference sits in written, in order, and
	// dataLines and writtenLines hold where each line of Data and of written
	// starts. SourcePosition reads them, so it costs no scan of the file.
	// They are set only when the spec holds a reference.
	written                 []byte
	spans                   []span
	dataLines, writtenLines []int
}

// span is one placeholder's bytes in Data and its reference's bytes in the
// spec as written.
type span struct {
	dataStart, dataEnd, writtenStart, writtenEnd int
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
		replaced.spans = append(replaced.spans, span{
			dataStart: len(out), dataEnd: len(out) + len(placeholder), writtenStart: start, writtenEnd: end,
		})
		out = append(out, placeholder...)
		last = end
	}
	replaced.Data = append(out, data[last:]...)
	if len(replaced.spans) > 0 {
		replaced.written = data
		replaced.dataLines, replaced.writtenLines = lineStarts(replaced.Data), lineStarts(data)
	}
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

// SourcePosition returns where a position in Data sits in the spec as
// written. line and column are 1-indexed, with columns counted in characters,
// as the YAML parser reports them. A placeholder and its reference differ in
// length, and a reference may span lines, so a key that follows a reference
// moves when the reference goes back.
//
// It reads the offsets ReplaceReferences recorded, so a call costs a binary
// search and a walk along one line, however many references the file holds.
func (r Replaced) SourcePosition(line, column int) (int, int) {
	if len(r.spans) == 0 {
		return line, column
	}
	offset := r.dataOffset(line, column)

	// The last placeholder that starts before the position. With none, the
	// text before it is as written.
	i := sort.Search(len(r.spans), func(i int) bool { return r.spans[i].dataStart >= offset }) - 1
	if i < 0 {
		return line, column
	}
	s := r.spans[i]
	at := s.writtenStart
	if offset >= s.dataEnd {
		at = s.writtenEnd + offset - s.dataEnd
	}

	l := sort.Search(len(r.writtenLines), func(j int) bool { return r.writtenLines[j] > at }) - 1
	return l + 1, utf8.RuneCount(r.written[r.writtenLines[l]:at]) + 1
}

// dataOffset returns the byte offset in Data of a 1-indexed line and column,
// with columns counted in characters.
func (r Replaced) dataOffset(line, column int) int {
	if line > len(r.dataLines) {
		return len(r.Data)
	}
	offset := r.dataLines[max(line, 1)-1]
	for ; column > 1 && offset < len(r.Data) && breakLen(r.Data[offset:]) == 0; column-- {
		_, size := utf8.DecodeRune(r.Data[offset:])
		offset += size
	}
	return offset
}

// lineStarts returns the offset where each line of data starts, breaking
// lines where the YAML parser does.
func lineStarts(data []byte) []int {
	starts := []int{0}
	for i := 0; i < len(data); {
		if n := breakLen(data[i:]); n > 0 {
			i += n
			starts = append(starts, i)
			continue
		}
		_, size := utf8.DecodeRune(data[i:])
		i += size
	}
	return starts
}

// Line breaks other than LF and CR that the YAML parser counts as one.
var (
	nextLine           = []byte("\u0085")
	lineSeparator      = []byte("\u2028")
	paragraphSeparator = []byte("\u2029")
)

// breakLen returns the length in bytes of the line break data starts with, or
// 0. It counts what the YAML parser counts as one line break: LF, CR, CRLF,
// NEL, LS and PS. A file with CR line endings would otherwise read as one
// line.
func breakLen(data []byte) int {
	switch {
	case len(data) == 0:
		return 0
	case data[0] == '\n':
		return 1
	case data[0] == '\r':
		if len(data) > 1 && data[1] == '\n' {
			return 2
		}
		return 1
	case bytes.HasPrefix(data, nextLine):
		return len(nextLine)
	case bytes.HasPrefix(data, lineSeparator), bytes.HasPrefix(data, paragraphSeparator):
		return len(lineSeparator)
	}
	return 0
}
