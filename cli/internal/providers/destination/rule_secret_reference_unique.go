package destination

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const SecretReferenceUniqueRuleID = "destination/secret-reference-unique"

type secretReferenceUniqueRule struct {
	registry *definitions.Registry
}

// NewSecretReferenceUniqueRule warns when one variable fills two destination
// secret keys. Import builds a name from the spec id and the key, joined by
// "_", and both may contain "_" themselves, so two destinations can be given
// the same name: shop with android_api_key and shop-android with api_key both
// get SHOP_ANDROID_API_KEY, and one value would reach both.
func NewSecretReferenceUniqueRule(registry *definitions.Registry) vrules.Rule {
	return &secretReferenceUniqueRule{registry: registry}
}

func (r *secretReferenceUniqueRule) ID() string {
	return SecretReferenceUniqueRuleID
}

func (r *secretReferenceUniqueRule) Severity() vrules.Severity {
	return vrules.Warning
}

func (r *secretReferenceUniqueRule) Description() string {
	return "a variable referenced by a destination secret key must not fill another secret key, since names built from an id and a key can collide"
}

func (r *secretReferenceUniqueRule) AppliesTo() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (r *secretReferenceUniqueRule) Examples() vrules.Examples {
	return vrules.Examples{}
}

// Validate is unused. The rule reads specs as written, through ValidateSources.
func (r *secretReferenceUniqueRule) Validate(*vrules.ValidationContext) []vrules.ValidationResult {
	return nil
}

// ValidateSources counts only keys holding exactly one reference with no
// default. Anything else already gets a secret-reference warning.
func (r *secretReferenceUniqueRule) ValidateSources(specs map[string]*vrules.ValidationContext) map[string][]vrules.ValidationResult {
	byVariable := make(map[string][]secretSlot)
	for path, ctx := range specs {
		for _, slot := range readSecretSlots(r.registry, path, ctx.Source) {
			if slot.variable == "" {
				continue
			}
			byVariable[slot.variable] = append(byVariable[slot.variable], slot)
		}
	}

	results := make(map[string][]vrules.ValidationResult)
	for _, variable := range slices.Sorted(maps.Keys(byVariable)) {
		slots := byVariable[variable]
		if len(slots) < 2 {
			continue
		}
		slices.SortFunc(slots, func(a, b secretSlot) int {
			return cmp.Or(cmp.Compare(a.file, b.file), cmp.Compare(a.reference, b.reference))
		})

		for i, slot := range slots {
			others := slices.Concat(slots[:i], slots[i+1:])
			results[slot.file] = append(results[slot.file], vrules.ValidationResult{
				Reference: slot.reference,
				Message:   sharedVariableMessage(slot, others),
				Sensitive: true,
			})
		}
	}
	return results
}

func sharedVariableMessage(slot secretSlot, others []secretSlot) string {
	locations := make([]string, 0, len(others))
	for _, other := range others {
		locations = append(locations, fmt.Sprintf("secret key %q of destination %q", other.key, other.specID))
	}

	return fmt.Sprintf(
		"secret key %q of destination %q shares variable %s with %s; give each secret key its own variable unless they hold the same value",
		slot.key, slot.specID, slot.variable, joinAnd(locations),
	)
}

// joinAnd joins items as "a", "a and b" or "a, b and c".
func joinAnd(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
