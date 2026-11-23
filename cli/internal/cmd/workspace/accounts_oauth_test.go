package workspace

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff/handofftest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
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

// readSpec is the file --write-spec wrote for an id.
func readSpec(t *testing.T, dir, id string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, id+".yaml"))
	require.NoError(t, err)
	return string(data)
}

// The spec create writes is the one 'import workspace' writes for an account
// nobody manages: the import metadata names the remote account, so the apply
// that follows adopts it. create itself claims nothing, which is what keeps an
// account it made from being a managed resource with no spec.
func TestCreateWritesTheSpecThatAdoptsTheAccount(t *testing.T) {
	server := useHandoffServer(t)
	calls := telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t,
		"--definition", oauthDefinition,
		"--name", "Google Ads (prod)",
		"--write-spec", specDir,
		"--no-browser",
	)
	require.NoError(t, err, out)

	accountID := server.LastSession().AccountID
	account, ok := server.Account(accountID)
	require.True(t, ok, "the session reported an account the server does not have")
	assert.Empty(t, account.ExternalID, "create must leave the account for apply to adopt")

	// Quoted scalars are the shared YAML formatter's output, the same one
	// 'import workspace' writes account specs with.
	assert.Equal(t, `version: "rudder/v1"
kind: "account"
metadata:
  import:
    workspaces:
      - resources:
          - remote_id: "`+accountID+`"
            urn: "account:google-ads-prod"
        workspace_id: "`+handofftest.WorkspaceID+`"
  name: "account"
spec:
  account_definition_name: "`+oauthDefinition+`"
  id: "google-ads-prod"
  name: "Google Ads (prod)"
`, readSpec(t, specDir, "google-ads-prod"))

	assert.Contains(t, out, "Account created: "+accountID)
	assert.Contains(t, out, "Wrote "+filepath.Join(specDir, "google-ads-prod.yaml"))
	assert.Contains(t, out, "rudder-cli apply")
	assert.Contains(t, out, `rudder_account_id: "#account:google-ads-prod"`)

	require.Len(t, *calls, 1)
	assert.Equal(t, telemetrytest.Call{
		Command: "workspace accounts create",
		Errored: false,
		Extras: []telemetry.KV{
			{K: "definition", V: oauthDefinition},
			{K: "writeSpec", V: true},
			{K: "noBrowser", V: true},
		},
	}, (*calls)[0])
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

	assert.NotContains(t, readSpec(t, specDir, "ads"), "config:")
}

// Without --write-spec nothing local is touched, and the account is left in the
// state an account made in the dashboard is in: unmanaged, and waiting for
// 'import workspace' to write its spec.
func TestCreateWithoutWriteSpecNamesTheImportPath(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser")
	require.NoError(t, err, out)

	entries, err := os.ReadDir(specDir)
	require.NoError(t, err)
	assert.Empty(t, entries)

	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Empty(t, account.ExternalID)

	assert.Contains(t, out, "Nothing manages this account yet")
	assert.Contains(t, out, "rudder-cli import workspace")
	assert.Contains(t, out, "rudder-cli apply")
	// There is no id yet, so there is no reference to print: 'import workspace'
	// picks the id when it writes the spec.
	assert.NotContains(t, out, "rudder_account_id")
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

// A timeout leaves the session alone and says what each of the two states the
// user may be in costs: a finished consent is adopted, an unfinished one is
// just run again. Neither needs a session id.
func TestCreateTimesOutAndNamesBothWaysBack(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	out, err := execCreate(t,
		"--definition", oauthDefinition, "--name", "Google Ads (prod)",
		"--no-browser", "--timeout", "200ms",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out after 200ms")
	assert.Contains(t, err.Error(), "rudder-cli import workspace")
	assert.Contains(t, err.Error(), "run this command again")
	assert.NotContains(t, err.Error(), "--resume")
	assert.Contains(t, out, "Waiting for you to finish")
	assert.Equal(t, 1, server.Sessions())
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
	// Nothing was created, so there is nothing to adopt: the hint says to retry,
	// and names the webapp flag that refuses the page outright.
	assert.Contains(t, err.Error(), "Nothing was created")
	assert.Contains(t, err.Error(), "CLI handoff enabled")
	assert.NotContains(t, err.Error(), "import workspace")
}

// Ctrl-C ends the wait and nothing else. The signal is sent to this process, so
// the command's own signal.NotifyContext is what turns it into a cancelled
// wait: without that wiring the default action would kill the test binary.
func TestCreateCancelledBySignalPrintsTheWayBack(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			server := useHandoffServer(t)
			telemetrytest.Record(t)

			// Keeps the test binary alive if the command ever stops handling
			// the signal itself, so a regression fails rather than killing the
			// run.
			guard := make(chan os.Signal, 1)
			signal.Notify(guard, sig)
			t.Cleanup(func() { signal.Stop(guard) })

			// Sent once the session is open, so the signal lands while the
			// wait is running rather than during the POST that opens it.
			server.OnSessionOpened(func(string) {
				go func() {
					time.Sleep(50 * time.Millisecond)
					assert.NoError(t, syscall.Kill(syscall.Getpid(), sig))
				}()
			})

			_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads",
				"--no-browser", "--timeout", "30s")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cancelled")
			assert.Contains(t, err.Error(), "rudder-cli import workspace")
			assert.Equal(t, 1, server.Sessions())
		})
	}
}

