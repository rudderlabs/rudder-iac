package retl

import "fmt"

type SQLModelSpec struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Description string  `json:"description"`
	File        *string `json:"file"`
	SQL         *string `json:"sql"`
	AccountID   string  `json:"account_id"`
	PrimaryKey  string  `json:"primary_key"`
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
	return nil
}
