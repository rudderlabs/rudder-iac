package golang

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
)

// initialisms are written fully upper-case in identifiers (Go style guide / golint list).
var initialisms = map[string]bool{
	"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true, "DNS": true,
	"EOF": true, "GUID": true, "HTML": true, "HTTP": true, "HTTPS": true, "ID": true,
	"IP": true, "JSON": true, "LHS": true, "QPS": true, "RAM": true, "RHS": true,
	"RPC": true, "SLA": true, "SMTP": true, "SQL": true, "SSH": true, "TCP": true,
	"TLS": true, "TTL": true, "UDP": true, "UI": true, "UID": true, "UUID": true,
	"URI": true, "URL": true, "UTF8": true, "VM": true, "XML": true, "XMPP": true,
	"XSRF": true, "XSS": true,
}

// pascalCase turns a tracking-plan name into an identifier: words split at
// every rune that cannot appear in one and at case boundaries, each word
// upper-cased on its first rune or, if it is an initialism, as a whole.
func pascalCase(name string) (string, error) {
	words := core.SplitIntoWords(core.SanitizeForIdentifier(name))
	if len(words) == 0 {
		return "", fmt.Errorf("name %q has no letters or digits to build a Go identifier from", name)
	}

	// SCREAMING_SNAKE names would otherwise stay shouting (PAYMENTMETHOD); a
	// single upper-case word such as GET is kept as written.
	screaming := len(words) > 1 && !strings.ContainsFunc(name, func(r rune) bool {
		return unicode.IsLetter(r) && !unicode.IsUpper(r)
	})

	var b strings.Builder
	for _, word := range words {
		if upper := strings.ToUpper(word); initialisms[upper] {
			b.WriteString(upper)
			continue
		}
		first, size := utf8.DecodeRuneInString(word)
		rest := word[size:]
		if screaming {
			rest = strings.ToLower(rest)
		}
		b.WriteRune(unicode.ToUpper(first))
		b.WriteString(rest)
	}
	return b.String(), nil
}

// fieldName returns the exported struct field name for a property. Go exports
// only names starting with an upper-case letter, so a name starting with a
// digit or a caseless letter (CJK) gets an X prefix.
func fieldName(name string) (string, error) {
	n, err := pascalCase(name)
	if err != nil {
		return "", err
	}
	if first, _ := utf8.DecodeRuneInString(n); !unicode.IsUpper(first) {
		n = "X" + n
	}
	return n, nil
}
