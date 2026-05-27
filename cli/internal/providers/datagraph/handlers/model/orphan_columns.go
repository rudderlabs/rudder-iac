package model

import (
	"fmt"
	"sort"
)

// FindOrphanedColumns returns the sorted, deduplicated set of column names
// that are present in remote but missing from local. These are the rows the
// v1 column-metadata batch-upsert endpoint will leave behind: partial-merge
// semantics mean omitting a name from yaml does NOT delete the remote row.
//
// Entries with a missing or non-string "name" are silently skipped — the
// helper only attests to names it can compare reliably. The result is nil
// (not an empty slice) when there are no orphans, matching the "no warning"
// signal used by the plan-time emitter.
func FindOrphanedColumns(local, remote []map[string]any) []string {
	if len(remote) == 0 {
		return nil
	}

	localNames := make(map[string]struct{}, len(local))
	for _, entry := range local {
		if name, ok := stringName(entry); ok {
			localNames[name] = struct{}{}
		}
	}

	orphans := make(map[string]struct{})
	for _, entry := range remote {
		name, ok := stringName(entry)
		if !ok {
			continue
		}
		if _, kept := localNames[name]; kept {
			continue
		}
		orphans[name] = struct{}{}
	}

	if len(orphans) == 0 {
		return nil
	}

	out := make([]string, 0, len(orphans))
	for name := range orphans {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// FormatOrphanColumnWarning returns the user-facing warning string for a
// single orphaned column name. Wording is fixed by
// data-graph-column-metadata-clients-lld.md ("Provider integration") and
// pinned by TestFormatOrphanColumnWarning — changing it is a user-visible
// message change.
func FormatOrphanColumnWarning(name string) string {
	return fmt.Sprintf("metadata for %s will remain in the workspace; v1 has no clear/delete path", name)
}

func stringName(entry map[string]any) (string, bool) {
	raw, ok := entry["name"]
	if !ok {
		return "", false
	}
	name, ok := raw.(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}
