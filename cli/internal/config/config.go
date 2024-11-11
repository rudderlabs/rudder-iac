package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	cobra.CheckErr(err)

	return fmt.Sprintf("%s/.rudder", homeDir)
}

func DefaultConfigFile() string {
	return filepath.Join(defaultConfigPath(), "config.json")
}

func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
	} else {
		cfgFile = DefaultConfigFile()
	}

	createConfigFileIfNotExists(cfgFile)
	viper.SetConfigFile(cfgFile)

	// bind environment variables
	viper.BindEnv("auth.accessToken", "RUDDER_ACCESS_TOKEN")

	// set defaults
	viper.SetDefault("debug", false)
	viper.SetDefault("verbose", false)

	// load configuration
	_ = viper.ReadInConfig()
}

func createConfigFileIfNotExists(cfgFile string) {
	configPath := filepath.Dir(cfgFile)
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		fmt.Printf("Config file '%s' not found. Creating default configuration...\n", cfgFile)

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
