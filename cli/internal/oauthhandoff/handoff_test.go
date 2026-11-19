package oauthhandoff

import (
	"context"
	"errors"
	"io"
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

// A failed poll ends the wait rather than being retried: the session stays
// claimable, so the recovery is --resume.
func TestWaitStopsOnAPollFailure(t *testing.T) {
	api := &fakeAPI{err: errors.New("connection reset")}

	_, err := Wait(context.Background(), api, "sess", waitOpts(time.Second))
	assert.ErrorContains(t, err, "connection reset")
}

func TestWaitCapsThePollAtTheRemainingTime(t *testing.T) {
	api := &fakeAPI{state: &State{Status: StatusComplete}}

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
