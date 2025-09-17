package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// TrackingPlanConverter handles converting JSON files to YAML format
type TrackingPlanConverter struct {
	inputDir  string
	outputDir string
}

// NewTrackingPlanConverter creates a new TrackingPlanConverter
func NewTrackingPlanConverter(inputDir, outputDir string) *TrackingPlanConverter {
	return &TrackingPlanConverter{inputDir: inputDir, outputDir: outputDir}
}

// Convert converts JSON files to YAML format
func (c *TrackingPlanConverter) Convert() error {
	// Create output directory structure
	yamlDir := filepath.Join(c.outputDir, "yaml")
	if err := os.MkdirAll(yamlDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"tracking-plans", "events", "properties", "custom-types", "categories"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(yamlDir, subdir), 0755); err != nil {
			return fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}

	// Convert tracking plans
	if err := c.convertTrackingPlans(yamlDir); err != nil {
		return fmt.Errorf("failed to convert tracking plans: %w", err)
	}

	// Convert events
	if err := c.convertEvents(yamlDir); err != nil {
		return fmt.Errorf("failed to convert events: %w", err)
	}

	// Convert properties
	if err := c.convertProperties(yamlDir); err != nil {
		return fmt.Errorf("failed to convert properties: %w", err)
	}

	// Convert custom types
	if err := c.convertCustomTypes(yamlDir); err != nil {
		return fmt.Errorf("failed to convert custom types: %w", err)
	}

	// Convert categories
	if err := c.convertCategories(yamlDir); err != nil {
		return fmt.Errorf("failed to convert categories: %w", err)
	}

	fmt.Printf("Successfully converted JSON files to YAML format in %s\n", yamlDir)
	return nil
}

func (c *TrackingPlanConverter) convertTrackingPlans(yamlDir string) error {
	inputFile := filepath.Join(c.inputDir, "json", "tracking-plans.json")

	var trackingPlans []catalog.TrackingPlanWithIdentifiers
	if err := c.readJSONFile(inputFile, &trackingPlans); err != nil {
		return err
	}

	// Convert each tracking plan to a separate YAML file
	for _, tp := range trackingPlans {
		// Convert events to rule format
		var rules []map[string]interface{}
		for _, event := range tp.Events {
			// Convert properties to rule format
			var properties []map[string]interface{}
			for _, prop := range event.Properties {
				propData := map[string]interface{}{
					"$ref":     fmt.Sprintf("#/properties/generated_properties/%s", prop.ID),
					"required": prop.Required,
				}
				properties = append(properties, propData)
			}

			rule := map[string]interface{}{
				"type": "event_rule",
				"id":   fmt.Sprintf("%s_rule", event.ID),
				"event": map[string]interface{}{
					"$ref":            fmt.Sprintf("#/events/generated_events/%s", event.ID),
					"allow_unplanned": false,
				},
			}

			if len(properties) > 0 {
				rule["properties"] = properties
			}

			rules = append(rules, rule)
		}

		resource := map[string]interface{}{
			"version": "rudder/v0.1",
			"kind":    "tp",
			"metadata": map[string]interface{}{
				"name": sanitizeFilename(tp.Name),
			},
			"spec": map[string]interface{}{
				"id":           tp.ID,
				"display_name": tp.Name,
				"description":  tp.Description,
				"rules":        rules,
			},
		}

		filename := fmt.Sprintf("%s.yaml", sanitizeFilename(tp.Name))
		outputFile := filepath.Join(yamlDir, "tracking-plans", filename)
		if err := c.writeYAMLFile(outputFile, resource); err != nil {
			return fmt.Errorf("failed to write tracking plan %s: %w", tp.Name, err)
		}
	}

	return nil
}

func (c *TrackingPlanConverter) convertEvents(yamlDir string) error {
	inputFile := filepath.Join(c.inputDir, "json", "events.json")

	var events []catalog.Event
	if err := c.readJSONFile(inputFile, &events); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	// Group events by a logical grouping (for now, use a single file)
	var eventSpecs []map[string]interface{}
	for _, event := range events {
		eventSpec := map[string]interface{}{
			"id":          event.ID,
			"name":        event.Name,
			"event_type":  event.EventType,
			"description": event.Description,
		}

		if event.CategoryId != nil {
			eventSpec["category"] = fmt.Sprintf("#/categories/generated_categories/%s", *event.CategoryId)
		}

		eventSpecs = append(eventSpecs, eventSpec)
	}

	resource := map[string]interface{}{
		"version": "rudder/v0.1",
		"kind":    "events",
		"metadata": map[string]interface{}{
			"name": "generated_events",
		},
		"spec": map[string]interface{}{
			"events": eventSpecs,
		},
	}

	outputFile := filepath.Join(yamlDir, "events", "generated_events.yaml")
	if err := c.writeYAMLFile(outputFile, resource); err != nil {
		return fmt.Errorf("failed to write events: %w", err)
	}

	return nil
}

