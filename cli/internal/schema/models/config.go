package models

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// EventTypeConfig represents the configuration for event type to JSONPath mappings
type EventTypeConfig struct {
	EventMappings map[string]string `yaml:"event_mappings"`
}

// DefaultEventTypeConfig returns the default configuration where all event types map to $.properties
func DefaultEventTypeConfig() *EventTypeConfig {
	return &EventTypeConfig{
		EventMappings: map[string]string{
			"identify": "$.properties",
			"page":     "$.properties",
			"screen":   "$.properties",
			"group":    "$.properties",
			"alias":    "$.properties",
			// track is always $.properties and handled separately
		},
	}
}

// LoadEventTypeConfig loads configuration from a YAML file
func LoadEventTypeConfig(configPath string) (*EventTypeConfig, error) {
	if configPath == "" {
		return DefaultEventTypeConfig(), nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %s does not exist", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config EventTypeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config from %s: %w", configPath, err)
	}

	// Validate mappings
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config in %s: %w", configPath, err)
	}

	return &config, nil
}

// GetJSONPathForEventType returns the JSONPath for a given event type
func (c *EventTypeConfig) GetJSONPathForEventType(eventType string) string {
	// Track always uses $.properties regardless of config
	if eventType == "track" {
		return "$.properties"
	}

	// Check if event type has a specific mapping
	if path, exists := c.EventMappings[eventType]; exists {
		return path
	}

	// Default to $.properties for unknown event types
	return "$.properties"
}

// Validate validates the configuration
func (c *EventTypeConfig) Validate() error {
	validPaths := map[string]bool{
		"$.traits":         true,
		"$.context.traits": true,
		"$.properties":     true,
	}

	for eventType, jsonPath := range c.EventMappings {
		if !validPaths[jsonPath] {
			return fmt.Errorf("invalid JSONPath '%s' for event type '%s'. Valid paths are: $.traits, $.context.traits, $.properties", jsonPath, eventType)
		}
	}

	return nil
}

// HasCustomMappings returns true if the config has any non-default mappings
func (c *EventTypeConfig) HasCustomMappings() bool {
	defaultConfig := DefaultEventTypeConfig()

	// Check if any mapping is different from default
	for eventType, path := range c.EventMappings {
		if defaultPath := defaultConfig.GetJSONPathForEventType(eventType); path != defaultPath {
			return true
		}
	}

	return false
}
