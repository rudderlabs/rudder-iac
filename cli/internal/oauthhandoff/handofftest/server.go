// Package handofftest serves the OAuth account browser handoff API that
// docs/oauth-account-handoff.md specifies, so the CLI half can be driven end to
// end by go test. None of this exists in the control plane yet: the shapes here
// are the proposal, and this server is the only implementation of them.
//
// It also answers the account endpoints the CLI calls around the handoff (the
// external-id claim, the account list and get) and /v2/workspace, so a whole
// command can run against one base URL.
package handofftest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// SessionTTL is how long a pending session stays claimable, matching the
// contract's ten minutes.
const SessionTTL = 10 * time.Minute

// WorkspaceID is the workspace this server reports for any access token.
const WorkspaceID = "ws-handoff-test"

// Account is the slice of the control plane's account shape the CLI reads.
type Account struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	ExternalID  string `json:"externalId,omitempty"`
	Name        string `json:"name"`
	Definition  struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Category string `json:"category"`
	} `json:"definition"`
}

// NewAccount builds an account with its definition filled in, which is all any
// test here needs to vary.
func NewAccount(id, name, definitionName, externalID string) Account {
	a := Account{ID: id, WorkspaceID: WorkspaceID, Name: name, ExternalID: externalID}
	a.Definition.Name = definitionName
	a.Definition.Category = "destination"
	return a
}

// Session is a handoff session as the server holds it.
type Session struct {
	ID                    string
	AccountDefinitionName string
	Name                  string
	AccountID             string
	Reauth                bool
	Status                string
	Error                 string
	ExpiresAt             time.Time

	// changed is closed on every state change so a long-poll wakes at once.
	// ponytail: close-and-replace rather than sync.Cond, with one waiter per
	// session. devlisten's store.Wait is the pattern for the real thing.
	changed chan struct{}
}

// Server is the fake control plane.
type Server struct {
	t *testing.T

	mu       sync.Mutex
	sessions map[string]*Session
	order    []string
	accounts map[string]Account
	next     int
	onOpened func(sessionID string)

	srv *httptest.Server
}

// Start brings the server up and tears it down with the test.
func Start(t *testing.T) *Server {
	t.Helper()

	s := &Server{
		t:        t,
		sessions: map[string]*Session{},
		accounts: map[string]Account{},
	}

	mux := http.NewServeMux()
	// Proposed handoff API, workspace access token auth: what the CLI calls.
	mux.HandleFunc("POST /v2/oauth/cli-sessions", s.openSession)
	mux.HandleFunc("GET /v2/oauth/cli-sessions/{id}", s.pollSession)
	// Proposed webapp page and its report, user session auth: what the browser calls.
	mux.HandleFunc("GET /oauth/cli/{id}", s.consentPage)
	mux.HandleFunc("POST /v2/oauth/cli-sessions/{id}/result", s.reportResult)
	// Existing public account API.
	mux.HandleFunc("PUT /v2/accounts/{id}/external-id", s.setExternalID)
	mux.HandleFunc("GET /v2/accounts/{id}", s.getAccount)
	mux.HandleFunc("GET /v2/accounts", s.listAccounts)
	mux.HandleFunc("GET /v2/workspace", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"workspace": map[string]string{"id": WorkspaceID}})
	})

	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

// URL is the base URL to point the CLI config at.
func (s *Server) URL() string { return s.srv.URL }

// SeedAccount puts an account in the fake workspace, for a flow that starts
// from an account that already exists.
func (s *Server) SeedAccount(account Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[account.ID] = account
}

// Account reads one account back, to assert what a claim wrote.
func (s *Server) Account(id string) (Account, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[id]
	return account, ok
}

// LastSession is the most recently opened session, to assert the payload the
// CLI sent.
func (s *Server) LastSession() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.order) == 0 {
		return nil
	}
	session := *s.sessions[s.order[len(s.order)-1]]
	return &session
}

