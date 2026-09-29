package ingest

import (
	"strings"
	"unicode"
)

// invisibleRunes is copied from rudder-go-kit sanitize (v0.80.0), which the
// oracle uses to clean identifiers before it checks or hashes them.
var invisibleRunes = map[rune]bool{
	'\u0000': true, '\u0009': true, '\u00A0': true, '\u00AD': true, '\u034F': true,
	'\u061C': true, '\u115F': true, '\u1160': true, '\u17B4': true, '\u17B5': true,
	'\u180E': true, '\u2000': true, '\u2001': true, '\u2002': true, '\u2003': true,
	'\u2004': true, '\u2005': true, '\u2006': true, '\u2007': true, '\u2008': true,
	'\u2009': true, '\u200A': true, '\u200B': true, '\u200C': true, '\u200D': true,
	'\u200E': true, '\u200F': true, '\u202F': true, '\u205F': true, '\u2060': true,
	'\u2061': true, '\u2062': true, '\u2063': true, '\u2064': true, '\u206A': true,
	'\u206B': true, '\u206C': true, '\u206D': true, '\u206E': true, '\u206F': true,
	'\u3000': true, '\u2800': true, '\u3164': true, '\uFEFF': true, '\uFFA0': true,
}

func sanitizeAndTrim(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if invisibleRunes[r] || !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, s))
}
