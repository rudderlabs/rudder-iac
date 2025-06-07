package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
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
	Debug        bool   `mapstructure:"debug"`
	Experimental bool   `mapstructure:"experimental"`
	Verbose      bool   `mapstructure:"verbose"`
	APIURL       string `mapstructure:"apiURL"`
	Auth         struct {
		AccessToken string `mapstructure:"accessToken"`
	} `mapstructure:"auth"`
	Telemetry struct {
		Disabled     bool   `mapstructure:"disabled"`
		AnonymousID  string `mapstructure:"anonymousId"`
		WriteKey     string `mapstructure:"writeKey"`
		DataplaneURL string `mapstructure:"dataplaneURL"`
	} `mapstructure:"telemetry"`
}

func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// In CI environments or when home directory is not available,
		// fall back to current directory or temp directory
		if tmpDir := os.TempDir(); tmpDir != "" {
			return filepath.Join(tmpDir, ".rudder")
		}
		// Final fallback to current directory
		return ".rudder"
	}

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
	if err != nil {
		// In CI/test environments, if we can't create config file,
		// continue with in-memory config only
		log.Debug("could not create config file, using in-memory config", "error", err)
	}

	viper.SetConfigFile(cfgFile)

	// set defaults
	viper.SetDefault("debug", false)
	viper.SetDefault("experimental", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("apiURL", client.BASE_URL_V2)
	viper.SetDefault("telemetry.disabled", false)
	viper.SetDefault("telemetry.writeKey", TelemetryWriteKey)
	viper.SetDefault("telemetry.dataplaneURL", TelemetryDataplaneURL)

	viper.BindEnv("auth.accessToken", "RUDDERSTACK_ACCESS_TOKEN")
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")
	viper.BindEnv("experimental", "RUDDERSTACK_CLI_EXPERIMENTAL")
	viper.BindEnv("telemetry.writeKey", "RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY")
	viper.BindEnv("telemetry.dataplaneURL", "RUDDERSTACK_CLI_TELEMETRY_DATAPLANE_URL")
	viper.BindEnv("telemetry.disabled", "RUDDERSTACK_CLI_TELEMETRY_DISABLED")

	// load configuration - this is optional in case file doesn't exist
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

func updateConfig(f func(data []byte) ([]byte, error)) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// If no config file is being used, skip the update
		log.Debug("no config file in use, skipping config update")
		return
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Debug("could not read config file", "error", err)
		return
	}

	newData, err := f(data)
	if err != nil {
		log.Debug("could not update config data", "error", err)
		return
	}

	formattedData := pretty.Pretty(newData)

	err = os.WriteFile(configFile, formattedData, 0644)
	if err != nil {
		log.Debug("could not write config file", "error", err)
		return
	}

	_ = viper.ReadInConfig()
}

func GetConfig() Config {
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		// In case of error, return default config
		log.Debug("could not unmarshal config, using defaults", "error", err)
		return Config{
			Debug:        false,
			Experimental: false,
			Verbose:      false,
			APIURL:       client.BASE_URL_V2,
		}
	}

	return config
}

func GetConfigDir() string {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return defaultConfigPath()
	}
	return filepath.Dir(configFile)
}
