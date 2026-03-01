package rules

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// LegacyVersionPatterns returns MatchPatterns for a kind matching both
// legacy spec versions (rudder/0.1 and rudder/v0.1).
func LegacyVersionPatterns(kind string) []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(kind, specs.SpecVersionV0_1),
		rules.MatchKindVersion(kind, specs.SpecVersionV0_1Variant),
	}
}
