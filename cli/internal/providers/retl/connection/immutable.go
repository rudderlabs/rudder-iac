package connection

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// immutableConfigKeys are the config keys the update request cannot carry, in
// the order checkImmutableUnchanged reports them.
var immutableConfigKeys = []string{"sync_behaviour", "cursor_column", "object", "event"}

// CheckImmutableChanges refuses, from the plan alone, an update that changes a
// config key the API cannot update in place. Update refuses the same change,
// but only when the syncer reaches that connection, after the resources ahead
// of it in the plan are already applied; here it fails before anything is
// touched and before the plan is shown, so --dry-run reports it too (DEX-1020).
//
// A connection whose source or destination changes is replaced rather than
// updated, and the replacement is created with the new config, so it is left
// alone.
func CheckImmutableChanges(diff *differ.Diff) error {
	prefix := ResourceType + ":"
	var problems []string
	for _, urn := range slices.Sorted(maps.Keys(diff.UpdatedResources)) {
		if !strings.HasPrefix(urn, prefix) {
			continue
		}
		diffs := diff.UpdatedResources[urn].Diffs
		if _, moved := diffs[SourceKey]; moved {
			continue
		}
		if _, moved := diffs[DestinationKey]; moved {
			continue
		}
		for _, key := range immutableConfigKeys {
			if changed, ok := changedConfigKey(diffs, key); ok {
				problems = append(problems, fmt.Sprintf(
					"%s: %s is immutable (%v -> %v); delete and recreate the connection to apply it",
					urn, key, changed.SourceValue, changed.TargetValue))
				break
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("cannot update rETL connection config that is immutable:\n  %s", strings.Join(problems, "\n  "))
}

// changedConfigKey finds the property diff for a config key. A nested key such
// as event.name reports under "config.event.name", so the key is matched as a
// path prefix; the nested case has no single before/after, so the first match
// stands for it.
func changedConfigKey(diffs map[string]differ.PropertyDiff, key string) (differ.PropertyDiff, bool) {
	path := ConfigKey + "." + key
	for _, property := range slices.Sorted(maps.Keys(diffs)) {
		if property == path || strings.HasPrefix(property, path+".") {
			return diffs[property], true
		}
	}
	return differ.PropertyDiff{}, false
}
