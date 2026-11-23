package oauthhandoff

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff/handofftest"
)

const definition = "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH"

// newClient points the real API client at the handoff test server, so these
// tests exercise the same request path the CLI uses.
func newClient(t *testing.T, server *handofftest.Server) *Client {
	t.Helper()

	c, err := client.New("test-token", client.WithBaseURL(server.URL()))
	require.NoError(t, err)
	return NewClient(c)
}

func waitOpts(timeout time.Duration) WaitOptions {
	return WaitOptions{Timeout: timeout, PollWait: 2 * time.Second}
}

func TestWaitReturnsTheCompletedSession(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	server.OnSessionOpened(server.Consent)
	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Google Ads (prod)"})
	require.NoError(t, err)
	assert.NotEmpty(t, session.WebURL)
	assert.True(t, session.ExpiresAt.After(time.Now()), "the session should carry a future expiry")

	state, err := Wait(context.Background(), api, session.ID, waitOpts(10*time.Second))
	require.NoError(t, err)
	assert.Equal(t, &State{
		Status:                StatusComplete,
		AccountID:             "acc-002",
		AccountDefinitionName: definition,
		Name:                  "Google Ads (prod)",
	}, state)
}

func TestWaitReportsTheDenialReason(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	const reason = "Your RudderStack user can't create accounts in this workspace. " +
		"Creating an integration account needs the credentialsManagement role."
	server.OnSessionOpened(func(id string) { server.Deny(id, reason) })

	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Denied"})
	require.NoError(t, err)

	_, err = Wait(context.Background(), api, session.ID, waitOpts(10*time.Second))

	var denied *DeniedError
	require.ErrorAs(t, err, &denied)
	assert.Equal(t, reason, denied.Reason)
}

func TestWaitReportsAnExpiredSession(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	server.OnSessionOpened(server.Expire)
	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Abandoned"})
	require.NoError(t, err)

	_, err = Wait(context.Background(), api, session.ID, waitOpts(10*time.Second))
	assert.ErrorIs(t, err, ErrExpired)
}

// A timeout must leave the session claimable: that is what makes --resume pick
// up an authorization finished after the CLI stopped waiting.
func TestWaitTimesOutAndLeavesTheSessionClaimable(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Slow"})
	require.NoError(t, err)

	_, err = Wait(context.Background(), api, session.ID, WaitOptions{Timeout: 200 * time.Millisecond})
	require.ErrorIs(t, err, ErrTimedOut)

	server.Consent(session.ID)
	state, err := Wait(context.Background(), api, session.ID, waitOpts(10*time.Second))
	require.NoError(t, err)
	assert.Equal(t, StatusComplete, state.Status)
	assert.Equal(t, 1, server.Sessions(), "resuming must not open a second session")
}

// Ctrl-C ends the wait and nothing else: the session is still pending, so the
// user can finish in the browser and resume.
func TestWaitCancelLeavesTheSessionPending(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Interrupted"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err = Wait(ctx, api, session.ID, waitOpts(30*time.Second))
	require.ErrorIs(t, err, context.Canceled)

	server.Consent(session.ID)
	state, err := Wait(context.Background(), api, session.ID, waitOpts(10*time.Second))
	require.NoError(t, err)
	assert.Equal(t, StatusComplete, state.Status)
}

// fakeAPI answers one fixed state, for the cases the server cannot produce.
type fakeAPI struct {
	state *State
	err   error
	wait  time.Duration
}

func (f *fakeAPI) Open(context.Context, OpenRequest) (*Session, error) {
	return &Session{ID: "sess", WebURL: "https://example.invalid/oauth/cli/sess"}, nil
}

func (f *fakeAPI) Poll(_ context.Context, _ string, wait time.Duration) (*State, error) {
	f.wait = wait
	return f.state, f.err
}

func TestWaitRefusesAnUnknownStatus(t *testing.T) {
	api := &fakeAPI{state: &State{Status: "revoked"}}

	_, err := Wait(context.Background(), api, "sess", waitOpts(time.Second))
	assert.ErrorContains(t, err, `unexpected handoff session status "revoked"`)
}

