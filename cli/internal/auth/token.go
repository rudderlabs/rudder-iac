package auth

import (
	"errors"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
)

var log = logger.New("auth")

type tokenSources struct {
	getenv   func(string) string
	keychain TokenStore
	cfg      config.Config
}

// AccessToken returns the token for API calls, or "" when there is none.
func AccessToken() string {
	return resolveAccessToken(tokenSources{
		getenv:   os.Getenv,
		keychain: NewKeychain(),
		cfg:      config.GetConfig(),
	})
}

// resolveAccessToken looks in the environment, then the keychain, then the
// config file, so CI can override a stored login and old logins keep working.
// The keychain is read only after a browser login recorded it, which keeps
// token-only setups and tests away from the OS keychain.
func resolveAccessToken(src tokenSources) string {
	if token := src.getenv(config.AccessTokenEnvVar); token != "" {
		return token
	}

	if src.cfg.Auth.TokenStore == config.TokenStoreKeychain {
		token, err := src.keychain.Get(KeychainAccount(src.cfg.APIURL))
		switch {
		case err == nil && token != "":
			return token
		case err != nil && !errors.Is(err, ErrTokenNotFound):
			log.Warn("reading token from keychain", "error", err)
		}
	}

	// With the env var unset, viper's auth.accessToken is the file value.
	return src.cfg.Auth.AccessToken
}
