package varsubst

import (
	"bytes"
	"crypto/rand"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

type Resolver interface {
	Resolve(name string) (value string, found bool)
}

// Matches a {{ .VAR }} token. Group 1 captures the dot-prefixed token; the dot
// is required so substitution claims only its own syntax and leaves other
// `{{ … }}` dialects — notably the RudderStack UI's `{{ path || fallback }}`
// destination config templates — untouched. Group 2 (optional) captures the
// default value after pipe — a single `}` is allowed (so defaults can contain
// regex like `[a-z]{3}`), but `}}` always terminates the token. Surrounding
// whitespace is stripped by the enclosing \s* groups. The variable name pattern
// is still validated in code after matching, so a malformed dotted token like
// `{{ .9BAD }}` is reported rather than silently passed through.
var varRegex = regexp.MustCompile(`\{\{\s*(\.[^}\s|]+)(?:\s*\|\s*((?:[^}]|}[^}])*?))?\s*\}\}`)

var validVarName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// IsValidVariableName reports whether name satisfies the variable name grammar
// a {{ .name }} token requires. Code that generates tokens (e.g. import
// scaffolding) uses it to reject a bad name at the moment the token is
// created, rather than when substitution later fails on it.
func IsValidVariableName(name string) bool {
	return validVarName.MatchString(name)
}

// Substitutor performs `{{ .VAR }}` substitution against a chain of resolvers.
//
// Known limitations:
//
//   - Substitution always runs inside YAML quoted strings. There is currently
//     no escape mechanism to write a literal `{{ }}` token (e.g. in
//     description text), so any `{{ .X }}`-shaped substring in a quoted value
//     is treated as a variable reference.
//   - Resolved values are injected verbatim, except that a multi-line value
//     filling a quoted scalar keeps its line breaks and backslashes (see
//     restoreMultilineValues). Values containing YAML special characters, or
//     content that parses as a different YAML type (`true`, `123`, `null`,
//     sequences) can change the document's semantics. Callers that need to
//     force a string should quote at the call site: `flag: "{{ .FLAG }}"`.
//     Double quotes break on a one-line value containing `"` or `\`; single
//     quotes (`'{{ .FLAG }}'`, which import generates) break on a value
//     containing `'`, which must be doubled in the variable.
type Substitutor interface {
	SubstituteBytes(data []byte) ([]byte, []SubstitutionError)
}

type substitutor struct {
	resolvers []Resolver
}

func NewSubstitutor(resolvers ...Resolver) Substitutor {
	return &substitutor{resolvers: resolvers}
}

// rawError stores the name of the variable, the byte offset, and the error kind.
// This is used to compute the line/column positions of the error.
type rawError struct {
	name   string
	offset int
	err    error
}

// SubstituteBytes finds all {{ .VAR }} tokens and resolves them through the
// resolver chain. Matches are processed last-to-first so that replacing a token
// with a different-length value does not invalidate earlier byte offsets.
// A copy of the original bytes is kept so that error positions can be computed
// against unmodified content after all replacements are done.
func (s *substitutor) SubstituteBytes(data []byte) ([]byte, []SubstitutionError) {
	matches := varRegex.FindAllSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return data, nil
	}

	original := make([]byte, len(data))
	copy(original, data)

	var (
		rawErrors []rawError
		// A multi-line value goes in as a one-line placeholder until the spec
		// can be parsed. The random nonce keeps a placeholder from matching
		// text already in the spec or in another value.
		multiline = map[string]string{}
		nonce     = rand.Text()
	)

	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		matchStart, matchEnd := match[0], match[1]

		if isInComment(data, matchStart) {
			continue
		}

		token := string(data[match[2]:match[3]])

		varName, err := parseVarName(token)
		if err != nil {
			rawErrors = append(rawErrors, rawError{name: varName, offset: matchStart, err: err})
			continue
		}

		var (
			defaultVal string
			hasDefault = match[4] != -1
		)
		if hasDefault {
			// Trim trailing whitespace: when the default contains a single `}`
			// (e.g. `[a-z]{3}`), regex backtracking can absorb the trailing
			// space before `}}` into the capture group.
			defaultVal = strings.TrimRight(string(data[match[4]:match[5]]), " \t")
		}

		var (
			resolved string
			found    bool
		)
		for _, r := range s.resolvers {
			resolved, found = r.Resolve(varName)
			if found {
				break
			}
		}

		if !found {
			if hasDefault {
				resolved = defaultVal
			} else {
				rawErrors = append(rawErrors, rawError{name: varName, offset: matchStart, err: ErrUndefinedVariable})
				continue
			}
		}

		if hasLoneSingleQuote(resolved) && inSingleQuotedScalar(data, matchStart) {
			rawErrors = append(rawErrors, rawError{name: varName, offset: matchStart, err: ErrUnescapedSingleQuote})
			continue
		}

		if resolved == "" && !isAdjacentToQuote(data, matchStart, matchEnd) {
			resolved = `""`
		}

		// strconv.Quote can't escape a value that isn't valid UTF-8 without
		// changing it, so such a value keeps the verbatim behavior.
		if isMultiline(resolved) && utf8.ValidString(resolved) {
			// The trailing - keeps placeholder 1 from matching inside 10.
			placeholder := "varsubst-" + nonce + "-" + strconv.Itoa(len(multiline)) + "-"
			multiline[placeholder] = resolved
			resolved = placeholder
		}

		data = replaceRange(data, matchStart, matchEnd, []byte(resolved))
	}

	errs := computePositions(original, rawErrors)
	return restoreMultilineValues(data, multiline), errs
}

