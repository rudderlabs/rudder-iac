package validate

import (
	"fmt"
	"regexp"
	"strings"

	catalog "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

// CategoryValidator validates category entities
type CategoryValidator struct {
}

// Category name validation regex: ^[A-Z_a-z][\s\w,.-]{2,64}$
var categoryNameRegex = regexp.MustCompile(`^[A-Z_a-z][\s\w,.-]{2,64}$`)

func (cv *CategoryValidator) Validate(dc *catalog.DataCatalog) []ValidationError {
	log.Info("validating categories in catalog")

	var errors []ValidationError

	// Track category names and IDs for duplicate validation
	categoryNames := make(map[string]bool)
	categoryIDs := make(map[string]bool)

	// Categories required keys and format validation
	for group, categories := range dc.Categories {
		for _, category := range categories {
			reference := fmt.Sprintf("#/categories/%s/%s", group, category.LocalID)

			// Check mandatory fields
			if category.LocalID == "" || category.Name == "" {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("id and name fields on category are mandatory"),
					Reference: reference,
				})
				continue
			}

			// Validate category name doesn't have leading or trailing whitespace
			if category.Name != strings.TrimSpace(category.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("category name cannot have leading or trailing whitespace characters"),
					Reference: reference,
				})
				continue
			}

			// Category name format validation
			if !categoryNameRegex.MatchString(category.Name) {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("category name must start with a letter (upper/lower case) or underscore, followed by 2-64 characters including spaces, word characters, commas, periods, and hyphens"),
					Reference: reference,
				})
				continue
			}

			// Check for duplicate category names
			if categoryNames[category.Name] {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate category name: %s", category.Name),
					Reference: reference,
				})
				continue
			}
			categoryNames[category.Name] = true

			// Check for duplicate category IDs
			if categoryIDs[category.LocalID] {
				errors = append(errors, ValidationError{
					error:     fmt.Errorf("duplicate category id: %s", category.LocalID),
					Reference: reference,
				})
			}
			categoryIDs[category.LocalID] = true
		}
	}

	return errors
}
