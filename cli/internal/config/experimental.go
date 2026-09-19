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
	// EventRuleIncludes enables including event rules from other tracking plans
	EventRuleIncludes bool `mapstructure:"eventRuleIncludes"`
	// ImportMerge enables import-manifest.yaml generation during `import workspace`
	// and treats the import-manifest kind as a recognized spec kind during
	// validation. This feature `import workspace --merge` links matching
	// remote resources to existing local project resources instead of
	// generating duplicate specs
	ImportMerge bool `mapstructure:"importMerge"`
	// UnverifiedDestinations enables registration of destination definitions
	// that still need the unverified gate.
	UnverifiedDestinations bool `mapstructure:"unverifiedDestinations"`
	// RetlConnectionSupport enables the rETL connection kind in the rETL
	// provider: its spec kind, resource type, lifecycle and import matcher.
	//
	// Turning it on also puts retl-connections@rudder/v1 in scope for rule-doc
	// generation, where no authored fragment covers it yet, so `make
	// gen-rule-docs` and TestGenerateRuleCatalog_CompleteAndDriftFree fail until
	// DEX-829 lands the fragments. CLI validation is unaffected either way.
	RetlConnectionSupport bool `mapstructure:"retlConnectionSupport"`

	// RETLTableSupport registers the retl-source-table spec kind with the RETL
	// provider.
	RETLTableSupport bool `mapstructure:"retlTableSupport"`
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
