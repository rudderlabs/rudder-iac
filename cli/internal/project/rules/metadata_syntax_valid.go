package rules

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

type MetadataSyntaxValidRule struct {
	parseSpec ParseSpecFunc
	appliesTo []rules.MatchPattern
}

func NewMetadataSyntaxValidRule(parseSpec ParseSpecFunc, appliesToVersions []string) rules.Rule {
	patterns := make([]rules.MatchPattern, len(appliesToVersions))
	for i, v := range appliesToVersions {
		patterns[i] = rules.MatchPattern{Kind: "*", Version: v}
	}
	return &MetadataSyntaxValidRule{
		parseSpec: parseSpec,
		appliesTo: patterns,
	}
}

func (r *MetadataSyntaxValidRule) ID() string {
	return "project/metadata-syntax-valid"
}

func (r *MetadataSyntaxValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *MetadataSyntaxValidRule) Description() string {
	return "metadata syntax must be valid"
}

func (r *MetadataSyntaxValidRule) AppliesTo() []rules.MatchPattern {
	return r.appliesTo
}

func (r *MetadataSyntaxValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid: []string{
			heredoc.Doc(`
metadata:
  name: my-project
`),
			heredoc.Doc(`
metadata:
  name: my-project
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - local_id: src-local
            remote_id: src-remote-456
`),
		},
		Invalid: []string{
			// Invalid: missing required 'name' field in metadata
			heredoc.Doc(`
metadata: # name is missing
  import:
    workspaces:
      - workspace_id: ws-123
`),
			// Invalid: missing required 'workspace_id' field in workspaces entry
			heredoc.Doc(`
metadata:
  name: my-project
  import:
    workspaces: # missing workspace_id
      - resources:
          - local_id: src-local
            remote_id: src-remote-456
`),
		},
	}
}

func (r *MetadataSyntaxValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	// If the metadata is empty, we don't validate it
	// as we only validate the metadata when we have somedata in it.
	if len(ctx.Metadata) == 0 {
		return nil
	}

	metadata, err := decodeMetadata(ctx.Metadata)
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/metadata",
				Message:   fmt.Sprintf("metadata needs to be valid: %s", err.Error()),
			},
		}
	}

	// ValidateStruct returns a list of validation results
	// or nil. We pass "/metadata" as the base path since we're
	// validating metadata which lives under the metadata key in YAML.
	validationErrors, err := rules.ValidateStruct(metadata, "/metadata")
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/metadata",
				Message:   fmt.Sprintf("metadata needs to be valid: %s", err.Error()),
			},
		}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)
	// Before returning the results, simply prepend the metadata
	// path to the reference to allow for unique identification.
	results = lo.Map(results, func(result rules.ValidationResult, _ int) rules.ValidationResult {
		return rules.ValidationResult{
			Reference: "/metadata" + result.Reference,
			Message:   result.Message,
		}
	})

	// If struct-level validation already has errors, return early
	// to avoid confusing errors from the import cross-check.
	if len(results) > 0 {
		return results
	}

	results = append(results, r.validateImportIDs(ctx, metadata)...)

	return results
}

// validateImportIDs checks that every local_id in the metadata import block
// exists as an external ID in the spec. This ensures that imported resources
// are actually defined in the spec body.
func (r *MetadataSyntaxValidRule) validateImportIDs(ctx *rules.ValidationContext, metadata *specs.Metadata) []rules.ValidationResult {
	if metadata.Import == nil {
		return nil
	}

	parsed, err := r.parseSpec(ctx.FilePath, &specs.Spec{
		Version:  ctx.Version,
		Kind:     ctx.Kind,
		Metadata: ctx.Metadata,
		Spec:     ctx.Spec,
	})
	if err != nil {
		return nil
	}

	externalIDSet := make(map[string]struct{}, len(parsed.LocalIDs))
	for _, localID := range parsed.LocalIDs {
		externalIDSet[localID.ID] = struct{}{}
	}

	var results []rules.ValidationResult
	for i, workspace := range metadata.Import.Workspaces {
		for j, resource := range workspace.Resources {
			_, exists := externalIDSet[resource.LocalID]

			if exists {
				continue
			}

			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/metadata/import/workspaces/%d/resources/%d/local_id", i, j),
				Message:   fmt.Sprintf("local_id '%s' from import metadata not found in spec", resource.LocalID),
			})
		}
	}

	return results
}

func decodeMetadata(m map[string]any) (*specs.Metadata, error) {
	var metadata specs.Metadata

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml",
		Result:  &metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	err = decoder.Decode(m)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return &metadata, nil
}
