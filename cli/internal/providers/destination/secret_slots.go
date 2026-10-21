package destination

import (
	"strconv"
	"strings"
)

// showsSecret reports whether the line validate prints for a result at
// configPath can show a secret value. The line belongs to the path, or to its
// nearest ancestor when the path is missing from the spec, and a list item's
// line shows its first property. So the answer is yes for a secret key, for a
// key leading to one, and for anything inside a list item that holds one.
func showsSecret(secretKeys []string, configPath string) bool {
	trimmed := strings.Trim(configPath, "/")
	if trimmed == "" {
		return false
	}

	// keys drops list indices, since secret keys name a list's members without
	// one. itemKeys counts the keys before the outermost list item.
	var (
		keys     []string
		itemKeys = -1
	)
	for _, segment := range strings.Split(trimmed, "/") {
		if _, err := strconv.Atoi(segment); err == nil {
			if itemKeys < 0 {
				itemKeys = len(keys)
			}
			continue
		}
		keys = append(keys, segment)
	}

	path := strings.Join(keys, ".")
	for _, secretKey := range secretKeys {
		if secretKey == path || strings.HasPrefix(secretKey, path+".") {
			return true
		}
		if itemKeys > 0 && strings.HasPrefix(secretKey, strings.Join(keys[:itemKeys], ".")+".") {
			return true
		}
	}
	return false
}
