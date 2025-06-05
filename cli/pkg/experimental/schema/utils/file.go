package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/experimental/schema/models"
)

// ReadSchemasFile reads a schemas JSON file and returns the parsed structure
func ReadSchemasFile(filepath string) (*models.SchemasFile, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	var schemasFile models.SchemasFile
	if err := json.Unmarshal(data, &schemasFile); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filepath, err)
	}

	return &schemasFile, nil
}

// WriteSchemasFile writes the schemas structure to a JSON file with proper formatting
func WriteSchemasFile(filepath string, schemasFile *models.SchemasFile, indent int) error {
	var data []byte
	var err error

	if indent > 0 {
		indentStr := ""
		for i := 0; i < indent; i++ {
			indentStr += " "
		}
		data, err = json.MarshalIndent(schemasFile, "", indentStr)
	} else {
		data, err = json.Marshal(schemasFile)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	return nil
}

// WriteJSONFile writes data to a JSON file with proper indentation
func WriteJSONFile(filename string, data interface{}, indent int) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if indent > 0 {
		encoder.SetIndent("", fmt.Sprintf("%*s", indent, ""))
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}
