package workspace

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

// The derivation is the import flow's, so the same display name gets the same
// spec id whether the account is created here or adopted by 'import workspace'.
func TestBaseExternalID(t *testing.T) {
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
			assert.Equal(t, want, baseExternalID(name))
		})
	}
}

// A spec file already using the id is the second run of the same command. It
// used to fail after the account had been created and claimed, so a script
// reading the exit code retried and the retry created a second OAuth account
// nobody can delete. Now it is the same collision as a claimed id, resolved
// the same way: one rule, whether or not --write-spec is passed.
func TestCreateStepsAroundAnExistingSpecFile(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	existing := filepath.Join(specDir, "google-ads-prod.yaml")
	require.NoError(t, os.WriteFile(existing, []byte("version: \"rudder/v1\"\n"), 0644))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t,
		"--definition", oauthDefinition,
		"--name", "Google Ads (prod)",
		"--write-spec", specDir,
		"--no-browser",
	)
	require.NoError(t, err, out)

	created, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Equal(t,
		handofftest.NewAccount(created.ID, "Google Ads (prod)", oauthDefinition, "google-ads-prod-1"),
		created)

	assert.Contains(t, out, "Claimed as google-ads-prod-1")
	assert.Contains(t, out, `"google-ads-prod" is taken by a spec in `+specDir)
	assert.FileExists(t, filepath.Join(specDir, "google-ads-prod-1.yaml"))

	// The file that held the name is not touched.
	content, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Equal(t, "version: \"rudder/v1\"\n", string(content))
}

// The one --write-spec refusal left is a target that cannot hold a spec at all,
// and it still has to come before the session: a refusal afterwards leaves an
// account the CLI cannot delete.
func TestCreateRefusesAnUnusableSpecDirBeforeOpeningTheSession(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	notADir := filepath.Join(t.TempDir(), "specs")
	require.NoError(t, os.WriteFile(notADir, nil, 0644))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t,
		"--definition", oauthDefinition, "--name", "Ads",
		"--write-spec", notADir, "--no-browser",
	)
	require.Error(t, err)
	assert.Equal(t, 0, server.Sessions(), "no account may be created when the spec cannot be written")
	assert.Contains(t, err.Error(), "--write-spec "+notADir+" cannot hold the account spec")
	assert.NotContains(t, out, "Account created")
}

// A write that fails for a reason no id can step around (here a directory the
// process cannot write to) must not fail the command: the remote work has
// succeeded by then, and a non-zero exit invites the retry that creates a
// second account.
func TestWriteFailureAfterTheClaimWarnsRatherThanFails(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	specDir := filepath.Join(t.TempDir(), "specs")
	require.NoError(t, os.Mkdir(specDir, 0500))
	t.Cleanup(func() { _ = os.Chmod(specDir, 0700) })

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t,
		"--definition", oauthDefinition, "--name", "Google Ads (prod)",
		"--write-spec", specDir, "--no-browser",
	)
	require.NoError(t, err, out)

	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Equal(t,
		handofftest.NewAccount(account.ID, "Google Ads (prod)", oauthDefinition, "google-ads-prod"),
		account)

	assert.Contains(t, out, "Claimed as google-ads-prod")
	assert.Contains(t, out, "The spec was not written, but the account is created and claimed")
	assert.Contains(t, out, "id: google-ads-prod")
	assert.Contains(t, out, "account_definition_name: "+oauthDefinition)
}

func TestReauthorizeTimeoutDoesNotOfferResume(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

	// No consent: the wait runs out while the session is still pending.
	_, err := execReauthorize(t, "#account:google-ads-prod", "--no-browser", "--timeout", "200ms")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "timed out after 200ms")
	// reauthorize has no --resume flag, so naming one sends the user to a
	// command that cannot take it. Finishing in the browser is enough on its own.
	assert.NotContains(t, err.Error(), "--resume")
	assert.Contains(t, err.Error(), "re-authorizes")
}

