package varsubst

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

// quotedVarRegex matches a {{ .VAR }} token directly enclosed in double
// quotes, capturing the token. Built from varRegex so the token grammar lives
// in one place.
var quotedVarRegex = regexp.MustCompile(`"(` + varRegex.String() + `)"`)

// TokenMask stores the generated sentinels used to parse raw YAML containing
// unresolved {{ .VAR }} tokens without resolving those variables.
type TokenMask struct {
	prefix string
	tokens map[string]string
}

// MaskTokensForYAMLParse replaces active {{ .VAR }} tokens with generated
// bare sentinels so YAML parsers can read raw specs without treating unquoted
// tokens as flow mappings. Callers should restore parsed string values with
// RestoreString before formatting or writing.
func MaskTokensForYAMLParse(data []byte) ([]byte, TokenMask) {
	matches := varRegex.FindAllIndex(data, -1)
	if len(matches) == 0 {
		return data, TokenMask{}
	}

	mask := TokenMask{
		prefix: sentinelPrefix(data),
		tokens: make(map[string]string, len(matches)),
	}

	replacements := make([]tokenReplacement, 0, len(matches))
	for _, match := range matches {
		matchStart, matchEnd := match[0], match[1]
		if isInComment(data, matchStart) {
			continue
		}

		sentinel := mask.prefix + strconv.Itoa(len(replacements)) + "__"
		mask.tokens[sentinel] = string(data[matchStart:matchEnd])
		replacements = append(replacements, tokenReplacement{
			start:       matchStart,
			end:         matchEnd,
			replacement: sentinel,
		})
	}

	for i := len(replacements) - 1; i >= 0; i-- {
		replacement := replacements[i]
		data = replaceRange(data, replacement.start, replacement.end, []byte(replacement.replacement))
	}

	return data, mask
}

// UnquoteTokens replaces every double-quoted "{{ .VAR }}" token in data with
// its unquoted form, so generated specs read as template references rather
// than string literals. YAML encoders cannot emit a scalar starting with '{'
// unquoted (it reads as a flow mapping), so generators that emit references
// post-process their output with this. Read paths that parse raw specs without
// substitution should call MaskTokensForYAMLParse first. Tokens embedded in
// longer strings keep their quotes.
func UnquoteTokens(data []byte) []byte {
	return quotedVarRegex.ReplaceAll(data, []byte("$1"))
}

// RestoreString replaces generated sentinels in value with their original
// {{ .VAR }} token text.
func (m TokenMask) RestoreString(value string) string {
	if len(m.tokens) == 0 {
		return value
	}

	pairs := make([]string, 0, len(m.tokens)*2)
	for sentinel, token := range m.tokens {
		pairs = append(pairs, sentinel, token)
	}

	return strings.NewReplacer(pairs...).Replace(value)
}

// ContainsSentinel reports whether value still contains this mask's sentinel
// prefix after restoration.
func (m TokenMask) ContainsSentinel(value string) bool {
	return m.prefix != "" && strings.Contains(value, m.prefix)
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

type tokenReplacement struct {
	start       int
	end         int
	replacement string
}

func sentinelPrefix(data []byte) string {
	for salt := 0; ; salt++ {
		salted := strconv.AppendInt(nil, int64(salt), 10)
		salted = append(salted, ':')
		salted = append(salted, data...)
		sum := sha256.Sum256(salted)
		prefix := "__RUDDER_VARSUBST_" + hex.EncodeToString(sum[:8]) + "_"
		if !bytes.Contains(data, []byte(prefix)) {
			return prefix
		}
	}
}
