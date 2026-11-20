package specdocs

import (
	"slices"
	"strings"
	"unicode"
)

// candidate is a value to suggest and the other names it goes by.
type candidate struct {
	value   string
	aliases []string
}

// suggest returns the candidates input most likely meant: first the names that
// equal it once case and punctuation are ignored, so a caller taking the first
// suggestion gets one of them, then the names that contain it or are contained
// in it, and the values a typo or two away.
func suggest(input string, candidates []candidate) []string {
	needle := normalize(input)
	if needle == "" {
		return []string{}
	}

	var exact, near []string
	for _, c := range candidates {
		switch {
		case equals(needle, c):
			exact = append(exact, c.value)
		case matches(needle, c):
			near = append(near, c.value)
		}
	}
	slices.Sort(exact)
	slices.Sort(near)
	return append(append([]string{}, slices.Compact(exact)...), slices.Compact(near)...)
}

func equals(needle string, c candidate) bool {
	for _, name := range append([]string{c.value}, c.aliases...) {
		if normalize(name) == needle {
			return true
		}
	}
	return false
}

func matches(needle string, c candidate) bool {
	for _, name := range append([]string{c.value}, c.aliases...) {
		hay := normalize(name)
		if len(needle) >= 3 && strings.Contains(hay, needle) {
			return true
		}
		if len(hay) >= 3 && strings.Contains(needle, hay) {
			return true
		}
	}

	allowed := 1
	if len(needle) > 4 {
		allowed = 2
	}
	return editDistance(needle, normalize(c.value)) <= allowed
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev = curr
	}
	return prev[len(b)]
}
