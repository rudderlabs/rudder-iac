package workspace

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff/handofftest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

const oauthDefinition = "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH"

// useHandoffServer points the whole CLI at the handoff test server and turns
// the experimental flag on, so a command runs end to end against it.
func useHandoffServer(t *testing.T) *handofftest.Server {
	t.Helper()

	server := handofftest.Start(t)
	testutils.SetConfig(t, "auth.accessToken", "test-token")
	testutils.SetConfig(t, "apiURL", server.URL())
	// The per-flag key is inert without the umbrella: GetConfig zeroes every
	// experimental flag when "experimental" is off.
	testutils.SetConfig(t, "experimental", true)
	testutils.SetConfig(t, "flags.oauthAccounts", true)
	for _, key := range []string{
		"concurrency.catalogClient",
		"concurrency.compositeProvider",
		"concurrency.catalogProvider",
		"concurrency.dataGraph",
	} {
		testutils.SetConfig(t, key, 1)
	}
	return server
}

// runner builds the command's own runner against the test server, for the
// cases that need a context of their own.
func runner(t *testing.T, server *handofftest.Server, out *bytes.Buffer, timeout time.Duration) *handoffRunner {
	t.Helper()

	c, err := client.New("test-token", client.WithBaseURL(server.URL()))
	require.NoError(t, err)
	return &handoffRunner{
		api:           oauthhandoff.NewClient(c),
		accounts:      c,
		out:           out,
		launchBrowser: false,
		timeout:       timeout,
	}
}

func execCreate(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := newCmdCreateAccount()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	err := telemetrytest.Execute(cmd, []string{"workspace", "accounts"}, args)
	return out.String(), err
}

func execReauthorize(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := newCmdReauthorizeAccount()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	err := telemetrytest.Execute(cmd, []string{"workspace", "accounts"}, args)
	return out.String(), err
}

func TestCreateClaimsTheAccountAndWritesTheSpec(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t,
		"--definition", oauthDefinition,
		"--name", "Google Ads (prod)",
		"--write-spec", specDir,
		"--no-browser",
	)
	require.NoError(t, err, out)

	// The claim is the point: without the external id on the remote account,
	// apply has no way to recognise it later.
	accountID := server.LastSession().AccountID
	account, ok := server.Account(accountID)
	require.True(t, ok, "the session reported an account the server does not have")
	assert.Equal(t, "google-ads-prod", account.ExternalID)

	spec, err := os.ReadFile(filepath.Join(specDir, "google-ads-prod.yaml"))
	require.NoError(t, err)
	// Quoted scalars are the shared YAML formatter's output, the same one
	// 'import workspace' writes account specs with.
	assert.Equal(t, `version: "rudder/v1"
kind: "account"
metadata:
  name: "google-ads-prod"
spec:
  account_definition_name: "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH"
  id: "google-ads-prod"
  name: "Google Ads (prod)"
`, string(spec))

	assert.Contains(t, out, "Account created: "+accountID)
	assert.Contains(t, out, "Claimed as google-ads-prod")
	assert.Contains(t, out, `rudder_account_id: "#account:google-ads-prod"`)
}

// An OAuth account has no authorable config: its definition declares no option
// fields and its secret lives in rudder-auth. A config block would be a lie,
// and Track A's validate rules refuse one.
func TestCreateWritesNoConfigBlock(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--write-spec", specDir, "--no-browser")
	require.NoError(t, err)

	spec, err := os.ReadFile(filepath.Join(specDir, "ads.yaml"))
	require.NoError(t, err)
	assert.NotContains(t, string(spec), "config:")
}