// A failed poll ends the wait rather than being retried. The session is
// untouched, so the caller's own recovery applies.
func TestWaitStopsOnAPollFailure(t *testing.T) {
	api := &fakeAPI{err: errors.New("connection reset")}

	_, err := Wait(context.Background(), api, "sess", waitOpts(time.Second))
	assert.ErrorContains(t, err, "connection reset")
}

func TestWaitCapsThePollAtTheRemainingTime(t *testing.T) {
	api := &fakeAPI{state: &State{Status: StatusComplete, AccountID: "acc-1"}}

	_, err := Wait(context.Background(), api, "sess", WaitOptions{Timeout: 50 * time.Millisecond})
	require.NoError(t, err)
	assert.LessOrEqual(t, api.wait, 50*time.Millisecond,
		"a poll must not outlive the caller's deadline")
}

func TestWaitPaintsWhileAPollIsInFlight(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	session, err := api.Open(context.Background(), OpenRequest{AccountDefinitionName: definition, Name: "Watched"})
	require.NoError(t, err)

	// The lesson from the spike: a loop that polls and then paints freezes the
	// counter for a whole poll. Painting has to happen during one.
	var ticks int
	go func() {
		time.Sleep(300 * time.Millisecond)
		server.Consent(session.ID)
	}()

	_, err = Wait(context.Background(), api, session.ID, WaitOptions{
		Timeout:   10 * time.Second,
		PollWait:  5 * time.Second,
		TickEvery: 20 * time.Millisecond,
		Tick:      func(time.Duration) { ticks++ },
	})
	require.NoError(t, err)
	assert.Greater(t, ticks, 5, "the elapsed counter froze for the length of a poll")
}

func TestOpenRequestValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request OpenRequest
		wantErr string
	}{
		{name: "create", request: OpenRequest{AccountDefinitionName: definition, Name: "Google Ads"}},
		{name: "reauthorize", request: OpenRequest{AccountID: "acc-1"}},
		{
			name:    "both shapes",
			request: OpenRequest{AccountID: "acc-1", AccountDefinitionName: definition},
			wantErr: "not both",
		},
		{name: "no definition", request: OpenRequest{Name: "Google Ads"}, wantErr: "account definition name"},
		{name: "no name", request: OpenRequest{AccountDefinitionName: definition}, wantErr: "account name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestOpenRefusesAnInvalidRequestWithoutCallingTheAPI(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	_, err := api.Open(context.Background(), OpenRequest{})
	require.Error(t, err)
	assert.Equal(t, 0, server.Sessions())
}

func TestOpenReportsTheAPIError(t *testing.T) {
	server := handofftest.Start(t)
	api := newClient(t, server)

	_, err := api.Open(context.Background(), OpenRequest{AccountID: "acc-missing"})
	assert.ErrorContains(t, err, "no such account")
}

// recordingDoer captures the request the client builds, to assert the wait
// parameter the contract bounds.
type recordingDoer struct {
	path string
	body []byte
}

func (d *recordingDoer) Do(_ context.Context, _, path string, body io.Reader) ([]byte, error) {
	d.path = path
	if body != nil {
		d.body, _ = io.ReadAll(body)
	}
	return []byte(`{"status":"pending"}`), nil
}

func TestPollClampsTheWaitToTheContractMaximum(t *testing.T) {
	doer := &recordingDoer{}

	_, err := NewClient(doer).Poll(context.Background(), "sess-1", 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "/v2/oauth/cli-sessions/sess-1?wait=2m0s", doer.path)
}

func TestPollDefaultsTheWait(t *testing.T) {
	doer := &recordingDoer{}

	_, err := NewClient(doer).Poll(context.Background(), "sess-1", 0)
	require.NoError(t, err)
	assert.Equal(t, "/v2/oauth/cli-sessions/sess-1?wait=30s", doer.path)
}

func TestCanOpenBrowser(t *testing.T) {
	for _, tc := range []struct {
		name string
		goos string
		env  map[string]string
		want bool
	}{
		{name: "macOS", goos: "darwin", want: true},
		{name: "Windows", goos: "windows", want: true},
		{name: "Linux desktop", goos: "linux", env: map[string]string{"DISPLAY": ":0"}, want: true},
		{name: "Linux Wayland", goos: "linux", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, want: true},
		{name: "Linux with BROWSER", goos: "linux", env: map[string]string{"BROWSER": "firefox"}, want: true},
		{name: "Linux headless", goos: "linux", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := canOpenBrowser(tc.goos, func(key string) string { return tc.env[key] })
			assert.Equal(t, tc.want, got)
		})
	}
}

