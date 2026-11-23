// Package oauthhandoff talks to the control plane's browser handoff session,
// the server-held rendezvous that lets the CLI create or re-authorize an OAuth
// integration account without ever handling a provider token.
//
// A CLI loopback listener cannot be the OAuth redirect target: rudder-auth owns
// the redirect_uri, and the config backend takes the authorize URL from
// rudder-auth's 302 rather than choosing it (oauth.service.ts:577 and :584).
// Creating an OAuth account is also user-attributed, needing a userId and the
// credentialsManagement role that a workspace access token does not carry. So
// the browser runs the whole consent exchange under the user's dashboard
// session and the CLI watches a session the server holds.
//
// The server half is not released: it is open as rudder-config-backend#6972,
// rudder-control-plane#2806 and rudder-webapp#10426. docs/oauth-account-handoff.md
// is the contract; the types and verbs here match it exactly, and handofftest
// serves it for tests.
package oauthhandoff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// Status is a handoff session's state. Only these four are contractual; an
// unknown value aborts the wait rather than being treated as pending, so a new
// server status cannot silently look like progress.
type Status string

const (
	StatusPending  Status = "pending"
	StatusComplete Status = "complete"
	StatusDenied   Status = "denied"
	StatusExpired  Status = "expired"
)

const (
	// DefaultPollWait is how long one long-poll hangs server-side. Short enough
	// to stay under a proxy's idle timeout, long enough that a five-minute wait
	// is ten requests rather than hundreds.
	DefaultPollWait = 30 * time.Second
	// MaxPollWait is the contract's ceiling on the wait parameter.
	MaxPollWait = 120 * time.Second
	// pollSlack is the grace a single poll gets over its own wait before the
	// request is abandoned, so a server that never answers cannot hang the CLI
	// past the overall timeout.
	pollSlack = 10 * time.Second

	sessionsPath = "/v2/oauth/cli-sessions"
)

var (
	// ErrTimedOut ends a wait that ran past the caller's own deadline. The
	// session is untouched and stays claimable, which is what --resume reads.
	ErrTimedOut = errors.New("timed out waiting for consent")
	// ErrExpired ends a wait on a session the server let expire.
	ErrExpired = errors.New("the authorization link expired before it was used")
)

// DeniedError carries the reason the browser half refused, so the CLI reports
// what the page decided instead of guessing from its own token.
type DeniedError struct {
	Reason string
}

func (e *DeniedError) Error() string {
	if e.Reason == "" {
		return "authorization was refused"
	}
	return stripControl(e.Reason)
}

// stripControl drops control characters, so a reason the server controls cannot
// move the cursor or recolour the terminal it is printed to.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, s)
}

// OpenRequest opens a session. Exactly one of its two shapes is valid, and the
// discriminator is AccountID, mirroring the backend's own isCreateOAuthAccountRequest
// (oauth.service.ts:61): a definition and a name create an account, an account
// id re-authorizes one.
type OpenRequest struct {
	AccountDefinitionName string `json:"accountDefinitionName,omitempty"`
	Name                  string `json:"name,omitempty"`
	AccountID             string `json:"accountId,omitempty"`
}

// Validate rejects a request the server would refuse with a 400, so a mistake
// in the CLI surfaces as a CLI error rather than an API error.
func (r OpenRequest) Validate() error {
	reauth := r.AccountID != ""
	switch {
	case reauth && (r.AccountDefinitionName != "" || r.Name != ""):
		return errors.New("a handoff session carries either an account id or a definition and name, not both")
	case reauth:
		return nil
	case r.AccountDefinitionName == "":
		return errors.New("a handoff session needs an account definition name")
	case r.Name == "":
		return errors.New("a handoff session needs an account name")
	}
	return nil
}

