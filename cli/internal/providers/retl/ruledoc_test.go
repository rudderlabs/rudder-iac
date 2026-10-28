package retl_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	bingads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googleads"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	eventstream "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs/docstest"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderRuleDocs runs the provider's authored fragments through the real
// docs generator together with its live rules, asserting every rule resolves
// and passes the DocumentedRules validation invariants. The connection rule
// examples are then loaded as projects, so a fragment cannot drift from what
// validate reports.
//
// Both experimental kinds are on, as in the CI catalog, because the embedded
// fragments cover their rules too; without them those fragments would be
// orphans, which is exactly what the gen-rule-docs workflow sets the flags for.
func TestProviderRuleDocs(t *testing.T) {
	// http, Amplitude, Google Ads and Facebook Pixel are verified, so a default
	// CLI registers them and the examples using them run as written. Bing and
	// Customer.io Audience are unverified and registered only behind a flag; the
	// examples use them for Bing's mirror-only object mapping and Customer.io
	// Audience's own rETL flow, their fragments name the flag, and it is
	// registered unconditionally here.
	registry := definitions.NewRegistry()
	for _, definition := range []*definitions.DestinationDefinition{
		httpdest.NewDefinition(),
		am.NewDefinition(),
		googleads.NewDefinition(),
		facebookpixel.NewDefinition(),
		bingads.NewDefinition(),
		customerioaudience.NewDefinition(),
	} {
		require.NoError(t, registry.Register(definition))
	}
	p := retl.New(
		newDefaultMockClient(),
		retl.WithTableSupport(),
		retl.WithConnectionSupport(registry),
	)

	syntactic := p.SyntacticRules()
	semantic := p.SemanticRules()

	doc, verrs := docs.Generate(syntactic, semantic, p.RuleDocEntries(), "test", "2026-06-03T00:00:00Z")
	assert.Empty(t, verrs, "expected no validation errors, got: %v", verrs)
	require.Len(t, doc.Rules, len(syntactic)+len(semantic))

	var examples int
	for _, entry := range p.RuleDocEntries() {
		// The connection fragments run in full. The table and SQL model semantic
		// fragments run their valid examples only, which is enough to catch an
		// account that no longer validates (a newly required config key). Their
		// invalid examples, and the syntax fragments, predate this check
		// (references without /spec, duplicates against models their files do
		// not carry); DEX-915 fixes them and drops the filter.
		connection := strings.HasPrefix(entry.RuleID, "retl/connection/")
		semantic := slices.Contains([]string{"retl/table/semantic-valid", "retl/sqlmodel/semantic-valid"}, entry.RuleID)
		if !connection && !semantic {
			continue
		}
		for _, behaviour := range entry.MatchBehavior {
			examples += len(behaviour.Valid) + len(behaviour.Invalid)
			for _, example := range behaviour.Valid {
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadExample(t, registry, example.Files)
					assert.NoError(t, err)
					assert.Empty(t, diagnostics, "unexpected diagnostics:\n%s", docstest.Describe(diagnostics))
				})
			}
			if !connection {
				continue
			}
			for _, example := range behaviour.Invalid {
				t.Run(example.ExampleID, func(t *testing.T) {
					diagnostics, err := loadExample(t, registry, example.Files)
					docstest.AssertExpectedDiagnostics(t, entry.RuleID, example, diagnostics, err)
				})
			}
		}
	}

	// The filter is a prefix match on rule IDs: rename the connection rules or
	// move their fragments and every subtest above disappears silently, leaving
	// the docs.Generate assertions to report a pass on their own.
	require.NotZero(t, examples, "no rule doc example ran")
}

// loadExample runs the example files through project.Load, the path validate
// takes, with every provider that owns a kind the examples use.
func loadExample(t *testing.T, registry *definitions.Registry, files map[string]string) (validation.Diagnostics, error) {
	t.Helper()

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"retl":        retl.New(newDefaultMockClient(), retl.WithTableSupport(), retl.WithConnectionSupport(registry)),
		"eventstream": eventstream.New(source.NewMockSourceClient(), eventstream.WithDestinationRegistry(registry)),
		"destination": destination.NewProvider(nil, registry),
		"account":     accounts.NewProvider(nil),
	})
	require.NoError(t, err)

	return docstest.LoadDiagnostics(cp, files, varsubst.NewSubstitutor(exampleVariables))
}

// exampleVariables stands in for the var file a project passes with
// --var-file. The examples write account and destination secrets as
// references, as destination/secret-reference asks, and validate substitutes
// them before the destination config is checked against its patterns.
var exampleVariables = variables{
	"PG_PASSWORD":                            "example-password",
	"MY_GOOGLEADS_DESTINATION_CONVERSION_ID": "AW-123456789",
	"MY_PIXEL_DESTINATION_PIXEL_ID":          "123456789012345",
	"MY_AMPLITUDE_DESTINATION_API_KEY":       "example-api-key",
	"MY_CUSTOMERIO_AUDIENCE_API_KEY":         "example-api-key",
	"MY_CUSTOMERIO_AUDIENCE_APP_API_KEY":     "example-app-api-key",
	"MY_CUSTOMERIO_AUDIENCE_SITE_ID":         "example-site-id",
}

type variables map[string]string

func (v variables) Resolve(name string) (string, bool) {
	value, ok := v[name]
	return value, ok
}
