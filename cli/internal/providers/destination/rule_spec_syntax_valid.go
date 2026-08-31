package destination

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const SpecSyntaxValidRuleID = "destination/spec-syntax-valid"

// transformationRefRegex matches scalar transformation refs of the form
// #transformation:<id>, following the datacatalog reference convention.
var transformationRefRegex = regexp.MustCompile(
	fmt.Sprintf(`^#%s:([a-zA-Z0-9_-]+)$`, ttypes.TransformationSpecKind),
)

type specSyntaxValidRule struct {
	registry *definitions.Registry
}

// NewSpecSyntaxValidRule validates the destination spec envelope, that the
// (type, definition_version) pair is registered, the transformation ref
// format, and the per-type config via the definition registry.
func NewSpecSyntaxValidRule(registry *definitions.Registry) vrules.Rule {
	return &specSyntaxValidRule{registry: registry}
}

func (r *specSyntaxValidRule) ID() string {
	return SpecSyntaxValidRuleID
}

func (r *specSyntaxValidRule) Severity() vrules.Severity {
	return vrules.Error
}

func (r *specSyntaxValidRule) Description() string {
	return "destination spec envelope, registered type/version, transformation reference format and config must be valid"
}

func (r *specSyntaxValidRule) AppliesTo() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (r *specSyntaxValidRule) Examples() vrules.Examples {
	return vrules.Examples{}
}

// Validate builds all references relative to the spec root; prefixSpecReferences
// prepends /spec on every return path so results address the full document.
func (r *specSyntaxValidRule) Validate(ctx *vrules.ValidationContext) []vrules.ValidationResult {
	spec, unknownKeys, err := decodeDestinationSpec(ctx.Spec)
	if err != nil {
		return prefixSpecReferences([]vrules.ValidationResult{{
			Message: err.Error(),
		}})
	}

	results := make([]vrules.ValidationResult, 0, len(unknownKeys))
	for _, key := range unknownKeys {
		results = append(results, vrules.ValidationResult{
			Reference: "/" + key,
			Message:   fmt.Sprintf("unknown field %q", key),
		})
	}

	validationErrors, err := vrules.ValidateStructWithTagPriority(spec, []string{"mapstructure"})
	if err != nil {
		results = append(results, vrules.ValidationResult{Message: err.Error()})
		return prefixSpecReferences(results)
	}
	if len(validationErrors) > 0 {
		structResults := funcs.ParseMapstructureValidationErrors(validationErrors, reflect.TypeFor[DestinationSpec]())
		return prefixSpecReferences(append(results, structResults...))
	}

	if !r.registry.IsSupported(spec.Type) {
		results = append(results, vrules.ValidationResult{
			Reference: "/type",
			Message:   r.unsupportedTypeMessage(spec.Type),
		})
		return prefixSpecReferences(results)
	}

	versions, err := r.registry.Versions(spec.Type)
	if err != nil {
		results = append(results, vrules.ValidationResult{Reference: "/type", Message: err.Error()})
		return prefixSpecReferences(results)
	}
	if !slices.Contains(versions, spec.DefinitionVersion) {
		results = append(results, vrules.ValidationResult{
			Reference: "/definition_version",
			Message: fmt.Sprintf(
				"version not valid for destination type '%s'; valid versions: %s",
				spec.Type,
				joinInt64(versions),
			),
		})
		return prefixSpecReferences(results)
	}

	if spec.Transformation != "" && !transformationRefRegex.MatchString(spec.Transformation) {
		results = append(results, vrules.ValidationResult{
			Reference: "/transformation",
			Message:   "'transformation' is invalid: must be of pattern #transformation:<id>",
		})
	}

	def, err := r.registry.Get(spec.Type, spec.DefinitionVersion)
	if err != nil {
		results = append(results, vrules.ValidationResult{Reference: "/definition_version", Message: err.Error()})
		return prefixSpecReferences(results)
	}

	for _, configErr := range def.ValidateConfig(spec.Config) {
		results = append(results, vrules.ValidationResult{
			Reference: "/config" + configErr.Path,
			Message:   configErr.Message,
		})
	}

	results = append(results, sourceTypeKeyResults(def, spec)...)
	return prefixSpecReferences(results)
}

func (r *specSyntaxValidRule) unsupportedTypeMessage(destType string) string {
	supported := r.registry.SupportedTypes()
	if len(supported) == 0 {
		return fmt.Sprintf("destination type '%s' is not supported; no destination types are currently supported", destType)
	}
	return fmt.Sprintf("destination type '%s' is not supported; supported types: %s", destType, strings.Join(supported, ", "))
}

// sourceTypeKeyResults flags platform keys under source-type-scoped config
// blocks (connection_mode, use_native_sdk, consent_management) that the
// definition does not support — invalid regardless of project connections.
func sourceTypeKeyResults(def *definitions.RegisteredDefinition, spec *DestinationSpec) []vrules.ValidationResult {
	localKeys := def.LocalSourceTypeKeys()

	var results []vrules.ValidationResult
	for _, configKey := range def.SourceTypeConfigKeys() {
		// Non-map shapes are owned by the config model validation.
		block, ok := spec.Config[configKey].(map[string]any)
		if !ok {
			continue
		}

		for _, sourceType := range slices.Sorted(maps.Keys(block)) {
			if slices.Contains(localKeys, sourceType) {
				continue
			}
			results = append(results, vrules.ValidationResult{
				Reference: fmt.Sprintf("/config/%s/%s", configKey, sourceType),
				Message: fmt.Sprintf(
					"source type '%s' is not supported by destination type '%s'; supported source types: %s",
					sourceType, spec.Type, strings.Join(localKeys, ", "),
				),
			})
		}
	}
	return results
}

// decodeDestinationSpec decodes the raw spec map with the same mapstructure
// semantics the apply cycle uses, collecting all unknown envelope keys in one
// pass instead of failing on the first.
func decodeDestinationSpec(raw map[string]any) (*DestinationSpec, []string, error) {
	var (
		spec DestinationSpec
		md   mapstructure.Metadata
	)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:   &spec,
		Metadata: &md,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating spec decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return nil, nil, fmt.Errorf("decoding spec: %w", err)
	}

	unknown := make([]string, 0, len(md.Unused))
	for _, key := range md.Unused {
		// Nested unused paths (dotted) belong to config model validation.
		if strings.Contains(key, ".") {
			continue
		}
		unknown = append(unknown, key)
	}
	sort.Strings(unknown)

	return &spec, unknown, nil
}

func prefixSpecReferences(results []vrules.ValidationResult) []vrules.ValidationResult {
	for i := range results {
		results[i].Reference = "/spec" + results[i].Reference
	}
	return results
}

func joinInt64(values []int64) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, ", ")
}
