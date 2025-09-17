package convert_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConvertTrackingPlans(t *testing.T) {
	// Create temporary directories for test
	tmpDir, err := os.MkdirTemp("", "tp-convert-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Setup input directory with JSON data
	inputDir := tmpDir
	jsonDir := filepath.Join(inputDir, "json")
	err = os.MkdirAll(jsonDir, 0755)
	require.NoError(t, err)

	// Create test data
	now := time.Now()
	trackingPlans := []catalog.TrackingPlanWithIdentifiers{
		{
			ID:           "tp-1",
			Name:         "E-commerce Plan",
			Description:  strPtr("Main tracking plan"),
			CreationType: "manual",
			Version:      1,
			WorkspaceID:  "ws-1",
			CreatedAt:    now,
			UpdatedAt:    now,
			Events: []catalog.TrackingPlanEventPropertyIdentifiers{
				{
					ID:          "event-1",
					Name:        "Page View",
					Description: "User viewed a page",
					EventType:   "track",
					WorkspaceId: "ws-1",
					CreatedAt:   now,
					UpdatedAt:   now,
					Properties: []*catalog.TrackingPlanEventProperty{
						{
							ID:       "prop-1",
							Name:     "page_url",
							Required: true,
						},
					},
				},
			},
		},
	}

	events := []catalog.Event{
		{
			ID:          "event-1",
			Name:        "Page View",
			Description: "User viewed a page",
			EventType:   "track",
			CategoryId:  strPtr("cat-1"),
			WorkspaceId: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	properties := []catalog.Property{
		{
			ID:          "prop-1",
			Name:        "page_url",
			Description: "URL of the page",
			Type:        "string",
			WorkspaceId: "ws-1",
			Config:      map[string]interface{}{"format": "url"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	customTypes := []catalog.CustomType{
		{
			ID:          "ct-1",
			Name:        "Custom String",
			Description: "Custom string type",
			Type:        "string",
			DataType:    "primitive",
			Version:     1,
			WorkspaceId: "ws-1",
			Config:      map[string]interface{}{"maxLength": 255},
			Rules:       map[string]interface{}{"pattern": "^[a-zA-Z0-9]*$"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	categories := []catalog.Category{
		{
			ID:          "cat-1",
			Name:        "User Actions",
			WorkspaceID: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	// Write JSON files
	writeJSONFile(t, filepath.Join(jsonDir, "tracking-plans.json"), trackingPlans)
	writeJSONFile(t, filepath.Join(jsonDir, "events.json"), events)
	writeJSONFile(t, filepath.Join(jsonDir, "properties.json"), properties)
	writeJSONFile(t, filepath.Join(jsonDir, "custom-types.json"), customTypes)
	writeJSONFile(t, filepath.Join(jsonDir, "categories.json"), categories)

	// Create converter and run conversion
	outputDir := tmpDir
	converter := convert.NewTrackingPlanConverter(inputDir, outputDir)
	err = converter.Convert()
	require.NoError(t, err)

	// Verify YAML files were created
	yamlDir := filepath.Join(outputDir, "yaml")

	// Check tracking plan YAML
	tpYamlFile := filepath.Join(yamlDir, "tracking-plans", "E-commerce_Plan.yaml")
	assert.FileExists(t, tpYamlFile)

	var tpResource map[string]interface{}
	readYAMLFile(t, tpYamlFile, &tpResource)
	assert.Equal(t, "rudder/v0.1", tpResource["version"])
	assert.Equal(t, "tp", tpResource["kind"])
	assert.Equal(t, "E-commerce_Plan", tpResource["metadata"].(map[string]interface{})["name"])

	spec := tpResource["spec"].(map[string]interface{})
	assert.Equal(t, "tp-1", spec["id"])
	assert.Equal(t, "E-commerce Plan", spec["display_name"])
	assert.Equal(t, "Main tracking plan", spec["description"])

	// Verify rules are included in the tracking plan
	rules := spec["rules"].([]interface{})
	assert.Len(t, rules, 1)
	rule := rules[0].(map[string]interface{})
	assert.Equal(t, "event_rule", rule["type"])
	assert.Equal(t, "event-1_rule", rule["id"])

	// Verify event reference
	event := rule["event"].(map[string]interface{})
	assert.Equal(t, "#/events/event-1", event["$ref"])
	assert.Equal(t, false, event["allow_unplanned"])

	// Verify rule properties are included
	ruleProperties := rule["properties"].([]interface{})
	assert.Len(t, ruleProperties, 1)
	ruleProperty := ruleProperties[0].(map[string]interface{})
	assert.Equal(t, "#/properties/prop-1", ruleProperty["$ref"])
	assert.Equal(t, true, ruleProperty["required"])

	// Check event YAML (now grouped)
	eventYamlFile := filepath.Join(yamlDir, "events", "generated_events.yaml")
	assert.FileExists(t, eventYamlFile)

	var eventResource map[string]interface{}
	readYAMLFile(t, eventYamlFile, &eventResource)
	assert.Equal(t, "rudder/v0.1", eventResource["version"])
	assert.Equal(t, "events", eventResource["kind"])

	eventSpec := eventResource["spec"].(map[string]interface{})
	eventsArray := eventSpec["events"].([]interface{})
	assert.Len(t, eventsArray, 1)

	eventData := eventsArray[0].(map[string]interface{})
	assert.Equal(t, "event-1", eventData["id"])
	assert.Equal(t, "Page View", eventData["name"])
	assert.Equal(t, "track", eventData["event_type"])

	// Check property YAML (now grouped)
	propYamlFile := filepath.Join(yamlDir, "properties", "generated_properties.yaml")
	assert.FileExists(t, propYamlFile)

	var propResource map[string]interface{}
	readYAMLFile(t, propYamlFile, &propResource)
	assert.Equal(t, "rudder/v0.1", propResource["version"])
	assert.Equal(t, "properties", propResource["kind"])

	propSpec := propResource["spec"].(map[string]interface{})
	propertiesArray := propSpec["properties"].([]interface{})
	assert.Len(t, propertiesArray, 1)

	propData := propertiesArray[0].(map[string]interface{})
	assert.Equal(t, "prop-1", propData["id"])
	assert.Equal(t, "page_url", propData["name"])
	assert.Equal(t, "string", propData["type"])

	// Check custom type YAML (now grouped)
	ctYamlFile := filepath.Join(yamlDir, "custom-types", "generated_custom_types.yaml")
	assert.FileExists(t, ctYamlFile)

	var ctResource map[string]interface{}
	readYAMLFile(t, ctYamlFile, &ctResource)
	assert.Equal(t, "rudder/v0.1", ctResource["version"])
	assert.Equal(t, "custom-types", ctResource["kind"])

	ctSpec := ctResource["spec"].(map[string]interface{})
	typesArray := ctSpec["types"].([]interface{})
	assert.Len(t, typesArray, 1)

	typeData := typesArray[0].(map[string]interface{})
	assert.Equal(t, "ct-1", typeData["id"])
	assert.Equal(t, "Custom String", typeData["name"])
	assert.Equal(t, "string", typeData["type"])

	// Check category YAML (now grouped)
	catYamlFile := filepath.Join(yamlDir, "categories", "generated_categories.yaml")
	assert.FileExists(t, catYamlFile)

	var catResource map[string]interface{}
	readYAMLFile(t, catYamlFile, &catResource)
	assert.Equal(t, "rudder/v0.1", catResource["version"])
	assert.Equal(t, "categories", catResource["kind"])

	catSpec := catResource["spec"].(map[string]interface{})
	categoriesArray := catSpec["categories"].([]interface{})
	assert.Len(t, categoriesArray, 1)

	catData := categoriesArray[0].(map[string]interface{})
	assert.Equal(t, "cat-1", catData["id"])
	assert.Equal(t, "User Actions", catData["name"])
}

func TestConvertEmptyFiles(t *testing.T) {
	// Create temporary directories for test
	tmpDir, err := os.MkdirTemp("", "tp-convert-empty-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Setup input directory with empty JSON files
	inputDir := tmpDir
	jsonDir := filepath.Join(inputDir, "json")
	err = os.MkdirAll(jsonDir, 0755)
	require.NoError(t, err)

	// Create empty JSON files
	emptyFiles := []string{"tracking-plans.json", "events.json", "properties.json", "custom-types.json", "categories.json"}
	for _, file := range emptyFiles {
		err = os.WriteFile(filepath.Join(jsonDir, file), []byte("[]"), 0644)
		require.NoError(t, err)
	}

	// Create converter and run conversion
	outputDir := tmpDir
	converter := convert.NewTrackingPlanConverter(inputDir, outputDir)
	err = converter.Convert()
	require.NoError(t, err)

	// Verify directories were created even with empty data
	yamlDir := filepath.Join(outputDir, "yaml")
	subdirs := []string{"tracking-plans", "events", "properties", "custom-types", "categories"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(yamlDir, subdir)
		assert.DirExists(t, dirPath)
	}
}

func TestConvertMissingInputDirectory(t *testing.T) {
	// Create temporary directory for output only
	tmpDir, err := os.MkdirTemp("", "tp-convert-missing-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Try to convert with non-existent input directory
	inputDir := filepath.Join(tmpDir, "nonexistent")
	outputDir := tmpDir
	converter := convert.NewTrackingPlanConverter(inputDir, outputDir)

	err = converter.Convert()
	require.NoError(t, err) // Should not error on missing files, just skip them
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Name", "Simple_Name"},
		{"Name-with-dashes", "Name-with-dashes"},
		{"Name_with_underscores", "Name_with_underscores"},
		{"Name with spaces and symbols!", "Name_with_spaces_and_symbols_"},
		{"123Numbers", "123Numbers"},
		{"Mixed-Case_NAME", "Mixed-Case_NAME"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			// We can't directly test sanitizeFilename since it's not exported,
			// but we can test the behavior by checking generated filenames
			tmpDir, err := os.MkdirTemp("", "tp-sanitize-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create a tracking plan with the test name
			jsonDir := filepath.Join(tmpDir, "json")
			err = os.MkdirAll(jsonDir, 0755)
			require.NoError(t, err)

			trackingPlans := []catalog.TrackingPlanWithIdentifiers{
				{
					ID:          "tp-1",
					Name:        test.input,
					Description: strPtr("Test plan"),
					Version:     1,
					WorkspaceID: "ws-1",
				},
			}

			writeJSONFile(t, filepath.Join(jsonDir, "tracking-plans.json"), trackingPlans)

			converter := convert.NewTrackingPlanConverter(tmpDir, tmpDir)
			err = converter.Convert()
			require.NoError(t, err)

			// Check that the expected filename was created
			expectedFile := filepath.Join(tmpDir, "yaml", "tracking-plans", test.expected+".yaml")
			assert.FileExists(t, expectedFile)
		})
	}
}

func writeJSONFile(t *testing.T, filePath string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filePath, jsonData, 0644)
	require.NoError(t, err)
}

func readYAMLFile(t *testing.T, filePath string, target interface{}) {
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	err = yaml.Unmarshal(data, target)
	require.NoError(t, err)
}

func strPtr(s string) *string {
	return &s
}