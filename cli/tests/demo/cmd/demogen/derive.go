package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// readOnlyVerbs are CLI commands that only read. A step ending in one shows
// its own proof on screen; a step that does not is asserted invisibly in Go,
// which is what a demo has to say out loud.
//
// ponytail: a flat allowlist, not command-tree introspection. It is advisory —
// a wrong answer mislabels a manifest field, it does not break a demo. Replace
// it with a real read-only annotation on the cobra commands if that ever
// matters. --json is deliberately not here: it is an output format, not a
// read/write property, and letting it stand in for "read-only" would make a
// write like `apply --json --confirm=false` look like it proved itself on
// screen.
var readOnlyVerbs = map[string]bool{
	"list":     true,
	"validate": true,
	"preview":  true,
	"get":      true,
	"view":     true,
	"info":     true,
	"diff":     true,
}

var readOnlyFlags = map[string]bool{
	"--dry-run": true,
}

// DeriveProse turns a subtest name into a sentence.
//
// go test reports names with spaces replaced by underscores, so the raw form is
// "should_create_entities_in_catalog_from_project". The prose it yields carries
// the step but never the reason — that is what demo.Say is for.
func DeriveProse(testName string) string {
	name := testName
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	name = strings.TrimPrefix(name, "Test")
	words := strings.Split(strings.ReplaceAll(name, "_", " "), " ")

	// "should" adds nothing to a narrated step and reads oddly in the middle of
	// one ("migrated create specs should produce…" -> "…specs produce…").
	kept := make([]string, 0, len(words))
	for _, w := range words {
		if strings.EqualFold(w, "should") || w == "" {
			continue
		}
		kept = append(kept, w)
	}

	if len(kept) == 0 {
		return ""
	}

	// A bare test name is CamelCase; a subtest name is already spaced.
	if len(kept) == 1 && hasInnerUpper(kept[0]) {
		kept = splitCamel(kept[0])
	}

	joined := strings.Join(kept, " ")

	// joined[:1] would slice by byte, truncating a multi-byte leading rune
	// mid-codepoint — decode the first rune explicitly instead.
	r, size := utf8.DecodeRuneInString(joined)

	return strings.ToUpper(string(r)) + joined[size:]
}

func hasInnerUpper(s string) bool {
	for _, r := range s[1:] {
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

func splitCamel(s string) []string {
	var (
		out  []string
		word strings.Builder
	)

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			out = append(out, strings.ToLower(word.String()))
			word.Reset()
		}
		word.WriteRune(r)
	}
	out = append(out, strings.ToLower(word.String()))

	return out
}

// IsThin reports whether derived prose is too short to tell a viewer anything.
// "success" and "failure" are real subtest names in this suite and describe
// nothing; a step like that is exactly where demo.Say earns its keep.
func IsThin(testName string) bool {
	return len(strings.Fields(DeriveProse(testName))) < 2
}

// Annotated reports whether a step carries hand-written narration.
func Annotated(s Step) bool {
	for _, r := range s.Records {
		if r.Kind == demo.KindSay {
			return true
		}
	}

	return false
}

// Verification reports whether the step proves its result on screen.
func Verification(s Step) string {
	for i := len(s.Records) - 1; i >= 0; i-- {
		r := s.Records[i]
		if r.Kind != demo.KindExec {
			continue
		}

		if isReadOnly(r.Argv) {
			return "visible"
		}

		return "none"
	}

	return "none"
}

func isReadOnly(argv []string) bool {
	for _, a := range argv {
		if readOnlyFlags[a] || readOnlyVerbs[a] {
			return true
		}
	}

	return false
}
