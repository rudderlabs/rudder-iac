package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

var (
	log                   = logger.New("config")
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
		Disabled bool   `mapstructure:"disabled"`
		UserID   string `mapstructure:"userId"`
	} `mapstructure:"telemetry"`
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
	viper.SetDefault("apiURL", client.BASE_URL_V2)
	viper.SetDefault("telemetry.writeKey", "")
	viper.SetDefault("telemetry.dataplaneURL", "")

	viper.BindEnv("auth.accessToken", "RUDDERSTACK_ACCESS_TOKEN")
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")
	viper.BindEnv("telemetry.writeKey", "RUDDERSTACK_TELEMETRY_WRITE_KEY")
	viper.BindEnv("telemetry.dataplaneURL", "RUDDERSTACK_TELEMETRY_DATAPLANE_URL")

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
	configFile := viper.ConfigFileUsed()
	data, err := os.ReadFile(configFile)
	cobra.CheckErr(err)

	newData, err := sjson.SetBytes(data, "auth.accessToken", accessToken)
	cobra.CheckErr(err)

	formattedData := pretty.Pretty(newData)

	err = os.WriteFile(configFile, formattedData, 0644)
	cobra.CheckErr(err)
}

func SetTelemetryDisabled(disabled bool) {
	configFile := viper.ConfigFileUsed()
	data, err := os.ReadFile(configFile)
	cobra.CheckErr(err)

	newData, err := sjson.SetBytes(data, "telemetry.disabled", disabled)
	cobra.CheckErr(err)

	formattedData := pretty.Pretty(newData)

	err = os.WriteFile(configFile, formattedData, 0644)
	cobra.CheckErr(err)
}

func SetTelemetryUserID(userID string) {
	configFile := viper.ConfigFileUsed()
	data, err := os.ReadFile(configFile)
	cobra.CheckErr(err)

	newData, err := sjson.SetBytes(data, "telemetry.userId", userID)
	cobra.CheckErr(err)

	formattedData := pretty.Pretty(newData)

	err = os.WriteFile(configFile, formattedData, 0644)
	cobra.CheckErr(err)
}

func GetTelemetryUserID() string {
	return viper.GetString("telemetry.userId")
}

func GetTelemetryWriteKey() (writeKey string) {
	// Always prefer the value overriden by customer using env var
	writeKey = viper.GetString("telemetry.writeKey")
	if writeKey == "" {
		// fallback to the default value
		// which might be provided through ldflags when
		// building the binary
		writeKey = TelemetryWriteKey
	}
	return
}

func GetTelemetryDataplaneURL() (dataplaneURL string) {
	// Always prefer the value overriden by customer using env var
	dataplaneURL = viper.GetString("telemetry.dataplaneURL")
	if dataplaneURL == "" {
		// fallback to the default value
		// which might be provided through ldflags when
		// building the binary
		dataplaneURL = TelemetryDataplaneURL
	}
	return
}

func GetConfig() Config {
	var config Config
	err := viper.Unmarshal(&config)
	cobra.CheckErr(err)

	return config
}

func GetConfigDir() string {
	return filepath.Dir(viper.ConfigFileUsed())
}