// Sessions counts the sessions opened so far, which is how a resume is told
// apart from a second create.
func (s *Server) Sessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.order)
}

// OnSessionOpened plays the user for the next session only: once the CLI has
// opened one, fn runs on its own goroutine. Pass Consent or Deny.
func (s *Server) OnSessionOpened(fn func(sessionID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onOpened = fn
}

// Consent completes a session the way the webapp would: a create session mints
// an account, a re-auth session keeps the one it was opened with.
func (s *Server) Consent(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.Status != "pending" {
		return
	}
	if !session.Reauth {
		s.next++
		account := NewAccount(
			fmt.Sprintf("acc-%03d", s.next), session.Name, session.AccountDefinitionName, "")
		s.accounts[account.ID] = account
		session.AccountID = account.ID
	}
	session.Status = "complete"
	s.publishLocked(session)
}

// Deny refuses a session, standing in for a user whose role lacks
// credentialsManagement.
func (s *Server) Deny(sessionID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.Status != "pending" {
		return
	}
	session.Status, session.Error = "denied", reason
	s.publishLocked(session)
}

// Expire ages a session out without waiting SessionTTL.
func (s *Server) Expire(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.Status != "pending" {
		return
	}
	session.Status = "expired"
	s.publishLocked(session)
}

func (s *Server) publishLocked(session *Session) {
	close(session.changed)
	session.changed = make(chan struct{})
}

func (s *Server) openSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountDefinitionName string `json:"accountDefinitionName"`
		Name                  string `json:"name"`
		AccountID             string `json:"accountId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "malformed request body")
		return
	}

	// The discriminator the backend already uses for re-auth: oauth.service.ts:61,
	// isCreateOAuthAccountRequest is !('accountId' in request).
	reauth := req.AccountID != ""
	switch {
	case reauth && (req.AccountDefinitionName != "" || req.Name != ""):
		apiError(w, http.StatusBadRequest, "an account id cannot be combined with a definition or name")
		return
	case !reauth && req.AccountDefinitionName == "":
		apiError(w, http.StatusBadRequest, "accountDefinitionName is required")
		return
	case !reauth && req.Name == "":
		apiError(w, http.StatusBadRequest, "name is required")
		return
	}

	s.mu.Lock()
	if reauth {
		if _, ok := s.accounts[req.AccountID]; !ok {
			s.mu.Unlock()
			apiError(w, http.StatusNotFound, "no such account")
			return
		}
	}
	s.next++
	session := &Session{
		ID:                    fmt.Sprintf("sess-%08x", 0x9f3a2c1e+s.next),
		AccountDefinitionName: req.AccountDefinitionName,
		Name:                  req.Name,
		AccountID:             req.AccountID,
		Reauth:                reauth,
		Status:                "pending",
		ExpiresAt:             time.Now().Add(SessionTTL),
		changed:               make(chan struct{}),
	}
	s.sessions[session.ID] = session
	s.order = append(s.order, session.ID)
	opened := s.onOpened
	s.onOpened = nil
	s.mu.Unlock()

	if opened != nil {
		go opened(session.ID)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"sessionId": session.ID,
		"webUrl":    s.srv.URL + "/oauth/cli/" + session.ID,
		"expiresAt": session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// pollSession is the long-poll: it answers at once on a state change and
// otherwise returns the current state once wait runs out, so the CLI's loop is
// a plain re-request.
func (s *Server) pollSession(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	session, ok := s.sessions[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		apiError(w, http.StatusNotFound, "no such handoff session")
		return
	}

	wait, err := time.ParseDuration(orDefault(r.URL.Query().Get("wait"), "30s"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "wait is not a duration")
		return
	}
	if wait > 120*time.Second {
		wait = 120 * time.Second
	}

	s.mu.Lock()
	status, changed := session.Status, session.changed
	s.mu.Unlock()

	if status == "pending" && wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-changed:
		case <-timer.C:
		case <-r.Context().Done():
			return
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                session.Status,
		"accountId":             session.AccountID,
		"accountDefinitionName": session.AccountDefinitionName,
		"name":                  session.Name,
		"error":                 session.Error,
	})
}

// reportResult is what the webapp page calls. First writer wins, and a second
// call against a resolved session is a no-op, so a double-submitted page cannot
// create two accounts.
func (s *Server) reportResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"accountId"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "malformed request body")
		return
	}

	id := r.PathValue("id")
	s.mu.Lock()
	_, ok := s.sessions[id]
	s.mu.Unlock()
	if !ok {
		apiError(w, http.StatusNotFound, "no such handoff session")
		return
	}

	if req.Error != "" {
		s.Deny(id, req.Error)
	} else {
		s.Consent(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

// consentPage stands in for the webapp route, collapsing the real page's
// oauth/url, provider consent, redirect and oauth/callback into two buttons. It
// is here so the flow can be walked by hand against this server, not just by a
// test.
func (s *Server) consentPage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	session, ok := s.sessions[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "no such handoff session", http.StatusNotFound)
		return
	}

	action, subject := "Authorize", session.AccountDefinitionName
	if session.Reauth {
		action, subject = "Re-authorize", session.AccountID
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, consentHTML, action, subject, session.ID, session.ID)
}

func (s *Server) setExternalID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExternalID string `json:"externalId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ExternalID == "" {
		apiError(w, http.StatusBadRequest, "externalId is required")
		return
	}

	id := r.PathValue("id")
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[id]
	if !ok {
		apiError(w, http.StatusNotFound, "no such account")
		return
	}
	account.ExternalID = req.ExternalID
	s.accounts[id] = account
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	account, ok := s.accounts[r.PathValue("id")]
	s.mu.Unlock()
	if !ok {
		apiError(w, http.StatusNotFound, "no such account")
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	hasExternalID := r.URL.Query().Get("hasExternalId")

	s.mu.Lock()
	defer s.mu.Unlock()

	data := []Account{}
	for _, account := range s.accounts {
		switch hasExternalID {
		case "true":
			if account.ExternalID == "" {
				continue
			}
		case "false":
			if account.ExternalID != "" {
				continue
			}
		}
		data = append(data, account)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":   data,
		"paging": map[string]any{"total": len(data), "next": ""},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// apiError answers in the public API's error shape, so client.APIError carries
// the message through to the CLI.
func apiError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

const consentHTML = `<!doctype html>
<meta charset="utf-8"><title>RudderStack: authorize</title>
<style>
 body{font:16px/1.5 system-ui,sans-serif;max-width:34rem;margin:4rem auto;padding:0 1rem}
 code{background:#f1f1f3;padding:.1rem .3rem;border-radius:3px}
 button{font:inherit;padding:.6rem 1.1rem;border-radius:6px;border:1px solid #ccc;cursor:pointer}
 .go{background:#4f46e5;color:#fff;border-color:#4f46e5}
 .note{color:#666;font-size:.85rem;margin-top:2rem}
 .done{color:#117a39;font-weight:600}
</style>
<h2>%s this integration account</h2>
<p><b>%s</b> &middot; session <code>%s</code></p>
<p>The real page posts <code>oauth/url</code> under your dashboard session,
sends you to the provider's consent screen, takes the redirect back from
rudder-auth, posts <code>oauth/callback</code>, and reports the account id.</p>
<p id="act">
 <button class="go" onclick="report(null)">Authorize</button>
 <button onclick="report('Your RudderStack user can&#39;t create accounts in this workspace.')">Deny</button>
</p>
<p class="note">Test double. There is no provider, no rudder-auth and no
consent screen behind these buttons.</p>
<script>
async function report(error){
  await fetch('/v2/oauth/cli-sessions/%s/result',{method:'POST',
    headers:{'content-type':'application/json'},
    body:JSON.stringify(error?{error}:{accountId:null})});
  document.getElementById('act').innerHTML =
    '<span class="done">Done. Go back to your terminal.</span>';
}
</script>
`
