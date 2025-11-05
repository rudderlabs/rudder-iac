package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

var (
	log = logger.New("config")
	// Below variables are set during build time using ldflags
	// and locations are defined in the Makefile. Any changes
	// to the below variables should be reflected in the Makefile
	TelemetryWriteKey     = ""
	TelemetryDataplaneURL = ""
)

type Config = struct {
	Debug   bool   `mapstructure:"debug"`
	Verbose bool   `mapstructure:"verbose"`
	APIURL  string `mapstructure:"apiURL"`
	Auth    struct {
		AccessToken string `mapstructure:"accessToken"`
	} `mapstructure:"auth"`
	Telemetry struct {
		Disabled     bool   `mapstructure:"disabled"`
		AnonymousID  string `mapstructure:"anonymousId"`
		WriteKey     string `mapstructure:"writeKey"`
		DataplaneURL string `mapstructure:"dataplaneURL"`
	} `mapstructure:"telemetry"`
	ExperimentalFlags ExperimentalConfig `mapstructure:"flags"`
}

func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	cobra.CheckErr(err)

	return fmt.Sprintf("%s/.rudder", homeDir)
}

func DefaultConfigFile() string {
	return filepath.Join(defaultConfigPath(), "config.json")
}

func InitConfig(cfgFile string) {
	log.Debug("initializing the configuration", "location", cfgFile)

	if cfgFile != "" {
		// Use config file from the flag.
	} else {
		cfgFile = DefaultConfigFile()
	}

	err := createConfigFileIfNotExists(cfgFile)
	cobra.CheckErr(err)

	viper.SetConfigFile(cfgFile)

	// set defaults
	viper.SetDefault("debug", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("apiURL", client.BASE_URL)
	viper.SetDefault("telemetry.disabled", false)
	viper.SetDefault("telemetry.writeKey", TelemetryWriteKey)
	viper.SetDefault("telemetry.dataplaneURL", TelemetryDataplaneURL)

	viper.BindEnv("auth.accessToken", "RUDDERSTACK_ACCESS_TOKEN")
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")
	viper.BindEnv("telemetry.writeKey", "RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY")
	viper.BindEnv("telemetry.dataplaneURL", "RUDDERSTACK_CLI_TELEMETRY_DATAPLANE_URL")
	viper.BindEnv("telemetry.disabled", "RUDDERSTACK_CLI_TELEMETRY_DISABLED")
	viper.BindEnv("debug", "RUDDERSTACK_CLI_DEBUG")

	// Automatically bind environment variables for all experimental flags
	BindExperimentalFlags()

	// load configuration
	_ = viper.ReadInConfig()
}

func createConfigFileIfNotExists(cfgFile string) error {
	configPath := filepath.Dir(cfgFile)

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		log.Info("Config file not found, creating default configuration", "location", cfgFile)

		if err := os.MkdirAll(configPath, 0755); err != nil {
			return fmt.Errorf("error creating config directory: %v", err)
		}

		file, err := os.Create(cfgFile)
		if err != nil {
			return fmt.Errorf("error creating config file: %v", err)
		}
		defer file.Close()
	}

	return nil
}

func SetAccessToken(accessToken string) {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.SetBytes(data, "auth.accessToken", accessToken)
	})
}

func SetTelemetryDisabled(disabled bool) {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.SetBytes(data, "telemetry.disabled", disabled)
	})
}

func SetTelemetryAnonymousID(anonymousID string) {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.SetBytes(data, "telemetry.anonymousID", anonymousID)
	})
}

func SetExperimentalFlag(flagName string, enabled bool) {
	updateConfig(func(data []byte) ([]byte, error) {
		// Validate flag name exists
		if !IsValidExperimentalFlag(flagName) {
			return nil, fmt.Errorf("unknown experimental flag: %s", flagName)
		}

		return sjson.SetBytes(data, fmt.Sprintf("flags.%s", flagName), enabled)
	})
}

func ResetExperimentalFlags() {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.DeleteBytes(data, "flags")
	})
}

func updateConfig(f func(data []byte) ([]byte, error)) {
	configFile := viper.ConfigFileUsed()
	data, err := os.ReadFile(configFile)
	cobra.CheckErr(err)

	newData, err := f(data)
	cobra.CheckErr(err)

	formattedData := pretty.Pretty(newData)

	err = os.WriteFile(configFile, formattedData, 0644)
	cobra.CheckErr(err)

	_ = viper.ReadInConfig()
}

func GetConfig() Config {
	var config Config
	err := viper.Unmarshal(&config)
	cobra.CheckErr(err)

	if !viper.GetBool("experimental") {
		config.ExperimentalFlags = ExperimentalConfig{}
	}

	return config
}

func GetConfigDir() string {
	return filepath.Dir(viper.ConfigFileUsed())
}
