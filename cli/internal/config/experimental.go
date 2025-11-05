package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

// ExperimentalConfig defines all available experimental flags
// All flags default to false for safety - explicit opt-in required
type ExperimentalConfig struct {
	// ConcurrentSyncs enables concurrent sync operations when applying changes
	ConcurrentSyncs bool `mapstructure:"concurrentSyncs"`

	// RudderTyper enables the new typer engine for type checking and validation
	RudderTyper bool `mapstructure:"rudderTyper"`
}

// getAvailableExperimentalFlags returns information about all available experimental flags
func getAvailableExperimentalFlags() []string {
	cfg := GetConfig()
	experimental := cfg.ExperimentalFlags
	expType := reflect.TypeOf(experimental)

	var flags []string
	for i := 0; i < expType.NumField(); i++ {
		field := expType.Field(i)

		// Get the mapstructure tag as the flag name
		flagName := field.Tag.Get("mapstructure")
		if flagName != "" {
			flags = append(flags, flagName)
		}
	}

	return flags
}

// IsValidExperimentalFlag checks if a flag name is valid
func IsValidExperimentalFlag(flagName string) bool {
	flags := getAvailableExperimentalFlags()
	for _, flag := range flags {
		if flag == flagName {
			return true
		}
	}
	return false
}

// GetEnvironmentVariableName returns the environment variable name for a given experimental flag
func GetEnvironmentVariableName(flagName string) string {
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(flagName, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")

	return fmt.Sprintf("RUDDERSTACK_X_%s", strings.ToUpper(snake))
}

// BindExperimentalFlags automatically binds environment variables for all experimental flags
func BindExperimentalFlags() {
	flags := getAvailableExperimentalFlags()

	for _, flag := range flags {
		// Set default value to false (all experimental flags are disabled by default)
		viperKey := fmt.Sprintf("flags.%s", flag)
		viper.SetDefault(viperKey, false)

		// Generate environment variable name
		envVarName := GetEnvironmentVariableName(flag)

		// Bind environment variable
		viper.BindEnv(viperKey, envVarName)
	}
}