func parseVarName(token string) (string, error) {
	if token[0] != '.' {
		return token, ErrInvalidVarSyntax
	}

	name := token[1:]
	if !validVarName.MatchString(name) {
		return name, ErrInvalidVarSyntax
	}

	return name, nil
}

func replaceRange(data []byte, start, end int, replacement []byte) []byte {
	result := make([]byte, 0, start+len(replacement)+len(data)-end)
	result = append(result, data[:start]...)
	result = append(result, replacement...)
	result = append(result, data[end:]...)
	return result
}

// isMultiline reports whether value holds a line break, which YAML folds into
// a space inside a quoted scalar.
func isMultiline(value string) bool {
	return strings.ContainsAny(value, "\r\n\u0085\u2028\u2029")
}

// restoreMultilineValues puts each multi-line value back in place of its
// placeholder. Where the placeholder is the whole value of a quoted scalar,
// the scalar becomes a double-quoted one with the line breaks escaped, so
// parsing keeps them: every escape strconv.Quote writes means the same in a
// YAML double-quoted scalar. Anywhere else the value goes in verbatim, as it
// always has.
//
// A value put back verbatim can change the YAML around the placeholders left,
// or be what the spec needs to parse at all, so values go back one at a time
// and the spec is parsed again after each, until every placeholder left is the
// whole of a quoted scalar.
func restoreMultilineValues(data []byte, multiline map[string]string) []byte {
	for len(multiline) > 0 {
		quotes, parsed := placeholderQuotes(data, multiline)
		if parsed && len(quotes) == len(multiline) {
			return quotePlaceholders(data, multiline, quotes)
		}

		// A placeholder written outside quotes goes back first: its value is
		// the kind that can change the YAML around the rest. Sorting keeps the
		// order, and so the result, deterministic.
		var verbatim string
		for _, placeholder := range slices.Sorted(maps.Keys(multiline)) {
			if _, whole := quotes[placeholder]; whole {
				continue
			}
			if quoteAround(data, placeholder) == 0 {
				verbatim = placeholder
				break
			}
			if verbatim == "" {
				verbatim = placeholder
			}
		}
		data = bytes.Replace(data, []byte(verbatim), []byte(multiline[verbatim]), 1)
		delete(multiline, verbatim)
	}
	return data
}

// quotePlaceholders swaps each placeholder, with the quotes around it, for its
// value written as a double-quoted scalar.
func quotePlaceholders(data []byte, multiline map[string]string, quotes map[string]byte) []byte {
	for placeholder, value := range multiline {
		quote := quotes[placeholder]
		if quote == '\'' {
			// A single-quoted scalar reads '' as one quote, as it does for a
			// one-line value.
			value = strings.ReplaceAll(value, "''", "'")
		}
		// A double-quoted slot gets the value as written: unlike a one-line
		// value, its backslashes are not read as YAML escapes.
		data = bytes.Replace(data, []byte(string(quote)+placeholder+string(quote)), []byte(strconv.Quote(value)), 1)
	}
	return data
}

// placeholderQuotes parses data and returns the quote character of each quoted
// scalar whose whole value is a placeholder, and whether data parsed at all.
func placeholderQuotes(data []byte, multiline map[string]string) (map[string]byte, bool) {
	var document yaml.Node
	if yaml.Unmarshal(data, &document) != nil {
		return nil, false
	}

	quotes := map[string]byte{}
	collectPlaceholderQuotes(data, &document, multiline, quotes)
	return quotes, true
}

func collectPlaceholderQuotes(data []byte, node *yaml.Node, multiline map[string]string, quotes map[string]byte) {
	// The scalar is swapped through its text, so it must be written as the
	// placeholder between its quotes; a double-quoted scalar continued across
	// lines isn't.
	if _, ok := multiline[node.Value]; ok && node.Kind == yaml.ScalarNode {
		quote := quoteAround(data, node.Value)
		if quote == '\'' && node.Style&yaml.SingleQuotedStyle != 0 || quote == '"' && node.Style&yaml.DoubleQuotedStyle != 0 {
			quotes[node.Value] = quote
		}
	}
	for _, child := range node.Content {
		collectPlaceholderQuotes(data, child, multiline, quotes)
	}
}

