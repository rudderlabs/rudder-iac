package ruledoc_test

import (
	"strconv"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ruledoc"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuild_GatekeeperOnly proves Build assembles a valid catalog from a
// provider in isolation — no client, config, or auth. An empty provider still
// yields the four project-level gatekeeper rules (always registered by
// BuildRegistry), and their embedded fragments cover them exactly, so the
// catalog validates with zero errors. This is the assembly seam that package
// app exercises end-to-end against the real composed providers.
//
// Two gatekeeper rules consult the provider for context: duplicate-urn and the
// metadata import cross-check both call provider.ParseSpec to learn the URNs a
// spec declares, and resource-kind-version-valid keys off the provider's
// supported kind/version patterns. Their authored examples document that
// behaviour, so the mock is given just enough provider context — event-URN
// extraction and the events/transformation patterns — for those examples to
// run as documented. Everything else still flows through the real engine.
func TestBuild_GatekeeperOnly(t *testing.T) {
	cp := &testutils.MockProvider{
		// The gatekeeper examples span the events, properties and transformation
		// kinds. Declaring these patterns lets spec-syntax-valid accept every
		// example's kind/version, while resource-kind-version-valid recognises
		// transformation as a known kind and rudder/0.1 as a known version yet
		// still flags transformation@rudder/0.1 as an unregistered pairing.
		MatchPatterns: []vrules.MatchPattern{
			vrules.MatchKindVersion("events", "rudder/v1"),
			vrules.MatchKindVersion("events", "rudder/0.1"),
			vrules.MatchKindVersion("properties", "rudder/v1"),
			vrules.MatchKindVersion("transformation", "rudder/v1"),
		},
		// duplicate-urn and the metadata import cross-check resolve URNs via
		// ParseSpec; mirror how the datacatalog provider derives event URNs so
		// those examples collide / resolve exactly as documented.
		ParseSpecFn: extractEventURNs,
	}

	// Gatekeeper rules are syntactic, so no provider factory is exercised here;
	// nil is intentional and asserts no semantic example sneaks into this path.
	doc, verrs, err := ruledoc.Build(cp, "test", "2026-01-01T00:00:00Z", docs.ModeSubset, nil)
	require.NoError(t, err)
	assert.Empty(t, verrs, "gatekeeper fragments must cover the gatekeeper rules exactly")

	// spec-syntax-valid, resource-kind-version-valid, metadata-syntax-valid,
	// duplicate-urn — the rules BuildRegistry always registers.
	require.Len(t, doc.Rules, 4)
	for _, r := range doc.Rules {
		require.Len(t, r.AppliesTo, 1)
		assert.Equal(t, "*", r.AppliesTo[0].Kind)
		assert.Equal(t, "*", r.AppliesTo[0].Version)
	}
}

// extractEventURNs mirrors the datacatalog provider's URN extraction for the
// events kind: each entry under spec.events with an id yields an "event:<id>"
// URN. It is intentionally minimal — only the events kind, the only kind the
// gatekeeper fragments declare.
func extractEventURNs(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	parsed := &specs.ParsedSpec{URNs: []specs.URNEntry{}}
	if s == nil || s.Spec == nil {
		return parsed, nil
	}

	rawEvents, ok := s.Spec["events"].([]any)
	if !ok {
		return parsed, nil
	}

	for i, raw := range rawEvents {
		event, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, ok := event["id"].(string)
		if !ok || id == "" {
			continue
		}
		parsed.URNs = append(parsed.URNs, specs.URNEntry{
			URN:             resources.URN(id, "event"),
			JSONPointerPath: "/spec/events/" + strconv.Itoa(i) + "/id",
		})
	}

	return parsed, nil
}
