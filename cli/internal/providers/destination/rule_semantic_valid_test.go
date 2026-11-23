package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func destinationGraphResource(id, displayName string) *resources.Resource {
	return resources.NewResource(
		id,
		DestinationResourceType,
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&DestinationResource{ID: id, DisplayName: displayName}),
	)
}

func transformationGraphResource(id string) *resources.Resource {
	return resources.NewResource(
		id,
		ttypes.TransformationResourceType,
		resources.ResourceData{},
		[]string{},
	)
}

func runSemanticRule(t *testing.T, spec map[string]any, graph *resources.Graph) []vrules.ValidationResult {
	t.Helper()

	rule := NewSemanticValidRule()
	return rule.Validate(&vrules.ValidationContext{
		Spec:    spec,
		Kind:    DestinationSpecKind,
		Version: "rudder/v1",
		Graph:   graph,
	})
}

func TestSemanticValidRuleMetadata(t *testing.T) {
	t.Parallel()

	rule := NewSemanticValidRule()
	assert.Equal(t, SemanticValidRuleID, rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.NotEmpty(t, rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(DestinationSpecKind), rule.AppliesTo())
}

func TestSemanticValidRuleNoGraph(t *testing.T) {
	t.Parallel()

	results := runSemanticRule(t, validSpecMap(), nil)
	assert.Nil(t, results)
}

func TestSemanticValidRuleTransformationExists(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))
	graph.AddResource(transformationGraphResource("my-transformation"))

	spec := validSpecMap()
	spec["transformation"] = "#transformation:my-transformation"

	assert.Empty(t, runSemanticRule(t, spec, graph))
}

func TestSemanticValidRuleTransformationMissing(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))

	spec := validSpecMap()
	spec["transformation"] = "#transformation:ghost"

	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/transformation",
			Message:   "transformation 'ghost' not found in the project",
		},
	}, runSemanticRule(t, spec, graph))
}

func TestSemanticValidRuleNoTransformationRef(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))

	assert.Empty(t, runSemanticRule(t, validSpecMap(), graph))
}

func TestSemanticValidRuleDuplicateDisplayName(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))
	graph.AddResource(destinationGraphResource("webhook-copy", "Production Webhook"))

	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/display_name",
			Message:   "duplicate display_name 'Production Webhook' within kind 'destination'",
		},
	}, runSemanticRule(t, validSpecMap(), graph))
}

func TestSemanticValidRuleUniqueDisplayNames(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))
	graph.AddResource(destinationGraphResource("webhook-stage", "Staging Webhook"))

	assert.Empty(t, runSemanticRule(t, validSpecMap(), graph))
}

func TestSemanticValidRuleSkipsForeignRawData(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))
	// A destination-typed resource whose RawData is not *DestinationResource
	// must be skipped, not counted or panicked on.
	graph.AddResource(resources.NewResource(
		"weird",
		DestinationResourceType,
		resources.ResourceData{},
		[]string{},
		resources.WithRawData("not-a-destination"),
	))

	assert.Empty(t, runSemanticRule(t, validSpecMap(), graph))
}

func accountGraphResource(id string) *resources.Resource {
	return resources.NewResource(
		id,
		accounts.AccountResourceType,
		resources.ResourceData{},
		[]string{},
	)
}

func specWithAccount(value string) map[string]any {
	spec := validSpecMap()
	spec["config"] = map[string]any{
		"webhook_url":       "https://example.com/hook",
		"rudder_account_id": value,
	}
	return spec
}

func TestSemanticValidRuleAccountExists(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))
	graph.AddResource(accountGraphResource("google-ads-prod"))

	assert.Empty(t, runSemanticRule(t, specWithAccount("#account:google-ads-prod"), graph))
}

// Without this, a reference to an account that has no spec passes validate and
// fails mid-apply with a bare "referred resource does not exist".
func TestSemanticValidRuleAccountMissing(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))

	results := runSemanticRule(t, specWithAccount("#account:ghost"), graph)
	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/config/rudder_account_id",
			Message: "account 'ghost' not found in the project. Adopt it with 'rudder-cli import workspace', " +
				"or write the account's ID here instead of a reference",
		},
	}, results)
}

// A literal account id is not a reference and is nobody's to resolve.
func TestSemanticValidRuleLiteralAccountIDIsNotChecked(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))

	assert.Empty(t, runSemanticRule(t, specWithAccount("2fS9kQwPzX7rTnV4yLbH8mJcD1e"), graph))
}

// A malformed reference belongs to the syntactic rule, which reports it with
// the expected format; reporting it twice would be noise.
func TestSemanticValidRuleIgnoresMalformedAccountRef(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(destinationGraphResource("webhook-prod", "Production Webhook"))

	assert.Empty(t, runSemanticRule(t, specWithAccount("#account:"), graph))
}
