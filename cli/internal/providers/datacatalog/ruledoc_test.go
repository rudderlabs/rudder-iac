package datacatalog_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderRuleDocs runs the provider's authored fragments through the real
// docs generator together with its live rules, asserting every rule resolves
// and passes the DocumentedRules validation invariants.
func TestProviderRuleDocs(t *testing.T) {
	p := datacatalog.New(&MockCategoryCatalog{})

	syntactic := p.SyntacticRules()
	semantic := p.SemanticRules()

	doc, verrs := docs.Generate(syntactic, semantic, p.RuleDocEntries(), "test", "2026-06-03T00:00:00Z")
	assert.Empty(t, verrs, "expected no validation errors, got: %v", verrs)
	require.Len(t, doc.Rules, len(syntactic)+len(semantic))
}