// A cancelled context ends the wait with the same explanation, through the
// runner rather than through a signal.
func TestCreateCancelLeavesTheSessionAlone(t *testing.T) {
	server := useHandoffServer(t)
	out := &bytes.Buffer{}
	r := runner(t, server, out, 30*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.create(ctx, oauthDefinition, "Google Ads (prod)", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.Contains(t, err.Error(), "rudder-cli import workspace")

	// The session is untouched, so finishing in the browser still creates the
	// account the import path then adopts.
	sessionID := server.LastSession().ID
	server.Consent(sessionID)
	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Empty(t, account.ExternalID)
}

// A definition the CLI cannot create is refused by name, before any request.
func TestCreateRefusesAnUnsupportedDefinition(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	for _, definition := range []string{"DESTINATION_LINKEDIN_ADS_OAUTH", "SOURCE_POSTGRES"} {
		t.Run(definition, func(t *testing.T) {
			// --no-browser and a short timeout, so a regression in the
			// guard fails here rather than opening a browser tab and
			// waiting out the default five minutes.
			_, err := execCreate(t, "--definition", definition, "--name", "Nope",
				"--no-browser", "--timeout", "2s")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no OAuth account definition named")
			assert.Contains(t, err.Error(), oauthDefinition)
			// The legacy pair cannot be adopted either: they carry no account
			// definition, so 'import workspace' never lists them.
			assert.Contains(t, err.Error(), "create it in the dashboard and give")
			assert.NotContains(t, err.Error(), "adopt them with")
		})
	}
	assert.Equal(t, 0, server.Sessions(), "an unsupported definition must not reach the API")
}

func TestCreateRequiresADefinitionAndName(t *testing.T) {
	useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execCreate(t, "--no-browser", "--timeout", "2s")
	assert.ErrorContains(t, err, "--definition is required")

	_, err = execCreate(t, "--definition", oauthDefinition, "--no-browser", "--timeout", "2s")
	assert.ErrorContains(t, err, "--name is required")
}

func TestReauthorizeKeepsTheAccountID(t *testing.T) {
	server := useHandoffServer(t)
	calls := telemetrytest.Record(t)
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

	require.Len(t, *calls, 1)
	assert.Equal(t, telemetrytest.Call{
		Command: "workspace accounts reauthorize",
		Errored: false,
		Extras:  []telemetry.KV{{K: "noBrowser", V: true}},
	}, (*calls)[0])
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

	_, err := execReauthorize(t, "#account:missing", "--no-browser", "--timeout", "2s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no account is adopted as "missing"`)
	// Both commands named here have to be able to help: apply cannot create an
	// OAuth account, so it is import workspace that writes the spec.
	assert.Contains(t, err.Error(), "rudder-cli import workspace")
	assert.Equal(t, 0, server.Sessions())
}

// An argument that is only the reference prefix names nothing, and must not be
// looked up as the empty external id.
func TestReauthorizeRefusesAnEmptyReference(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	for _, ref := range []string{"#account:", "  ", ""} {
		_, err := execReauthorize(t, ref, "--no-browser", "--timeout", "2s")
		assert.ErrorContains(t, err, "names no account")
	}
	assert.Equal(t, 0, server.Sessions())
}

// Only an OAuth account has an authorization to renew.
func TestReauthorizeRefusesANonOAuthAccount(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-pg", "Postgres", "SOURCE_POSTGRES", "pg"))

	_, err := execReauthorize(t, "#account:pg", "--no-browser", "--timeout", "2s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SOURCE_POSTGRES account")
	assert.Contains(t, err.Error(), "rudder-cli apply")
	assert.Equal(t, 0, server.Sessions())
}

// The same check has to run on the remote-id path, which skips the external-id
// lookup entirely.
func TestReauthorizeRefusesANonOAuthAccountByItsRemoteID(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-pg", "Postgres", "SOURCE_POSTGRES", ""))

	_, err := execReauthorize(t, "acc-pg", "--no-browser", "--timeout", "2s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SOURCE_POSTGRES account")
	assert.Equal(t, 0, server.Sessions())
}

// A definition the CLI has no entry for may well be an OAuth one:
// integrations-config carries several the registry does not list. Claiming it
// holds its own credentials would be a guess, so the refusal says what it does
// know.
func TestReauthorizeRefusesAnUnknownDefinition(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-x", "Enhanced", "DESTINATION_GOOGLE_ADWORDS_ENHANCED_CONVERSIONS_OAUTH", "enhanced"))

	_, err := execReauthorize(t, "#account:enhanced", "--no-browser", "--timeout", "2s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not know the account definition")
	assert.Contains(t, err.Error(), "RudderStack dashboard")
	assert.NotContains(t, err.Error(), "holds its own credentials")
	assert.Equal(t, 0, server.Sessions())
}

// Both commands fail against today's backend, so neither runs without the flag.
func TestOAuthAccountCommandsNeedTheExperimentalFlag(t *testing.T) {
	useHandoffServer(t)
	telemetrytest.Record(t)
	testutils.SetConfig(t, "flags.oauthAccounts", false)

	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--no-browser", "--timeout", "2s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
	assert.Contains(t, err.Error(), "RUDDERSTACK_X_OAUTH_ACCOUNTS=true")
	assert.Contains(t, err.Error(), "not released yet")

	_, err = execReauthorize(t, "#account:google-ads-prod", "--no-browser", "--timeout", "2s")
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
		// The consent page is behind a webapp flag of its own, which no CLI
		// setting turns on.
		assert.Contains(t, cmd, "enable-cli-account-handoff")
	}
	assert.Contains(t, newCmdCreateAccount().Short, "(experimental)")
	assert.Contains(t, newCmdReauthorizeAccount().Short, "(experimental)")
	// A bare #account:id is a comment to a shell, so the usage line quotes it.
	assert.Contains(t, newCmdReauthorizeAccount().Use, `"#account:id"`)
	assert.Contains(t, newCmdReauthorizeAccount().Use, `"account-id"`)
}

// Both are registered, and both stay out of --help until the flag is on, the
// way the experimental sources group does.
func TestNewCmdAccountsHidesTheOAuthCommandsUntilTheFlagIsOn(t *testing.T) {
	hidden := map[string]bool{}
	for _, sub := range NewCmdAccounts().Commands() {
		hidden[sub.Name()] = sub.Hidden
	}
	assert.Equal(t, map[string]bool{"list": false, "create": true, "reauthorize": true}, hidden)

	UnhideOAuthAccounts()
	for _, cmd := range oauthAccountCmds {
		assert.False(t, cmd.Hidden)
	}
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

// An id a spec in the directory uses and an id the workspace has claimed are
// both stepped around, and the message names whichever it was.
func TestCreateStepsAroundAnIDAlreadySpokenFor(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

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

	assert.Contains(t, out, `"google-ads-prod" is taken by an account in this workspace, so the spec id is "google-ads-prod-1"`)
	assert.Contains(t, readSpec(t, specDir, "google-ads-prod-1"), `id: "google-ads-prod-1"`)

	// The file that held the name is not touched, and the account that holds
	// the claim keeps it.
	content, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Equal(t, "version: \"rudder/v1\"\n", string(content))
	claimed, ok := server.Account("acc-live")
	require.True(t, ok)
	assert.Equal(t, "google-ads-prod", claimed.ExternalID)
}

// A spec file nothing has claimed is stepped around too, and the message names
// the file rather than an account. Two runs of the same command in the same
// directory is that case, and both specs adopt their own account.
func TestCreateTwiceWithTheSameNameWritesDistinctSpecs(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	var remoteIDs []string
	for range 2 {
		// Armed per run: openSession takes the callback once, so a test can let
		// one session through and leave the next pending.
		server.OnSessionOpened(server.Consent)
		out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads",
			"--write-spec", specDir, "--no-browser", "--timeout", "10s")
		require.NoError(t, err, out)
		remoteIDs = append(remoteIDs, server.LastSession().AccountID)
	}

	assert.Contains(t, readSpec(t, specDir, "ads"), `remote_id: "`+remoteIDs[0]+`"`)
	assert.Contains(t, readSpec(t, specDir, "ads-1"), `remote_id: "`+remoteIDs[1]+`"`)
	assert.Contains(t, readSpec(t, specDir, "ads-1"), `urn: "account:ads-1"`)
}

// A YAML file that is not an account spec, in a subdirectory, says nothing
// about account ids: refusing or renaming because of it was wrong, since a
// destination spec called google-ads.yaml is an ordinary thing to have.
func TestCreateIgnoresANonAccountSpecInASubdirectory(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	sub := filepath.Join(specDir, "destinations")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "google-ads.yaml"), []byte(
		"version: rudder/v1\nkind: destination\nspec:\n  id: google-ads\n"), 0o644))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Google Ads",
		"--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)

	assert.FileExists(t, filepath.Join(specDir, "google-ads.yaml"))
	assert.NotContains(t, out, "is taken by")
}

