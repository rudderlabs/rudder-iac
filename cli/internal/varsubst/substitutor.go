package varsubst

import (
	"regexp"
	"slices"
	"strings"
)

type Resolver interface {
	Resolve(name string) (value string, found bool)
}

// Matches any {{ content }} token. Group 1 captures the token (anything that
// isn't whitespace, pipe, or closing brace). Group 2 (optional) captures the
// default value after pipe — a single `}` is allowed (so defaults can contain
// regex like `[a-z]{3}`), but `}}` always terminates the token. Surrounding
// whitespace is stripped by the enclosing \s* groups. Validation of the token
// (dot prefix, variable name pattern) happens in code after matching.
var varRegex = regexp.MustCompile(`\{\{\s*([^}\s|]+)(?:\s*\|\s*((?:[^}]|}[^}])*?))?\s*\}\}`)

var validVarName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Substitutor performs `{{ .VAR }}` substitution against a chain of resolvers.
//
// Known limitations:
//
//   - Substitution always runs inside YAML quoted strings. There is currently
//     no escape mechanism to write a literal `{{ }}` token (e.g. in
//     description text), so any `{{ .X }}`-shaped substring in a quoted value
//     is treated as a variable reference.
//   - Resolved values are injected verbatim. Values containing newlines, YAML
//     special characters, or content that parses as a different YAML type
//     (`true`, `123`, `null`, sequences) can change the document's semantics.
//     Callers that need to force a string should quote at the call site:
//     `flag: "{{ .FLAG }}"`.
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

	var rawErrors []rawError

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

		if resolved == "" && !isAdjacentToQuote(data, matchStart, matchEnd) {
			resolved = `""`
		}

		data = replaceRange(data, matchStart, matchEnd, []byte(resolved))
	}

	errs := computePositions(original, rawErrors)
	return data, errs
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
	for i := range original {
		if i == rawErrors[errorIdx].offset {
			lineEnd := i
			for lineEnd < len(original) && original[lineEnd] != '\n' {
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

		if original[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}

	return result
}