// hangingAPI answers a poll only when its context ends, like a server that
// never replies.
type hangingAPI struct{}

func (hangingAPI) Open(context.Context, OpenRequest) (*Session, error) {
	return nil, errors.New("unused")
}

func (hangingAPI) Poll(ctx context.Context, _ string, _ time.Duration) (*State, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// The deadline covers a poll that is still in flight: a slow response cannot
// succeed after Timeout, and it ends as the timeout rather than as a request error.
func TestWaitEnforcesTheDeadlineWhileAPollIsInFlight(t *testing.T) {
	started := time.Now()
	_, err := Wait(context.Background(), hangingAPI{}, "sess", WaitOptions{
		Timeout: 100 * time.Millisecond, PollWait: time.Minute,
	})

	assert.ErrorIs(t, err, ErrTimedOut)
	assert.Less(t, time.Since(started), 5*time.Second)
}

func TestWaitReportsACallerCancellationAsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := Wait(ctx, hangingAPI{}, "sess", WaitOptions{Timeout: time.Minute, PollWait: time.Minute})
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrTimedOut)
}

func TestDeniedErrorDropsControlCharacters(t *testing.T) {
	err := &DeniedError{Reason: "\x1b[31mdenied\x1b[0m\rby role"}
	assert.Equal(t, "[31mdenied[0mby role", err.Error())
}

// A complete session always names the account the consent created. Without one
// there is nothing to write a spec against, and the CLI would print "Account
// created:" with no id after it. The real server never sends this.
func TestWaitRefusesACompleteSessionWithNoAccount(t *testing.T) {
	api := &fakeAPI{state: &State{Status: StatusComplete}}

	_, err := Wait(context.Background(), api, "sess", waitOpts(time.Second))
	assert.ErrorContains(t, err, "completed without naming an account")
}

// countingAPI answers pending at once, like the real server does when less
// than its own re-read interval is left of the requested wait.
type countingAPI struct {
	mu    sync.Mutex
	polls int
}

func (c *countingAPI) Open(context.Context, OpenRequest) (*Session, error) {
	return &Session{ID: "sess", WebURL: "https://example.invalid/oauth/cli/sess"}, nil
}

func (c *countingAPI) Poll(context.Context, string, time.Duration) (*State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.polls++
	return &State{Status: StatusPending}, nil
}

// Nothing but the server's hold paced the polls, so a server that answers
// pending immediately turned the tail of every wait into thousands of session
// reads. The floor is what bounds it.
func TestWaitPacesPollsThatReturnAtOnce(t *testing.T) {
	api := &countingAPI{}

	_, err := Wait(context.Background(), api, "sess", WaitOptions{Timeout: 1200 * time.Millisecond})
	assert.ErrorIs(t, err, ErrTimedOut)

	api.mu.Lock()
	defer api.mu.Unlock()
	// One at the start, then one per floor. The ticker resolution adds slack,
	// so this is a ceiling rather than an equality.
	assert.LessOrEqual(t, api.polls, 4, "a wait of 1.2s must not poll more than once per 500ms")
	assert.GreaterOrEqual(t, api.polls, 2, "it must still poll")
}

// stubDoer answers one fixed body, for the session shapes the test server
// cannot be made to produce.
type stubDoer struct{ body string }

func (d stubDoer) Do(context.Context, string, string, io.Reader) ([]byte, error) {
	return []byte(d.body), nil
}