// The one --write-spec refusal left is a target that cannot hold a spec at all,
// and it still comes before the session: a refusal afterwards costs a consent
// the user has to give again.
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

// A directory the process cannot write to is the one write failure left that
// nothing can be done about beforehand. It must not fail the command: the
// account exists by then, and a non-zero exit invites the retry that runs a
// second consent.
func TestWriteFailureWarnsAndNamesTheImportPath(t *testing.T) {
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
	assert.Empty(t, account.ExternalID)

	assert.Contains(t, out, "Account created:")
	assert.Contains(t, out, "rudder-cli import workspace")
}

func TestReauthorizeTimeoutDoesNotOfferResume(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	server.SeedAccount(handofftest.NewAccount("acc-live", "Google Ads (prod)", oauthDefinition, "google-ads-prod"))

	// No consent: the wait runs out while the session is still pending.
	_, err := execReauthorize(t, "#account:google-ads-prod", "--no-browser", "--timeout", "200ms")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "timed out after 200ms")
	// reauthorize creates nothing, so there is nothing to adopt and no flag to
	// name: finishing in the browser is enough on its own.
	assert.NotContains(t, err.Error(), "--resume")
	assert.NotContains(t, err.Error(), "import workspace")
	assert.Contains(t, err.Error(), "re-authorizes")
}

