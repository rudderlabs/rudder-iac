package datacatalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestProvider_SupportedMatchPatterns(t *testing.T) {
	t.Parallel()

	p := datacatalog.New(&datacatalog.EmptyCatalog{})

	var want []vrules.MatchPattern
	for _, kind := range []string{
		localcatalog.KindProperties,
		localcatalog.KindEvents,
		localcatalog.KindCategories,
		localcatalog.KindCustomTypes,
	} {
		want = append(want, prules.LegacyVersionPatterns(kind)...)
		want = append(want, prules.V1VersionPatterns(kind)...)
	}
	want = append(want, prules.LegacyVersionPatterns(localcatalog.KindTrackingPlans)...)
	want = append(want, prules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)...)

	assert.ElementsMatch(t, want, p.SupportedMatchPatterns())
}