// Session is what opening returns: the id to poll and the URL to send the user to.
type Session struct {
	ID        string    `json:"sessionId"`
	WebURL    string    `json:"webUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// State is one poll's answer. AccountDefinitionName and Name are echoed from
// the opening request, which is what lets --resume rebuild a spec with no local
// state to carry between runs.
type State struct {
	Status                Status `json:"status"`
	AccountID             string `json:"accountId"`
	AccountDefinitionName string `json:"accountDefinitionName"`
	Name                  string `json:"name"`
	Error                 string `json:"error"`
}

// API is the handoff contract. Two calls, both documented in
// docs/oauth-account-handoff.md.
type API interface {
	// Open creates a pending session.
	Open(ctx context.Context, req OpenRequest) (*Session, error)
	// Poll reads a session, holding the request open for up to wait while the
	// session is pending.
	Poll(ctx context.Context, sessionID string, wait time.Duration) (*State, error)
}

// Doer is the slice of *client.Client this package uses: an authenticated
// request against the public API host.
type Doer interface {
	Do(ctx context.Context, method, path string, body io.Reader) ([]byte, error)
}

// Client is the HTTP implementation of API.
type Client struct {
	doer Doer
}

func NewClient(doer Doer) *Client {
	return &Client{doer: doer}
}

func (c *Client) Open(ctx context.Context, req OpenRequest) (*Session, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling handoff request: %w", err)
	}

	data, err := c.doer.Do(ctx, "POST", sessionsPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("opening a browser handoff session: %w", err)
	}

	session := &Session{}
	if err := json.Unmarshal(data, session); err != nil {
		return nil, fmt.Errorf("decoding the handoff session: %w", err)
	}
	if session.ID == "" || session.WebURL == "" {
		return nil, fmt.Errorf("the handoff session is missing its id or URL")
	}
	return session, nil
}

func (c *Client) Poll(ctx context.Context, sessionID string, wait time.Duration) (*State, error) {
	if wait <= 0 {
		wait = DefaultPollWait
	}
	if wait > MaxPollWait {
		wait = MaxPollWait
	}

	// A poll is bounded independently of the overall wait: the server holds the
	// request for `wait`, and pollSlack covers the round trip.
	ctx, cancel := context.WithTimeout(ctx, wait+pollSlack)
	defer cancel()

	path := fmt.Sprintf("%s/%s?wait=%s", sessionsPath, url.PathEscape(sessionID), wait)
	data, err := c.doer.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("checking the handoff session: %w", err)
	}

	state := &State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("decoding the handoff session state: %w", err)
	}
	return state, nil
}

// WaitOptions configures Wait. Tick is optional: it paints progress, and is
// left nil when nothing is watching.
type WaitOptions struct {
	// Timeout bounds the whole wait. Reaching it returns ErrTimedOut and leaves
	// the session claimable.
	Timeout time.Duration
	// PollWait is one long-poll's server-side hold. Zero means DefaultPollWait.
	PollWait time.Duration
	// Tick is called with the elapsed wait, every TickEvery.
	Tick func(elapsed time.Duration)
	// TickEvery defaults to 120ms, fast enough to animate a spinner.
	TickEvery time.Duration
}

// Wait long-polls until the session resolves or Timeout passes.
//
// The poll runs on its own goroutine because it hangs for up to PollWait: a
// loop that polls and then paints leaves the elapsed counter frozen for thirty
// seconds at a stretch, which reads as a hung CLI. The ticker paints while a
// poll is in flight.
//
// A failed poll ends the wait instead of being retried. The session is still
// claimable, so the recovery is --resume rather than a retry loop here.
//
// The deadline covers the polls too: each poll runs under a context that ends at
// the deadline, so a slow response cannot succeed after Timeout.
func Wait(ctx context.Context, api API, sessionID string, opts WaitOptions) (*State, error) {
	type result struct {
		state *State
		err   error
	}

	tickEvery := opts.TickEvery
	if tickEvery <= 0 {
		tickEvery = 120 * time.Millisecond
	}
	pollWait := opts.PollWait
	if pollWait <= 0 {
		pollWait = DefaultPollWait
	}

	var (
		started  = time.Now()
		deadline = started.Add(opts.Timeout)
		polls    = make(chan result, 1)
		inFlight bool
	)

	// Ends at the deadline, and is cancelled on return so a poll still in
	// flight is abandoned rather than left running.
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	for {
		if !inFlight {
			// Only checked between polls: a poll is capped at the remaining
			// time, so it cannot run past the deadline on its own.
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, ErrTimedOut
			}
			inFlight = true
			go func(wait time.Duration) {
				state, err := api.Poll(pollCtx, sessionID, wait)
				polls <- result{state: state, err: err}
			}(min(pollWait, remaining))
		}

		if opts.Tick != nil {
			opts.Tick(time.Since(started))
		}

		select {
		case <-ticker.C:
		case <-pollCtx.Done():
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ErrTimedOut
		case polled := <-polls:
			inFlight = false
			if polled.err != nil {
				// A poll cut off by the deadline is the timeout, not a request error.
				if ctx.Err() == nil && errors.Is(polled.err, context.DeadlineExceeded) && !time.Now().Before(deadline) {
					return nil, ErrTimedOut
				}
				return nil, polled.err
			}
			switch polled.state.Status {
			case StatusPending:
			case StatusComplete:
				return polled.state, nil
			case StatusDenied:
				return nil, &DeniedError{Reason: polled.state.Error}
			case StatusExpired:
				return nil, ErrExpired
			default:
				return nil, fmt.Errorf("unexpected handoff session status %q", polled.state.Status)
			}
		}
	}
}
