package auth

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// TokenGrant is the config-backend reply to a code exchange.
type TokenGrant struct {
	AccessToken   string `json:"accessToken"`
	TokenType     string `json:"tokenType"`
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceName string `json:"workspaceName"`
	TokenName     string `json:"tokenName"`
}

type BrowserLoginOptions struct {
	AppURL     string
	APIURL     string
	ClientName string
	NoBrowser  bool
	Timeout    time.Duration
	// Progress gets the URL and status lines. It is stderr in the CLI, so an
	// agent that parses stdout as JSON still sees the URL.
	Progress    io.Writer
	OpenBrowser func(string) error
	HTTPClient  *http.Client
}

type callbackResult struct {
	code string
	err  error
}

// BrowserLogin runs the authorization code + PKCE flow over a loopback
// redirect (RFC 7636, RFC 8252) and returns the workspace token.
func BrowserLogin(ctx context.Context, opts BrowserLoginOptions) (*TokenGrant, error) {
	verifier, err := newCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generating code verifier: %w", err)
	}
	state, err := newState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// RFC 8252: listen on loopback only, never on all interfaces.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting callback listener: %w", err)
	}
	redirectURI := fmt.Sprintf("http://%s/callback", ln.Addr())

	done := make(chan callbackResult, 1)
	srv := &http.Server{
		Handler:           newCallbackHandler(state, done),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	loginURL := authorizeURL(opts.AppURL, redirectURI, codeChallengeS256(verifier), state, opts.ClientName)
	fmt.Fprintf(opts.Progress, "Open this URL to sign in and pick a workspace:\n\n  %s\n\n", loginURL)
	if !opts.NoBrowser {
		if err := opts.OpenBrowser(loginURL); err != nil {
			fmt.Fprintln(opts.Progress, "Could not open a browser. Open the URL above yourself.")
		} else {
			fmt.Fprintln(opts.Progress, "Opened your browser. Waiting for you to authorize...")
		}
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var res callbackResult
	select {
	case res = <-done:
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the browser: %w", ctx.Err())
	}
	if res.err != nil {
		return nil, res.err
	}

	return exchangeCode(ctx, opts.HTTPClient, opts.APIURL, res.code, verifier, redirectURI)
}

func authorizeURL(appURL, redirectURI, challenge, state, clientName string) string {
	q := url.Values{
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"client_name":           {clientName},
	}
	return strings.TrimSuffix(appURL, "/") + "/cli-login?" + q.Encode()
}

// newCallbackHandler serves the loopback redirect. It reports the first
// callback that carries the right state; a callback with a wrong state is
// refused and does not end the wait, so a stray request cannot abort a login.
func newCallbackHandler(state string, done chan<- callbackResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
			writePage(w, http.StatusBadRequest, "Sign-in failed", "The sign-in request does not match this terminal. Run rudder-cli auth login again.")
			return
		}

		var res callbackResult
		switch {
		case q.Get("error") != "":
			res.err = fmt.Errorf("authorization refused: %s", q.Get("error"))
		case q.Get("code") == "":
			res.err = errors.New("callback has no authorization code")
		default:
			res.code = q.Get("code")
		}

		if res.err != nil {
			writePage(w, http.StatusBadRequest, "Sign-in failed", res.err.Error()+". Return to the terminal.")
		} else {
			writePage(w, http.StatusOK, "rudder-cli is signed in", "You can close this tab and return to your terminal or agent.")
		}

		select {
		case done <- res:
		default:
		}
	})
}

type tokenRequest struct {
	GrantType    string `json:"grantType"`
	Code         string `json:"code"`
	CodeVerifier string `json:"codeVerifier"`
	RedirectURI  string `json:"redirectUri"`
}

func exchangeCode(ctx context.Context, httpClient *http.Client, apiURL, code, verifier, redirectURI string) (*TokenGrant, error) {
	body, err := json.Marshal(tokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		CodeVerifier: verifier,
		RedirectURI:  redirectURI,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding token request: %w", err)
	}

	endpoint := strings.TrimSuffix(apiURL, "/") + "/cliAuth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		return nil, fmt.Errorf("exchanging code: HTTP %d %s", res.StatusCode, e.Error)
	}

	grant := &TokenGrant{}
	if err := json.Unmarshal(data, grant); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if grant.AccessToken == "" {
		return nil, errors.New("token response has no access token")
	}
	return grant, nil
}

// OpenBrowser opens u in the default browser without waiting for it.
func OpenBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func writePage(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	title, message = html.EscapeString(title), html.EscapeString(message)
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:-apple-system,Segoe UI,sans-serif;background:#f4f2ee;display:grid;place-items:center;min-height:100vh;margin:0}
.card{background:#fff;border-radius:14px;box-shadow:0 8px 30px #0001;padding:32px 36px;max-width:420px}h1{font-size:20px;margin:0 0 8px}p{color:#6b665e}</style>
</head><body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, title, title, message)
}
