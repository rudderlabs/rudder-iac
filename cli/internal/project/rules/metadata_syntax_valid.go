package rules

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type MetadataSyntaxValidRule struct {
}

func NewMetadataSyntaxValidRule() rules.Rule {
	return &MetadataSyntaxValidRule{}
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

func (r *MetadataSyntaxValidRule) AppliesTo() []string {
	return []string{"*"}
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
	results, err := rules.ValidateStruct(metadata, "/metadata")
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/metadata",
				Message:   fmt.Sprintf("metadata needs to be valid: %s", err.Error()),
			},
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
