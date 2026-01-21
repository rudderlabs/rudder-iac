package rules

import (
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type MetadataNameValidRule struct {
	pattern string
}

func NewMetadataNameValidRule() rules.Rule {
	return &MetadataNameValidRule{
		pattern: "^[a-zA-Z0-9_-]+$",
	}
}

func (r *MetadataNameValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	metadata, err := decodeMetadata(ctx.Metadata)
	if err != nil {
		// The reson why we are returning an empty slice is
		// because every rule only validates its own part of the datapoint.
		// If we are unable to decode or extract the datapoint,
		// we should not raise an error as it can create duplicate errors.
		return []rules.ValidationResult{}
	}

	if metadata.Name == "" {
		return nil
	}

	matched, _ := regexp.MatchString(r.pattern, metadata.Name)
	if !matched {
		return []rules.ValidationResult{
			{
				Reference: "/metadata/name",
				Message:   fmt.Sprintf("metadata name should match pattern: %s", r.pattern),
			},
		}
	}
	return nil
}

func (r *MetadataNameValidRule) ID() string {
	return "project/metadata-name-valid"
}

func (r *MetadataNameValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *MetadataNameValidRule) Description() string {
	return fmt.Sprintf("metadata name should match pattern: %s", r.pattern)
}

func (r *MetadataNameValidRule) AppliesTo() []string {
	return []string{"*"}
}

func (r *MetadataNameValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid:   []string{"user_tracking_events", "api-usage-properties"},
		Invalid: []string{"analytics@v2, _invalid_name_start", "name with spaces"},
	}
}