func TestCreateWithoutWriteSpecLeavesNoFile(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser")
	require.NoError(t, err, out)

	entries, err := os.ReadDir(specDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// --no-browser prints the link and waits identically. The same path runs on a
// headless host, where CanOpenBrowser is false.
func TestCreateWithoutABrowserPrintsTheLinkAndWaits(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser")
	require.NoError(t, err, out)

	assert.Contains(t, out, "/oauth/cli/"+server.LastSession().ID)
	assert.Contains(t, out, "Open that link in a browser on any machine signed in to RudderStack")
	assert.Contains(t, out, "Waiting for you to finish")
	assert.Contains(t, out, "Account created:")
}

// A timeout leaves the session claimable and names the command that picks it up.
func TestCreateTimesOutAndResumeClaimsTheSameAccount(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	out, err := execCreate(t,
		"--definition", oauthDefinition, "--name", "Google Ads (prod)",
		"--no-browser", "--timeout", "200ms",
	)
	require.Error(t, err)
	sessionID := server.LastSession().ID
	assert.Contains(t, err.Error(), "timed out after 200ms")
	assert.Contains(t, err.Error(), "rudder-cli workspace accounts create --resume "+sessionID)
	assert.Contains(t, out, "Waiting for you to finish")

	// The user finishes in the browser afterwards, then resumes.
	server.Consent(sessionID)
	out, err = execCreate(t, "--resume", sessionID, "--write-spec", specDir)
	require.NoError(t, err, out)

	assert.Equal(t, 1, server.Sessions(), "resume must not open a second session")
	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Equal(t, "google-ads-prod", account.ExternalID)
	// The session echoes the definition and name, so a resume needs no local state.
	spec, err := os.ReadFile(filepath.Join(specDir, "google-ads-prod.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(spec), `account_definition_name: "`+oauthDefinition+`"`)
}

func TestResumeRefusesFlagsTheSessionAlreadyCarries(t *testing.T) {
	useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execCreate(t, "--resume", "sess-1", "--definition", oauthDefinition)
	assert.ErrorContains(t, err, "drop --definition and --name")
}

// A permission denial comes from the page, not from the CLI's token: the
// browser half carries the user's own session and role.
func TestCreateSurfacesTheDenialFromThePage(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	server.OnSessionOpened(func(id string) {
		server.Deny(id, "Your RudderStack user can't create accounts in this workspace. "+
			"Creating an integration account needs the credentialsManagement role.")
	})

	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentialsManagement role")
	assert.Contains(t, err.Error(), "rudder-cli import workspace")
}

// Ctrl-C ends the wait and nothing else, so the session can still be resumed.
func TestCreateCancelLeavesTheSessionClaimable(t *testing.T) {
	server := useHandoffServer(t)
	out := &bytes.Buffer{}
	r := runner(t, server, out, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.create(ctx, oauthDefinition, "Google Ads (prod)", "", "")
	require.Error(t, err)
	sessionID := server.LastSession().ID
	assert.Contains(t, err.Error(), "cancelled")
	assert.Contains(t, err.Error(), "--resume "+sessionID)

	server.Consent(sessionID)
	require.NoError(t, r.create(context.Background(), "", "", "", sessionID))
	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Equal(t, "google-ads-prod", account.ExternalID)
}

// A definition the CLI cannot create is refused by name, before any request.
func TestCreateRefusesAnUnsupportedDefinition(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	for _, definition := range []string{"DESTINATION_LINKEDIN_ADS_OAUTH", "SOURCE_POSTGRES"} {
		t.Run(definition, func(t *testing.T) {
			_, err := execCreate(t, "--definition", definition, "--name", "Nope")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no OAuth account definition named")
			assert.Contains(t, err.Error(), oauthDefinition)
			assert.Contains(t, err.Error(), "rudder-cli import workspace")
		})
	}
	assert.Equal(t, 0, server.Sessions(), "an unsupported definition must not reach the API")
}

func TestCreateRequiresADefinitionAndName(t *testing.T) {
	useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execCreate(t)
	assert.ErrorContains(t, err, "--definition is required")

	_, err = execCreate(t, "--definition", oauthDefinition)
	assert.ErrorContains(t, err, "--name is required")
}

func TestReauthorizeKeepsTheAccountID(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

	server.OnSessionOpened(server.Consent)
	out, err := execReauthorize(t, "#account:google-ads-prod", "--no-browser")
	require.NoError(t, err, out)

	session := server.LastSession()
	assert.Equal(t, "acc-live", session.AccountID, "re-auth must name the account it renews")
	assert.Empty(t, session.AccountDefinitionName, "a re-auth session carries no definition")

	account, ok := server.Account("acc-live")
	require.True(t, ok)
	assert.Equal(t, "google-ads-prod", account.ExternalID, "re-auth must not disturb the claim")
	assert.Contains(t, out, "Re-authorized acc-live")
	assert.Contains(t, out, "The account id is unchanged")
}

func TestReauthorizeAcceptsABareExternalID(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

	server.OnSessionOpened(server.Consent)
	out, err := execReauthorize(t, "google-ads-prod", "--no-browser")
	require.NoError(t, err, out)
	assert.Equal(t, "acc-live", server.LastSession().AccountID)
}

// A user who copied the opaque id out of the dashboard gets the same flow.
func TestReauthorizeFallsBackToTheRemoteAccountID(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, ""))

	server.OnSessionOpened(server.Consent)
	out, err := execReauthorize(t, "acc-live", "--no-browser")
	require.NoError(t, err, out)
	assert.Equal(t, "acc-live", server.LastSession().AccountID)
}

func TestReauthorizeReportsAnUnknownReference(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execReauthorize(t, "#account:missing", "--no-browser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no account is adopted as "missing"`)
	assert.Equal(t, 0, server.Sessions())
}

// Only an OAuth account has an authorization to renew.
func TestReauthorizeRefusesANonOAuthAccount(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-pg", "Postgres", "SOURCE_POSTGRES", "pg"))

	_, err := execReauthorize(t, "#account:pg", "--no-browser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SOURCE_POSTGRES account")
	assert.Contains(t, err.Error(), "rudder-cli apply")
	assert.Equal(t, 0, server.Sessions())
}

// Both commands fail against today's backend, so neither runs without the flag.
func TestOAuthAccountCommandsNeedTheExperimentalFlag(t *testing.T) {
	useHandoffServer(t)
	telemetrytest.Record(t)
	testutils.SetConfig(t, "flags.oauthAccounts", false)

	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
	assert.Contains(t, err.Error(), "RUDDERSTACK_X_OAUTH_ACCOUNTS=true")
	assert.Contains(t, err.Error(), "not released yet")

	_, err = execReauthorize(t, "#account:google-ads-prod")
	assert.ErrorContains(t, err, "RUDDERSTACK_X_OAUTH_ACCOUNTS=true")
}

// The help says what the flag is and that the API behind it is not released,
// so the state of the feature is visible without reading the code.
func TestOAuthAccountCommandHelpNamesTheFlagAndTheMissingAPI(t *testing.T) {
	for _, cmd := range []string{newCmdCreateAccount().Long, newCmdReauthorizeAccount().Long} {
		assert.Contains(t, cmd, "RUDDERSTACK_X_OAUTH_ACCOUNTS=true")
		assert.Contains(t, cmd, "RUDDERSTACK_CLI_EXPERIMENTAL=true")
		assert.Contains(t, cmd, "cannot work against production")
		assert.Contains(t, cmd, "docs/oauth-account-handoff.md")
	}
	assert.Contains(t, newCmdCreateAccount().Short, "(experimental)")
	assert.Contains(t, newCmdReauthorizeAccount().Short, "(experimental)")
}

func TestNewCmdAccountsRegistersTheOAuthCommands(t *testing.T) {
	t.Parallel()

	var names []string
	for _, sub := range NewCmdAccounts().Commands() {
		names = append(names, sub.Name())
	}
	assert.ElementsMatch(t, []string{"list", "create", "reauthorize"}, names)
}

func TestSpecID(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]string{
		"Google Ads (prod)": "google-ads-prod",
		"LinkedIn Ads":      "linkedin-ads",
		"  spaced  out  ":   "spaced-out",
		"Ads // EU":         "ads-eu",
		"already-fine":      "already-fine",
		"Bing Ads 2026":     "bing-ads-2026",
		"!!!":               "",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, want, specID(name))
		})
	}
}
