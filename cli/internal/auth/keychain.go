package auth

import (
	"errors"
	"net/url"

	"github.com/zalando/go-keyring"
)

const keychainService = "rudder-cli"

// ErrTokenNotFound means the keychain holds no token for the account.
var ErrTokenNotFound = errors.New("token not found in keychain")

// TokenStore keeps access tokens in the OS keychain, one per account.
type TokenStore interface {
	Get(account string) (string, error)
	Set(account, token string) error
	Delete(account string) error
}

type keychain struct{}

// NewKeychain returns the OS keychain: macOS Keychain, Windows Credential
// Manager, or the Secret Service on Linux.
func NewKeychain() TokenStore {
	return keychain{}
}

func (keychain) Get(account string) (string, error) {
	token, err := keyring.Get(keychainService, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrTokenNotFound
	}
	return token, err
}

func (keychain) Set(account, token string) error {
	return keyring.Set(keychainService, account, token)
}

func (keychain) Delete(account string) error {
	err := keyring.Delete(keychainService, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrTokenNotFound
	}
	return err
}

// KeychainAccount keys the token by API host, so a token is only read back for
// the host it was issued for. The config keeps one login at a time: a login to
// another host replaces the previous one.
func KeychainAccount(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil || u.Host == "" {
		return "default"
	}
	return u.Host
}