// A --name whose spec id is free is written exactly as the user asked.
func TestCreateKeepsAFreeNameUnchanged(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()
	server.SeedAccount(handofftest.NewAccount("acc-other", "Bing Ads", oauthDefinition, "bing-ads"))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Google Ads (prod)",
		"--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)

	assert.Contains(t, readSpec(t, specDir, "google-ads-prod"), `id: "google-ads-prod"`)
	assert.NotContains(t, out, "is taken by")
}

// A display name the server trims must not be written untrimmed into the spec,
// or the spec drifts from the account on the first apply.
func TestCreateTrimsTheDisplayName(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "  Ads  ",
		"--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)

	assert.Contains(t, readSpec(t, specDir, "ads"), `name: "Ads"`)
	session := server.LastSession()
	assert.Equal(t, "Ads", session.Name, "the session carries the trimmed name too")
}

// writeAccountSpec is reached with a taken path only in a race now, since the
// id steps around the spec files it can see. The message still has to name the
// path once: writer.Write wraps os.OpenFile's own *PathError twice over, so the
// raw error says it three times.
func TestWriteAccountSpecNamesThePathOnce(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	plan := specPlan{externalID: "ads", workspaceID: "ws-1"}

	path, err := writeAccountSpec(context.Background(), dir, plan, "Ads", oauthDefinition, "acc-1")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ads.yaml"), path)

	_, err = writeAccountSpec(context.Background(), dir, plan, "Ads", oauthDefinition, "acc-1")
	require.Error(t, err)
	assert.Equal(t, path+" already exists, and the spec is never overwritten", err.Error())
	assert.Equal(t, 1, strings.Count(err.Error(), path))
}

