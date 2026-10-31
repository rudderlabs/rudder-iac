package importmatcher

import "strings"

// SameName reports whether a local and a remote name denote the same resource
// upstream. Source names are unique case-insensitively across a workspace —
// config-backend enforces it with LOWER(name) = LOWER(:name) — so a remote
// "Orders" and a local "orders" are one source, and a matcher that compared
// them exactly wrote a second spec for a source that already existed.
//
// Folded with strings.ToLower rather than strings.EqualFold, for the same
// reason as the name-clash rule: ToLower mirrors Postgres LOWER(), EqualFold
// folds less. Using EqualFold here would fail to link a pair that the server
// considers identical, which is the bug this fixes in a subtler form. Not
// trimmed, because stored names were never normalized.
func SameName(local, remote string) bool {
	return strings.ToLower(local) == strings.ToLower(remote)
}