// quoteAround returns the quote character written around placeholder in data,
// or 0 when the placeholder isn't written between quotes.
func quoteAround(data []byte, placeholder string) byte {
	for _, quote := range []byte{'\'', '"'} {
		if bytes.Contains(data, []byte(string(quote)+placeholder+string(quote))) {
			return quote
		}
	}
	return 0
}

// isInComment finds the start of the line containing matchStart, then scans
// forward tracking single/double quote state. Returns true if an unquoted #
// appears before matchStart, indicating the token is inside a YAML comment.
// Inside a double-quoted string, a backslash escapes the next character so
// that `\"` is treated as part of the string rather than closing it.
func isInComment(data []byte, matchStart int) bool {
	lineStart := matchStart
	// go backward to find the start of the line
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
			// In YAML double-quoted strings, `\X` is an escape sequence; skip
			// the next character so an escaped quote is not treated as a close.
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
		case '#':
			if !inSingleQuote && !inDoubleQuote {
				return true
			}
		}
	}

	return false
}

// inSingleQuotedScalar reports whether matchStart sits inside a single-quoted
// scalar that opens earlier on the same line. A scalar that spans lines is not
// detected, which leaves the existing behavior for it.
func inSingleQuotedScalar(data []byte, matchStart int) bool {
	lineStart := matchStart
	for lineStart > 0 && data[lineStart-1] != '\n' {
		lineStart--
	}

	var inSingle, inDouble bool
	for i := lineStart; i < matchStart; i++ {
		switch data[i] {
		case '\\':
			if inDouble && i+1 < matchStart {
				i++
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return false
			}
		}
	}

	return inSingle
}

// hasLoneSingleQuote reports whether value holds a run of single quotes of odd
// length. Inside a single-quoted scalar an even run is a valid escape (`''`
// reads as one quote), and an odd run leaves one quote that ends the scalar.
func hasLoneSingleQuote(value string) bool {
	run := 0
	for i := 0; i <= len(value); i++ {
		if i < len(value) && value[i] == '\'' {
			run++
			continue
		}
		if run%2 == 1 {
			return true
		}
		run = 0
	}
	return false
}

// isAdjacentToQuote reports whether the byte immediately before matchStart or
// immediately after matchEnd is a YAML quote character. Used to suppress the
// auto-quoting of empty resolved values when the token is already wrapped in
// quotes (e.g. `"{{ .VAR }}"`), which would otherwise produce invalid YAML.
func isAdjacentToQuote(data []byte, matchStart, matchEnd int) bool {
	if matchStart > 0 {
		if c := data[matchStart-1]; c == '"' || c == '\'' {
			return true
		}
	}
	if matchEnd < len(data) {
		if c := data[matchEnd]; c == '"' || c == '\'' {
			return true
		}
	}
	return false
}

// computePositions converts raw byte offsets into line/col positions in a single
// scan of the original (pre-substitution) bytes. Errors are reversed into ascending
// offset order because they were collected during the right-to-left substitution
// pass. Then positions are emitted as the scan reaches each error's offset.
func computePositions(original []byte, rawErrors []rawError) []SubstitutionError {
	if len(rawErrors) == 0 {
		return nil
	}

	// reverse into ascending offset order so we can scan the bytes left-to-right
	slices.Reverse(rawErrors)

	var (
		result    = make([]SubstitutionError, 0, len(rawErrors))
		line      = 1
		lineStart int
		errorIdx  int
	)

	// walk through original bytes once, tracking line/col as we go.
	// when the current byte offset matches the next error's offset,
	// scan forward to find the end of the line and emit the error.
	// Lines break where the YAML parser breaks them (breakLen), CR and CRLF
	// included, so the line matches the one validate places other
	// diagnostics on.
	for i := 0; i < len(original); {
		if i == rawErrors[errorIdx].offset {
			lineEnd := i
			for lineEnd < len(original) && breakLen(original[lineEnd:]) == 0 {
				lineEnd++
			}

			// Line and column numbers are 1-indexed (matching how editors and error messages display positions), but byte offsets are 0-indexed. 
			// The +1 converts from zero-based offset to one-based column.
			result = append(result, SubstitutionError{
				Name:     rawErrors[errorIdx].name,
				Line:     line,				
				Column:   i - lineStart + 1,
				LineText: string(original[lineStart:lineEnd]),
				Err:      rawErrors[errorIdx].err,
			})

			errorIdx++
			if errorIdx >= len(rawErrors) {
				break
			}
		}

		if n := breakLen(original[i:]); n > 0 {
			i += n
			line++
			lineStart = i
			continue
		}
		i++
	}

	return result
}

