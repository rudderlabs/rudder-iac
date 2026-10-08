package connection

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// CheckImmutableChanges refuses, from the plan alone, an update that changes a
// config key the API cannot update in place. Update refuses the same change,
// but only when the syncer reaches that connection, after the resources ahead
// of it in the plan are already applied; here it fails before anything is
// touched and before the plan is shown, so --dry-run reports it too (DEX-1020).
//
// The differ reports a nested change under the top-level key only, so the
// check compares the whole stored and desired config instead of looking for
// a dotted key such as config.sync_behaviour.
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
		config, ok := diffs[ConfigKey]
		if !ok {
			continue
		}
		// A config that does not parse is reported by Update with the
		// connection's name; this check only adds the early refusal.
		desired, err := configFromMap(config.TargetValue)
		if err != nil {
			continue
		}
		stored, err := configFromMap(config.SourceValue)
		if err != nil {
			continue
		}
		if err := checkImmutableUnchanged(desired, stored); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %s", urn, err))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("cannot update rETL connection config that is immutable:\n  %s", strings.Join(problems, "\n  "))
}
