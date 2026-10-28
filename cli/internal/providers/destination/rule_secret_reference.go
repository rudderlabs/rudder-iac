package destination

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const SecretReferenceRuleID = "destination/secret-reference"

type secretReferenceRule struct {
	registry *definitions.Registry
}

// NewSecretReferenceRule warns when a destination secret key holds anything but
// one variable reference with no default, so no secret value is written in a
// spec. It reads the spec before substitution, which also lets it report when a
// variable is still undefined.
func NewSecretReferenceRule(registry *definitions.Registry) vrules.Rule {
	return &secretReferenceRule{registry: registry}
}

func (r *secretReferenceRule) ID() string {
	return SecretReferenceRuleID
}

func (r *secretReferenceRule) Severity() vrules.Severity {
	return vrules.Warning
}

func (r *secretReferenceRule) Description() string {
	return "destination secret keys must hold one variable reference with no default, so no secret value is written in a spec; the check reads specs as written, before variable substitution"
}

func (r *secretReferenceRule) AppliesTo() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (r *secretReferenceRule) Examples() vrules.Examples {
	return vrules.Examples{}
}

// Validate is unused. The rule reads specs as written, through ValidateSources.
func (r *secretReferenceRule) Validate(*vrules.ValidationContext) []vrules.ValidationResult {
	return nil
}

func (r *secretReferenceRule) ValidateSources(specs map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	results := make(map[string][]vrules.ValidationResult)
	for path, ctx := range specs {
		for _, slot := range readSecretSlots(r.registry, ctx) {
			if slot.referencesVariable() {
				continue
			}
			results[path] = append(results[path], vrules.ValidationResult{
				Reference: slot.reference,
				Message:   secretReferenceMessage(slot),
			})
		}
	}
	return results
}

// secretReferenceMessage names the key and the variable to use, never the value.
func secretReferenceMessage(slot secretSlot) string {
	const fix = "set the value in a var file or a RUDDER_ environment variable"
	if slot.ref != nil {
		// The value is a reference with a default. The variable is already
		// named, so only the default has to go.
		return fmt.Sprintf("secret key %q must hold only a variable reference with no default; remove the default from the reference to %s and %s", slot.key, slot.ref.Name, fix)
	}
	if slot.suggested == "" {
		return fmt.Sprintf("secret key %q must hold only a variable reference with no default; %s", slot.key, fix)
	}
	return fmt.Sprintf("secret key %q must hold only a variable reference with no default, such as '{{ .%s }}'; %s", slot.key, slot.suggested, fix)
}
