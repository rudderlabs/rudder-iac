package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

// FakeWorkspaceID is the workspace the fake API reports for the access token.
const FakeWorkspaceID = "ws-fake"

// UseFakeAPI points the CLI config at a local server that resolves the
// workspace behind the access token and fails every other request, so a
// command gets past authentication and then fails at its first real API call.
func UseFakeAPI(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/workspace" {
			fmt.Fprintf(w, `{"workspace":{"id":%q}}`, FakeWorkspaceID)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"fake API failure"}`)
	}))
	t.Cleanup(srv.Close)

	SetConfig(t, "auth.accessToken", "fake-token")
	SetConfig(t, "apiURL", srv.URL)

	// Without config.InitConfig these have no defaults, and a zero
	// concurrency is rejected while building the providers.
	for _, key := range []string{
		"concurrency.catalogClient",
		"concurrency.compositeProvider",
		"concurrency.catalogProvider",
		"concurrency.dataGraph",
	} {
		SetConfig(t, key, 1)
	}
}

// SetConfig overrides a CLI config key for the duration of the test.
func SetConfig(t *testing.T, key string, value any) {
	t.Helper()

	previous := viper.Get(key)
	viper.Set(key, value)
	t.Cleanup(func() { viper.Set(key, previous) })
}