func (c *TrackingPlanConverter) convertProperties(yamlDir string) error {
	inputFile := filepath.Join(c.inputDir, "json", "properties.json")

	var properties []catalog.Property
	if err := c.readJSONFile(inputFile, &properties); err != nil {
		return err
	}

	if len(properties) == 0 {
		return nil
	}

	// Group properties by a logical grouping (for now, use a single file)
	var propertySpecs []map[string]interface{}
	for _, prop := range properties {
		propertySpec := map[string]interface{}{
			"id":          prop.ID,
			"name":        prop.Name,
			"type":        prop.Type,
			"description": prop.Description,
		}

		propertySpecs = append(propertySpecs, propertySpec)
	}

	resource := map[string]interface{}{
		"version": "rudder/v0.1",
		"kind":    "properties",
		"metadata": map[string]interface{}{
			"name": "generated_properties",
		},
		"spec": map[string]interface{}{
			"properties": propertySpecs,
		},
	}

	outputFile := filepath.Join(yamlDir, "properties", "generated_properties.yaml")
	if err := c.writeYAMLFile(outputFile, resource); err != nil {
		return fmt.Errorf("failed to write properties: %w", err)
	}

	return nil
}

func (c *TrackingPlanConverter) convertCustomTypes(yamlDir string) error {
	inputFile := filepath.Join(c.inputDir, "json", "custom-types.json")

	var customTypes []catalog.CustomType
	if err := c.readJSONFile(inputFile, &customTypes); err != nil {
		return err
	}

	if len(customTypes) == 0 {
		return nil
	}

	// Group custom types by a logical grouping (for now, use a single file)
	var typeSpecs []map[string]interface{}
	for _, ct := range customTypes {
		typeSpec := map[string]interface{}{
			"id":          ct.ID,
			"name":        ct.Name,
			"type":        ct.Type,
			"description": ct.Description,
		}

		// Add properties if they exist
		if len(ct.Properties) > 0 {
			var properties []map[string]interface{}
			for _, prop := range ct.Properties {
				propData := map[string]interface{}{
					"$ref": fmt.Sprintf("#/properties/generated_properties/%s", prop.ID),
				}
				properties = append(properties, propData)
			}
			typeSpec["properties"] = properties
		}

		typeSpecs = append(typeSpecs, typeSpec)
	}

	resource := map[string]interface{}{
		"version": "rudder/v0.1",
		"kind":    "custom-types",
		"metadata": map[string]interface{}{
			"name": "generated_custom_types",
		},
		"spec": map[string]interface{}{
			"types": typeSpecs,
		},
	}

	outputFile := filepath.Join(yamlDir, "custom-types", "generated_custom_types.yaml")
	if err := c.writeYAMLFile(outputFile, resource); err != nil {
		return fmt.Errorf("failed to write custom types: %w", err)
	}

	return nil
}

func (c *TrackingPlanConverter) convertCategories(yamlDir string) error {
	inputFile := filepath.Join(c.inputDir, "json", "categories.json")

	var categories []catalog.Category
	if err := c.readJSONFile(inputFile, &categories); err != nil {
		return err
	}

	if len(categories) == 0 {
		return nil
	}

	// Group categories by a logical grouping (for now, use a single file)
	var categorySpecs []map[string]interface{}
	for _, cat := range categories {
		categorySpec := map[string]interface{}{
			"id":          cat.ID,
			"name":        cat.Name,
			"description": fmt.Sprintf("%s-related events", cat.Name),
		}

		categorySpecs = append(categorySpecs, categorySpec)
	}

	resource := map[string]interface{}{
		"version": "rudder/v0.1",
		"kind":    "categories",
		"metadata": map[string]interface{}{
			"name": "generated_categories",
		},
		"spec": map[string]interface{}{
			"categories": categorySpecs,
		},
	}

	outputFile := filepath.Join(yamlDir, "categories", "generated_categories.yaml")
	if err := c.writeYAMLFile(outputFile, resource); err != nil {
		return fmt.Errorf("failed to write categories: %w", err)
	}

	return nil
}

func (c *TrackingPlanConverter) readJSONFile(filePath string, target interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, but that's okay - return empty data
			return nil
		}
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	if len(data) == 0 {
		// Empty file, return nil
		return nil
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", filePath, err)
	}

	return nil
}

func (c *TrackingPlanConverter) writeYAMLFile(filePath string, data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}

	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	// Replace spaces and special characters with underscores
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// NewCmdTPConvert creates a new cobra command for converting JSON to YAML
func NewCmdTPConvert() *cobra.Command {
	var inputDir string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert JSON files to YAML format",
		Long: heredoc.Doc(`
			Convert JSON files from the download command to YAML format.

			This command reads JSON files from the input directory and converts
			them to rudder-cli resource YAML format, organizing them into separate
			directories by resource type.
		`),
		Example: heredoc.Doc(`
			# Convert using default directories
			$ rudder-cli tp convert

			# Convert with custom directories
			$ rudder-cli tp convert --input-dir ./my-data/json --output-dir ./my-yaml
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("tp convert", err, []telemetry.KV{
					{K: "input_dir", V: inputDir},
					{K: "output_dir", V: outputDir},
				}...)
			}()

			// Set defaults
			if inputDir == "" {
				inputDir = "./tracking-plans"
			}
			if outputDir == "" {
				outputDir = "./tracking-plans"
			}

			// Validate input directory exists
			if _, err := os.Stat(filepath.Join(inputDir, "json")); os.IsNotExist(err) {
				return fmt.Errorf("input directory %s/json does not exist. Run 'rudder-cli tp download' first", inputDir)
			}

			converter := NewTrackingPlanConverter(inputDir, outputDir)
			return converter.Convert()
		},
	}

	cmd.Flags().StringVar(&inputDir, "input-dir", "./tracking-plans", "Input directory containing JSON files")
	cmd.Flags().StringVar(&outputDir, "output-dir", "./tracking-plans", "Output directory for YAML files")

	return cmd
}