// The id decides the file name, so a value that is not a plain file name must
// not reach the writer: it would create directories and a file outside the
// --write-spec target. The ids this command derives cannot do that; the guard
// is on the writer, where the path is joined.
func TestWriteAccountSpecRefusesAPathForAnID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	for _, id := range []string{"../escaped/ads", "/etc/ads", "../../ads"} {
		_, err := writeAccountSpec(context.Background(), dir,
			specPlan{externalID: id, workspaceID: "ws-1"}, "Ads", oauthDefinition, "acc-1")
		assert.ErrorContains(t, err, "is not a file name this command will write")
	}

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// A name with no letters or digits cannot become a spec id. It is refused
// before the session opens, because after consent the account exists and the
// same refusal leaves this run unable to write its spec.
func TestCreateRefusesANameWithNoSpecIDBeforeOpeningTheSession(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "!!!",
		"--no-browser", "--timeout", "2s")
	assert.ErrorContains(t, err, "no letters or digits")
	assert.Zero(t, server.Sessions(), "no session may be opened for a name with no spec id")
}

func TestCommandsRefuseAZeroTimeout(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)

	_, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads", "--timeout", "0", "--no-browser")
	assert.ErrorContains(t, err, "--timeout must be greater than zero")
	_, err = execReauthorize(t, "ads", "--timeout", "0", "--no-browser")
	assert.ErrorContains(t, err, "--timeout must be greater than zero")
	assert.Zero(t, server.Sessions())
}

// Each way a wait can end badly gets the advice that applies to it. A failed
// poll leaves the session open, so the caller's unresolved hint stands; an
// expired link and an unknown session id do not, and saying "finish in the
// browser" to either sends the user round the same loop.
func TestExplainWaitFailureMatchesTheAdviceToTheFailure(t *testing.T) {
	t.Parallel()
	r := &handoffRunner{out: &bytes.Buffer{}, timeout: time.Minute}
	hints := waitHints{unresolved: "  UNRESOLVED", denied: "DENIED", expired: "  EXPIRED"}

	for name, test := range map[string]struct {
		err      error
		contains string
		absent   []string
	}{
		"failed poll": {
			err: errors.New("could not parse error response"), contains: "UNRESOLVED",
			absent: []string{"DENIED", "EXPIRED"},
		},
		"timeout": {err: oauthhandoff.ErrTimedOut, contains: "UNRESOLVED", absent: []string{"EXPIRED"}},
		"cancelled": {
			err: context.Canceled, contains: "UNRESOLVED", absent: []string{"EXPIRED"},
		},
		"denied": {
			err: &oauthhandoff.DeniedError{Reason: "no"}, contains: "DENIED",
			absent: []string{"UNRESOLVED", "EXPIRED"},
		},
		"expired": {
			err: oauthhandoff.ErrExpired, contains: "EXPIRED",
			absent: []string{"UNRESOLVED", "DENIED"},
		},
		"unknown session": {
			err:    &client.APIError{HTTPStatusCode: 404, Message: "no such handoff session"},
			absent: []string{"UNRESOLVED", "DENIED", "EXPIRED"}, contains: "no authorization session with that id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := r.explainWaitFailure(test.err, hints)
			assert.ErrorContains(t, err, test.contains)
			for _, absent := range test.absent {
				assert.NotContains(t, err.Error(), absent)
			}
		})
	}
}

