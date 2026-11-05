package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testState = "state-123"

func callback(t *testing.T, done chan callbackResult, target string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	newCallbackHandler(testState, done).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestCallbackHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		target     string
		wantStatus int
		want       *callbackResult
	}{
		{
			name:       "returns the code when state matches",
			target:     "/callback?code=abc&state=" + testState,
			wantStatus: http.StatusOK,
			want:       &callbackResult{code: "abc"},
		},
		{
			name:       "reports a refusal",
			target:     "/callback?error=access_denied&state=" + testState,
			wantStatus: http.StatusBadRequest,
			want:       &callbackResult{err: errString("authorization refused: access_denied")},
		},
		{
			name:       "reports a missing code",
			target:     "/callback?state=" + testState,
			wantStatus: http.StatusBadRequest,
			want:       &callbackResult{err: errString("callback has no authorization code")},
		},
		{
			name:       "ignores a wrong state and keeps waiting",
			target:     "/callback?code=abc&state=other",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ignores a missing state and keeps waiting",
			target:     "/callback?code=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ignores other paths",
			target:     "/favicon.ico",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			done := make(chan callbackResult, 1)
			rec := callback(t, done, tt.target)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.want == nil {
				assert.Empty(t, done)
				return
			}

			require.Len(t, done, 1)
			got := <-done
			assert.Equal(t, tt.want.code, got.code)
			if tt.want.err == nil {
				assert.NoError(t, got.err)
				return
			}
			assert.EqualError(t, got.err, tt.want.err.Error())
		})
	}
}

func TestCallbackHandler_SecondCallbackDoesNotBlock(t *testing.T) {
	t.Parallel()

	done := make(chan callbackResult, 1)
	callback(t, done, "/callback?code=first&state="+testState)
	rec := callback(t, done, "/callback?code=second&state="+testState)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, callbackResult{code: "first"}, <-done)
}

func TestCallbackHandler_EscapesErrorInPage(t *testing.T) {
	t.Parallel()

	done := make(chan callbackResult, 1)
	rec := callback(t, done, "/callback?error=%3Cscript%3E&state="+testState)

	assert.NotContains(t, rec.Body.String(), "<script>")
	assert.Contains(t, rec.Body.String(), "&lt;script&gt;")
}

func TestAuthorizeURL(t *testing.T) {
	t.Parallel()

	got := authorizeURL("http://localhost:13000/", "http://127.0.0.1:5555/callback", "chal", "st", "rudder-cli on host")

	u, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "localhost:13000", u.Host)
	assert.Equal(t, "/cli-login", u.Path)
	assert.Equal(t, url.Values{
		"redirect_uri":          {"http://127.0.0.1:5555/callback"},
		"code_challenge":        {"chal"},
		"code_challenge_method": {"S256"},
		"state":                 {"st"},
		"client_name":           {"rudder-cli on host"},
	}, u.Query())
}

func TestExchangeCode(t *testing.T) {
	t.Parallel()

	var got tokenRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/cliAuth/token", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))

		_, _ = w.Write([]byte(`{"accessToken":"pat","tokenType":"Bearer","workspaceId":"ws-1","workspaceName":"Prod","tokenName":"rudder-cli on host"}`))
	}))
	t.Cleanup(srv.Close)

	grant, err := exchangeCode(context.Background(), srv.Client(), srv.URL, "code", "verifier", "http://127.0.0.1:1/callback")

	require.NoError(t, err)
	assert.Equal(t, tokenRequest{
		GrantType:    "authorization_code",
		Code:         "code",
		CodeVerifier: "verifier",
		RedirectURI:  "http://127.0.0.1:1/callback",
	}, got)
	assert.Equal(t, &TokenGrant{
		AccessToken:   "pat",
		TokenType:     "Bearer",
		WorkspaceID:   "ws-1",
		WorkspaceName: "Prod",
		TokenName:     "rudder-cli on host",
	}, grant)
}

func TestExchangeCode_ErrorBody(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, status, body, want string
	}{
		{"cli-auth error", "400", `{"error":"invalid_grant"}`, "exchanging code: HTTP 400: invalid_grant"},
		{"with description", "400", `{"error":"invalid_grant","error_description":"code expired"}`, "exchanging code: HTTP 400: invalid_grant: code expired"},
		{"backend message only", "409", `{"message":"Access token limit reached"}`, "exchanging code: HTTP 409: Access token limit reached"},
		{"no JSON", "502", `bad gateway`, "exchanging code: HTTP 502"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				code, _ := strconv.Atoi(tc.status)
				w.WriteHeader(code)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)

			_, err := exchangeCode(context.Background(), srv.Client(), srv.URL, "code", "verifier", "http://127.0.0.1:1/callback")

			assert.EqualError(t, err, tc.want)
		})
	}
}

func TestExchangeCode_EmptyToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	_, err := exchangeCode(context.Background(), srv.Client(), srv.URL, "code", "verifier", "http://127.0.0.1:1/callback")

	assert.EqualError(t, err, "token response has no access token")
}

type errString string

func (e errString) Error() string { return string(e) }
