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

func (t *MetadataSyntaxValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	metadata, err := decodeMetadata(ctx.Metadata)
	if err != nil {
		return []rules.ValidationResult{
			{

				Reference: "/metadata",
				Message:   "metadata should be a valid type",
			},
		}
	}

	if metadata.Name == "" {
		results = append(results, rules.ValidationResult{
			Reference: "/metadata/name",
			Message:   "metadata name should be string and not empty",
		})
	}

	if metadata.Import == nil {
		// If the metadata import is not present,
		// don't go validating the import section and return
		// early.
		return results
	}

	for idx, ws := range metadata.Import.Workspaces {
		if ws.WorkspaceID == "" {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/metadata/import/workspaces/%d/workspace_id", idx),
				Message:   "workspace_id should be a string and not empty",
			})
		}

		for resIdx, res := range ws.Resources {
			if res.LocalID == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/metadata/import/workspaces/%d/resources/%d/local_id", idx, resIdx),
					Message:   "local_id should be a string and not empty",
				})
			}
			if res.RemoteID == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/metadata/import/workspaces/%d/resources/%d/remote_id", idx, resIdx),
					Message:   "remote_id should be a string and not empty",
				})
			}
		}
	}

	return results
}

func decodeMetadata(input any) (*specs.Metadata, error) {

	var metadata specs.Metadata
	if input == nil {
		return &metadata, nil
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml",
		Result:  &metadata,
	})

	if err != nil {
		return &metadata, nil
	}

	err = decoder.Decode(input)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return &metadata, nil
}

func (r *MetadataSyntaxValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid: []string{
			heredoc.Doc(`
			metadata:
				import:
					workspaces:
						- resources:
							- local_id: "product_id"
							  remote_id: "some-upstream-id"
						  workspace_id: "w_123"
				name: "my-properties"
			`),
		},
		Invalid: []string{
			heredoc.Doc(`
			metadata:
				# missing name
				import:
					workspaces:
						- resources:
							- local_id: "product_id"
							  remote_id: "some-upstream-id"
						  workspace_id: "w_123"
			`),
			heredoc.Doc(`
			metadata:
				name: "my-properties"
				import:
					workspaces:
						- resources:
							- local_id: "product_id"
							  remote_id: "some-upstream-id"
						  # missing workspace_id
			`),
		},
	}
}