// An account spec holds its id in the file, not only in the file's name, and
// the loader reads specs from subdirectories. A file name counts only in the
// directory the spec is written to, which is the only place the write collides.
func TestLocalSpecIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "integrations")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	accountSpec := "version: rudder/v1\nkind: account\nmetadata:\n  name: ads\nspec:\n  id: ads\n  name: Ads\n"
	require.NoError(t, os.WriteFile(filepath.Join(sub, "integration.yaml"), []byte(accountSpec), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bing.yml"), []byte("version: rudder/v1\n"), 0o644))
	// Another kind's id is not an account id.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.yaml"), []byte(
		"version: rudder/v1\nkind: destination\nspec:\n  id: not-an-account\n"), 0o644))
	// A file whose whole name is the extension has no base name at all. It used
	// to be recorded as the empty id, which the namer then refused, failing the
	// run after the account existed.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".yaml"), []byte("version: rudder/v1\n"), 0o644))

	ids, err := localSpecIDs(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"ads":   filepath.Join(sub, "integration.yaml"),
		"bing":  filepath.Join(dir, "bing.yml"),
		"other": filepath.Join(dir, "other.yaml"),
	}, ids)
}

// A file in the --write-spec directory that would block a read must not stall
// the command, and must not be read at all.
func TestLocalSpecIDsSkipsAFileItCannotRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no FIFOs")
	}
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "blocking.yaml"), 0o644))

	done := make(chan struct{})
	go func() {
		defer close(done)
		ids, err := localSpecIDs(dir)
		assert.NoError(t, err)
		assert.Empty(t, ids)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("localSpecIDs blocked on a FIFO")
	}
}

// The directory walk and the namer run before the session opens, so nothing
// about the directory can fail once an account exists. A file named only
// ".yaml" is the case that used to.
func TestCreateWritesTheSpecWithAnOddFileInTheDirectory(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(specDir, ".yaml"), []byte("version: rudder/v1\n"), 0o644))

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Ads",
		"--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)
	assert.Contains(t, readSpec(t, specDir, "ads"), `id: "ads"`)
}

// The point of the spec create writes: the next apply adopts the account
// rather than trying to create it. The API refuses to create an OAuth account,
// and an account the CLI had claimed with no spec beside it was planned for
// deletion instead. This reads the written file back through the accounts
// provider and plans it against the workspace the fake server holds.
func TestTheWrittenSpecMakesTheNextApplyAdoptTheAccount(t *testing.T) {
	server := useHandoffServer(t)
	telemetrytest.Record(t)
	specDir := t.TempDir()

	server.OnSessionOpened(server.Consent)
	out, err := execCreate(t, "--definition", oauthDefinition, "--name", "Google Ads (prod)",
		"--write-spec", specDir, "--no-browser")
	require.NoError(t, err, out)

	c, err := client.New("test-token", client.WithBaseURL(server.URL()))
	require.NoError(t, err)
	provider := accounts.NewProvider(c.Accounts)

	path := filepath.Join(specDir, "google-ads-prod.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	spec, err := specs.New(data)
	require.NoError(t, err)
	require.NoError(t, provider.LoadSpec(path, spec))

	target, err := provider.ResourceGraph()
	require.NoError(t, err)

	// The source side is what apply sees remotely: accounts carrying an
	// external id. The new account carries none, so it is not there.
	remote, err := provider.LoadResourcesFromRemote(context.Background())
	require.NoError(t, err)
	state, err := provider.MapRemoteToState(remote)
	require.NoError(t, err)

	plan := planner.New(handofftest.WorkspaceID).Plan(syncer.StateToGraph(state), target)
	require.Len(t, plan.Operations, 1)
	assert.Equal(t, planner.Import, plan.Operations[0].Type,
		"the spec must be adopted, not created and not deleted")
	assert.Equal(t, "account:google-ads-prod", plan.Operations[0].Resource.URN())

	// Running that operation is what claims the id, through the same
	// external-id route the import flow uses.
	op := plan.Operations[0]
	_, err = provider.ImportRaw(context.Background(), op.Resource, op.Resource.ImportMetadata().RemoteId)
	require.NoError(t, err)

	account, ok := server.Account(server.LastSession().AccountID)
	require.True(t, ok)
	assert.Equal(t, "google-ads-prod", account.ExternalID)
}