// The URL the server returns is printed and handed to the host's browser
// opener, which reads its argument as something to launch: macOS open takes a
// leading -a as an application option, and a file:// or javascript: URL names
// a local handler. The real server builds the URL from its own dashboard
// address, so this guards against an API host that has been compromised or
// swapped out.
func TestOpenRefusesAWebURLItWillNotOpen(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"an application option": `{"sessionId":"s","webUrl":"-aCalculator"}`,
		"a local file":          `{"sessionId":"s","webUrl":"file:///Applications/Calculator.app"}`,
		"a script":              `{"sessionId":"s","webUrl":"javascript:alert(1)"}`,
		"no host":               `{"sessionId":"s","webUrl":"https:///oauth/cli/s"}`,
		"a user":                `{"sessionId":"s","webUrl":"https://someone@example.invalid/oauth/cli/s"}`,
		"plain http":            `{"sessionId":"s","webUrl":"http://example.invalid/oauth/cli/s"}`,
		"an escape sequence":    `{"sessionId":"s","webUrl":"https://example.invalid/\u001b[31m"}`,
		"a relative path":       `{"sessionId":"s","webUrl":"/oauth/cli/s"}`,
	} {
		t.Run(name, func(t *testing.T) {
			client := NewClient(stubDoer{body: body})

			_, err := client.Open(context.Background(), OpenRequest{AccountDefinitionName: "DEF", Name: "n"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "the handoff session's URL")
		})
	}
}

// http is allowed on a loopback host, which is what a local stack serves.
func TestOpenAcceptsLoopbackHTTP(t *testing.T) {
	t.Parallel()

	for _, url := range []string{
		"http://localhost:5555/oauth/cli/s",
		"http://127.0.0.1:5555/oauth/cli/s",
		"https://app.rudderstack.com/oauth/cli/s",
	} {
		client := NewClient(stubDoer{body: `{"sessionId":"s","webUrl":"` + url + `"}`})

		session, err := client.Open(context.Background(), OpenRequest{AccountDefinitionName: "DEF", Name: "n"})
		require.NoError(t, err)
		assert.Equal(t, url, session.WebURL)
	}
}

// Every opener below takes its argument as a scheme handler to launch, so a
// caller that skipped safeWebURL must not reach exec.
func TestOpenBrowserRefusesANonHTTPURL(t *testing.T) {
	t.Parallel()

	for _, url := range []string{"file:///Applications/Calculator.app", "-aCalculator", "javascript:alert(1)", ""} {
		assert.ErrorContains(t, OpenBrowser(url), "refusing to open")
	}
}

// A session with no id or no URL is not usable, and the CLI says so rather
// than printing an empty link.
func TestOpenRefusesASessionWithNoIDOrURL(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		`{"webUrl":"https://example.invalid/oauth/cli/s"}`,
		`{"sessionId":"s"}`,
		`{}`,
	} {
		client := NewClient(stubDoer{body: body})

		_, err := client.Open(context.Background(), OpenRequest{AccountDefinitionName: "DEF", Name: "n"})
		assert.ErrorContains(t, err, "missing its id or URL")
	}
}

// Printable is what every server-controlled string goes through before it is
// printed: a reason, an account id, a URL. Newlines go too, so a reason cannot
// add a line of its own to the CLI's output, and so do the invisible format
// characters that would reorder the line it sits on.
func TestPrintable(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct{ in, want string }{
		"colour codes":    {"\x1b[31mdenied\x1b[0m", "[31mdenied[0m"},
		"carriage return": {"denied\rby role", "deniedby role"},
		"newline":         {"denied\nAccount created: acc-1", "deniedAccount created: acc-1"},
		"bidi override":   {"ads\u202egnp.exe", "adsgnp.exe"},
		"zero width":      {"a\u200bb", "ab"},
		"plain text":      {"Your user can't create accounts.", "Your user can't create accounts."},
		"non-ascii":       {"Anzeigen für Kunden", "Anzeigen für Kunden"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, Printable(test.in))
		})
	}
}
