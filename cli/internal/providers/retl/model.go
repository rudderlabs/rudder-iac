package retl

import "fmt"

type SQLModelSpec struct {
	ID                   string  `json:"id" mapstructure:"id"`
	DisplayName          string  `json:"display_name" mapstructure:"display_name"`
	Description          string  `json:"description" mapstructure:"description"`
	File                 *string `json:"file" mapstructure:"file"`
	SQL                  *string `json:"sql" mapstructure:"sql"`
	AccountID            string  `json:"account_id" mapstructure:"account_id"`
	PrimaryKey           string  `json:"primary_key" mapstructure:"primary_key"`
	SourceDefinitionName string  `json:"source_definition_name" mapstructure:"source_definition_name"`
	Enabled              bool    `json:"enabled" mapstructure:"enabled"`
}

func ValidateSQLModelSpec(spec *SQLModelSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if spec.Description == "" {
		return fmt.Errorf("description is required")
	}
	if spec.File == nil && spec.SQL == nil {
		return fmt.Errorf("either file or sql is required")
	}
	if spec.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if spec.PrimaryKey == "" {
		return fmt.Errorf("primary_key is required")
	}
	if spec.SourceDefinitionName == "" {
		return fmt.Errorf("source_definition_name is required")
	}
	return nil
}
