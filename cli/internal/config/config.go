package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

var log = logger.New("config")

func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	cobra.CheckErr(err)

	return fmt.Sprintf("%s/.rudder", homeDir)
}

func DefaultConfigFile() string {
	return filepath.Join(defaultConfigPath(), "config.json")
}

func InitConfig(cfgFile string) {
	log.Info("initializing the configuration", "location", cfgFile)

	if cfgFile != "" {
		// Use config file from the flag.
	} else {
		cfgFile = DefaultConfigFile()
	}

	createConfigFileIfNotExists(cfgFile)
	viper.SetConfigFile(cfgFile)

	// set defaults
	viper.SetDefault("debug", false)
	viper.SetDefault("verbose", false)

	// load configuration
	_ = viper.ReadInConfig()
	// Once the config is read, bind the env's for the provider to make use of
	os.Setenv("R_ACCESS_TOKEN", viper.GetString("auth.accessToken"))
	// In case we have overriden the configbackend URL directly into the config
	if viper.IsSet("auth.cbURL") {
		os.Setenv("R_CONFIG_BACKEND", viper.GetString("auth.cbURL"))
	} else {
		os.Setenv("R_CONFIG_BACKEND", viper.GetString("https://api.rudderstack.com"))
	}
}

func createConfigFileIfNotExists(cfgFile string) {
	configPath := filepath.Dir(cfgFile)

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		log.Info("Config file not found, creating default configuration", "location", cfgFile)

		if err := os.MkdirAll(configPath, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			os.Exit(1)
		}

		file, err := os.Create(cfgFile)
		cobra.CheckErr(err)
		defer file.Close()
	}
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

// func SetConfigBackendURL(url string) {
// 	configFile := viper.ConfigFileUsed()
// 	data, err := os.ReadFile(configFile)
// 	cobra.CheckErr(err)

// 	newData, err := sjson.SetBytes(data, "auth.cbURL", url)
// 	cobra.CheckErr(err)

// 	formattedData := pretty.Pretty(newData)

// 	err = os.WriteFile(configFile, formattedData, 0644)
// 	cobra.CheckErr(err)
// }