// A --name whose spec id is free is claimed exactly as the user asked.
func TestCreateClaimsAFreeNameUnchanged(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-other", "Bing Ads", oauthDefinition, "bing-ads"))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Google Ads (prod)", "--no-browser")
	require.NoError(t, err, out)

	created, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	want := handofftest.NewAccount(created.ID, "Google Ads (prod)", oauthDefinition, "google-ads-prod")
	assert.Equal(t, want, created)
	assert.Contains(t, out, "Claimed as google-ads-prod")
	assert.NotContains(t, out, "was already claimed")
}

// A display name that collides with an already-claimed account used to 409 on
// the claim, after the user had already given consent in the browser: the
// account existed and the command had nothing to do but fail. The id is
// disambiguated the way the import flow disambiguates one instead.
func TestCreateDisambiguatesAClaimedName(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Google Ads (prod)", "--no-browser")
	require.NoError(t, err, out)

	created, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	want := handofftest.NewAccount(created.ID, "Google Ads (prod)", oauthDefinition, "google-ads-prod-1")
	assert.Equal(t, want, created)

	// The adopted account keeps the id its spec names.
	existing, ok := server.Account("acc-live")
	require.True(t, ok)
	assert.Equal(t, handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"), existing)

	// Printed, so the id is seen now rather than discovered on the next apply.
	assert.Contains(t, out, "Claimed as google-ads-prod-1")
	assert.Contains(t, out, `"google-ads-prod" is taken by an account in this workspace`)
	assert.Contains(t, out, `rudder_account_id: "#account:google-ads-prod-1"`)
}

// Two runs of the same command claim two accounts, which is the sequence that
// 409d: the second run's id has to step around the first run's.
func TestCreateTwiceWithTheSameNameClaimsDistinctIDs(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	var claimed []string
	for range 3 {
		// Armed per run: openSession takes the callback once, so that a test can
		// let one session through and leave the next pending.
		server.OnSessionOpened(server.Consent)
		// Bounded so a regression that stops consenting fails in seconds rather
		// than waiting out the default timeout.
		out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser", "--timeout", "10s")
		require.NoError(t, err, out)

		account, ok := server.Account(server.LastSession().AccountID)
		require.True(t, ok)
		claimed = append(claimed, account.ExternalID)
	}

	assert.Equal(t, []string{"ads", "ads-1", "ads-2"}, claimed)
}

// The spec a disambiguated run writes names the id that was claimed, so the
// apply that follows adopts the account this run created rather than the one
// that held the name first.
func TestCreateWritesTheDisambiguatedIDIntoTheSpec(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()
	server.SeedAccount(handofftest.NewAccount("acc-live", "Ads", oauthDefinition, "ads"))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)

	spec, err := os.ReadFile(filepath.Join(specDir, "ads-1.yaml"))
	require.NoError(t, err)
	assert.Equal(t, `version: "rudder/v1"
kind: "account"
metadata:
  name: "ads-1"
spec:
  account_definition_name: "`+oauthDefinition+`"
  id: "ads-1"
  name: "Ads"
`, string(spec))
	assert.Contains(t, out, "Wrote "+filepath.Join(specDir, "ads-1.yaml"))
}

// writeAccountSpec is reached with a taken path only in a race now, since the
// id steps around the spec files it can see. The message still has to name the
// path once: writer.Write wraps os.OpenFile's own *PathError twice over, so the
// raw error says it three times.
func TestWriteAccountSpecNamesThePathOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	path, err := writeAccountSpec(context.Background(), dir, "ads", "Ads", oauthDefinition)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ads.yaml"), path)

	_, err = writeAccountSpec(context.Background(), dir, "ads", "Ads", oauthDefinition)
	require.Error(t, err)
	assert.Equal(t, path+" already exists, and the spec is never overwritten", err.Error())
	assert.Equal(t, 1, strings.Count(err.Error(), path))
}
