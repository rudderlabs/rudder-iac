package funcs

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildLegacyReferenceRegex creates regex for legacy format: #/<kind>/<group>/<id>
func BuildLegacyReferenceRegex(kinds []string) *regexp.Regexp {
	if len(kinds) == 0 {
		return regexp.MustCompile(`^$`)
	}

	escapedKinds := make([]string, len(kinds))
	for i, kind := range kinds {
		escapedKinds[i] = regexp.QuoteMeta(kind)
	}
	kindsPattern := strings.Join(escapedKinds, "|")

	pattern := fmt.Sprintf(`^#/(%s)/([a-zA-Z0-9_-]+)/([a-zA-Z0-9_-]+)$`, kindsPattern)
	return regexp.MustCompile(pattern)
}
