package models

// Re-export the main schema models to eliminate duplication
// This removes ~130 lines of duplicate struct definitions
import "github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"

type Schema = models.Schema
type SchemasFile = models.SchemasFile
type SchemasResponse = models.SchemasResponse
