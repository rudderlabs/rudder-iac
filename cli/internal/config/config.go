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

const (
	AccessTokenEnvVar = "RUDDERSTACK_ACCESS_TOKEN"
	DefaultAppURL     = "https://app.rudderstack.com"

	// TokenStoreKeychain marks a browser login: the token lives in the OS
	// keychain and the config file holds none.
	TokenStoreKeychain = "keychain"
)

type Config = struct {
	Debug   bool   `mapstructure:"debug"`
	Verbose bool   `mapstructure:"verbose"`
	APIURL  string `mapstructure:"apiURL"`
	AppURL  string `mapstructure:"appURL"`
	Auth    struct {
		AccessToken   string `mapstructure:"accessToken"`
		TokenStore    string `mapstructure:"tokenStore"`
		WorkspaceID   string `mapstructure:"workspaceID"`
		WorkspaceName string `mapstructure:"workspaceName"`
	} `mapstructure:"auth"`
	Telemetry struct {
		Disabled     bool   `mapstructure:"disabled"`
		AnonymousID  string `mapstructure:"anonymousId"`
		WriteKey     string `mapstructure:"writeKey"`
		DataplaneURL string `mapstructure:"dataplaneURL"`
	} `mapstructure:"telemetry"`
	ExperimentalFlags ExperimentalConfig `mapstructure:"flags"`
	Concurrency       struct {
		Syncer            int `mapstructure:"syncer"`
		CatalogClient     int `mapstructure:"catalogClient"`
		CompositeProvider int `mapstructure:"compositeProvider"`
		CatalogProvider   int `mapstructure:"catalogProvider"`
		DataGraph         int `mapstructure:"dataGraph"`
	}
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
	viper.SetDefault("appURL", DefaultAppURL)
	viper.SetDefault("telemetry.disabled", false)
	viper.SetDefault("telemetry.writeKey", TelemetryWriteKey)
	viper.SetDefault("telemetry.dataplaneURL", TelemetryDataplaneURL)
	viper.SetDefault("concurrency.syncer", 30)
	viper.SetDefault("concurrency.catalogClient", 10)
	viper.SetDefault("concurrency.compositeProvider", 2)
	viper.SetDefault("concurrency.catalogProvider", 4)
	viper.SetDefault("concurrency.dataGraph", 4)

	viper.BindEnv("auth.accessToken", AccessTokenEnvVar)
	viper.BindEnv("apiURL", "RUDDERSTACK_API_URL")
	viper.BindEnv("appURL", "RUDDERSTACK_APP_URL")
	viper.BindEnv("telemetry.writeKey", "RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY")
	viper.BindEnv("telemetry.dataplaneURL", "RUDDERSTACK_CLI_TELEMETRY_DATAPLANE_URL")
	viper.BindEnv("telemetry.disabled", "RUDDERSTACK_CLI_TELEMETRY_DISABLED")
	viper.BindEnv("debug", "RUDDERSTACK_CLI_DEBUG")
	viper.BindEnv("experimental", ExperimentalEnvVar)
	viper.BindEnv("concurrency.syncer", "RUDDERSTACK_CLI_CONCURRENCY_SYNCER")
	viper.BindEnv("concurrency.catalogClient", "RUDDERSTACK_CLI_CONCURRENCY_CATALOG_CLIENT")
	viper.BindEnv("concurrency.compositeProvider", "RUDDERSTACK_CLI_CONCURRENCY_COMPOSITE_PROVIDER")
	viper.BindEnv("concurrency.catalogProvider", "RUDDERSTACK_CLI_CONCURRENCY_CATALOG_PROVIDER")

	// Automatically bind environment variables for all experimental flags
	BindExperimentalFlags()

	// load configuration
	_ = viper.ReadInConfig()
}

func createConfigFileIfNotExists(cfgFile string) error {
	configPath := filepath.Dir(cfgFile)

	info, err := os.Stat(cfgFile)
	if err == nil {
		// Older releases created the file 0644 while it held the access token,
		// and WriteFile keeps the mode of an existing file.
		if info.Mode().Perm()&0o077 != 0 {
			if err := os.Chmod(cfgFile, 0o600); err != nil {
				return fmt.Errorf("tightening config file mode: %w", err)
			}
		}
		return nil
	}

	if os.IsNotExist(err) {
		log.Info("Config file not found, creating default configuration", "location", cfgFile)

		if err := os.MkdirAll(configPath, 0o700); err != nil {
			return fmt.Errorf("error creating config directory: %v", err)
		}

		file, err := os.OpenFile(cfgFile, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("error creating config file: %v", err)
		}
		defer file.Close()
	}

	return nil
}

// SetKeychainLogin records a browser login. The token itself goes to the
// keychain, so any token an earlier login left in the file is removed.
func SetKeychainLogin(workspaceID, workspaceName string) {
	updateConfig(func(data []byte) ([]byte, error) {
		data, err := sjson.DeleteBytes(data, "auth.accessToken")
		if err != nil {
			return nil, err
		}
		return sjson.SetBytes(data, "auth", map[string]string{
			"tokenStore":    TokenStoreKeychain,
			"workspaceID":   workspaceID,
			"workspaceName": workspaceName,
		})
	})
}

// SetFileToken records a token login (`auth login --with-token`). It drops the
// keychain marker so the new token is the one in use.
func SetFileToken(accessToken string) {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.SetBytes(data, "auth", map[string]string{"accessToken": accessToken})
	})
}

func ClearAuth() {
	updateConfig(func(data []byte) ([]byte, error) {
		return sjson.DeleteBytes(data, "auth")
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

	err = os.WriteFile(configFile, formattedData, 0o600)
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

	// While concurrent syncs were gated, a sub-1 concurrency.syncer was inert
	// for anyone who had not enabled the flag: WithConcurrency was never applied
	// and the syncer fell back to 1. Reading the value unconditionally would
	// fail their next apply, so clamp instead of erroring. Users who did enable
	// the flag were already refused by WithConcurrency, and clamping is the
	// quieter of the two — chosen so that no existing config breaks on upgrade.
	if config.Concurrency.Syncer < 1 {
		config.Concurrency.Syncer = 1
	}

	return config
}

func GetConfigDir() string {
	return filepath.Dir(viper.ConfigFileUsed())
}
