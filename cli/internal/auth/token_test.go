package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

type fakeKeychain struct {
	tokens map[string]string
	err    error
	reads  []string
}

func (f *fakeKeychain) Get(account string) (string, error) {
	f.reads = append(f.reads, account)
	if f.err != nil {
		return "", f.err
	}
	token, ok := f.tokens[account]
	if !ok {
		return "", ErrTokenNotFound
	}
	return token, nil
}

func (f *fakeKeychain) Set(account, token string) error { return nil }
func (f *fakeKeychain) Delete(account string) error     { return nil }

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func cfgWith(tokenStore, fileToken string) config.Config {
	var cfg config.Config
	cfg.APIURL = "https://api.rudderstack.com"
	cfg.Auth.TokenStore = tokenStore
	cfg.Auth.AccessToken = fileToken
	return cfg
}

func TestResolveAccessToken(t *testing.T) {
	t.Parallel()

	stored := map[string]string{"api.rudderstack.com": "keychain-token"}

	tests := []struct {
		name      string
		env       map[string]string
		keychain  *fakeKeychain
		cfg       config.Config
		want      string
		wantReads []string
	}{
		{
			name:     "env var wins over keychain and file",
			env:      map[string]string{config.AccessTokenEnvVar: "env-token"},
			keychain: &fakeKeychain{tokens: stored},
			cfg:      cfgWith(config.TokenStoreKeychain, "file-token"),
			want:     "env-token",
		},
		{
			name:      "keychain wins over file after a browser login",
			keychain:  &fakeKeychain{tokens: stored},
			cfg:       cfgWith(config.TokenStoreKeychain, "file-token"),
			want:      "keychain-token",
			wantReads: []string{"api.rudderstack.com"},
		},
		{
			name:      "file token when the keychain has none",
			keychain:  &fakeKeychain{},
			cfg:       cfgWith(config.TokenStoreKeychain, "file-token"),
			want:      "file-token",
			wantReads: []string{"api.rudderstack.com"},
		},
		{
			name:      "file token when the keychain fails",
			keychain:  &fakeKeychain{err: errors.New("no dbus")},
			cfg:       cfgWith(config.TokenStoreKeychain, "file-token"),
			want:      "file-token",
			wantReads: []string{"api.rudderstack.com"},
		},
		{
			name:     "keychain is not read without a browser login",
			keychain: &fakeKeychain{tokens: stored},
			cfg:      cfgWith("", "file-token"),
			want:     "file-token",
		},
		{
			name:     "empty when no source has a token",
			keychain: &fakeKeychain{},
			cfg:      cfgWith("", ""),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveAccessToken(tokenSources{
				getenv:   env(tt.env),
				keychain: tt.keychain,
				cfg:      tt.cfg,
			})

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantReads, tt.keychain.reads)
		})
	}
}

func TestKeychainAccount(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "api.rudderstack.com", KeychainAccount("https://api.rudderstack.com"))
	assert.Equal(t, "localhost:15500", KeychainAccount("http://localhost:15500"))
	assert.Equal(t, "default", KeychainAccount(""))
	assert.Equal(t, "default", KeychainAccount("not a url"))
}

func TestReadTokenLine(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"tok\n":        "tok",
		"  tok  \r\n":  "tok",
		"tok":          "tok",
		"tok\nextra\n": "tok",
		"":             "",
	} {
		got, err := readTokenLine(strings.NewReader(input))
		assert.NoError(t, err)
		assert.Equal(t, want, got, "input %q", input)
	}
}
