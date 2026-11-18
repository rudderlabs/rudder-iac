package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// accountRefPrefix is the reference form a destination spec uses for its
// integration account, which is also what reauthorize takes as its argument so
// a user never has to look up the opaque remote id.
const accountRefPrefix = "#" + accounts.AccountSpecKind + ":"

const defaultHandoffTimeout = 5 * time.Minute

// experimentalNotice is on both commands' help, because neither can work until
// the control plane grows the handoff API that docs/oauth-account-handoff.md
// specifies. It is not a flag that merely hides a finished feature.
const experimentalNotice = `This command is EXPERIMENTAL and cannot work against production yet: the
browser handoff API it calls is not released. See
docs/oauth-account-handoff.md for the contract it is built against.

To enable it, set both:
  RUDDERSTACK_CLI_EXPERIMENTAL=true
  RUDDERSTACK_X_OAUTH_ACCOUNTS=true
or run 'rudder-cli experimental enable oauthAccounts' with
RUDDERSTACK_CLI_EXPERIMENTAL=true. The per-flag variable does nothing on its
own.`

const experimentalDisabledMessage = "this command is experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and " +
	"RUDDERSTACK_X_OAUTH_ACCOUNTS=true, or run 'rudder-cli experimental enable oauthAccounts' with " +
	"RUDDERSTACK_CLI_EXPERIMENTAL=true.\n" +
	"  Note that the control-plane handoff API this needs is not released yet, so it cannot work against production."

// requireOAuthAccounts gates both commands on the experimental flag. The
// umbrella RUDDERSTACK_CLI_EXPERIMENTAL is what makes the per-flag variable
// readable at all, so a user who sets only one gets told about both.
func requireOAuthAccounts(*cobra.Command, []string) error {
	if config.GetConfig().ExperimentalFlags.OAuthAccounts {
		return nil
	}
	return errors.New(experimentalDisabledMessage)
}

func newCmdCreateAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an OAuth integration account through the browser (experimental)",
		Long: `Create an OAuth integration account by authorizing it in a browser.

An OAuth account's credentials are held by RudderStack's auth service, and the
public account API refuses to create one. So the CLI opens a handoff session,
sends you to a RudderStack page that runs the provider consent flow under your
own dashboard session, and waits. It then claims the new account under the
spec id it derives from --name, and with --write-spec writes the matching
'kind: account' spec, so 'rudder-cli apply' adopts the same account afterwards.

` + experimentalNotice,
		Args:    cobra.NoArgs,
		PreRunE: requireOAuthAccounts,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			var (
				definition, _ = cmd.Flags().GetString("definition")
				name, _       = cmd.Flags().GetString("name")
				specDir, _    = cmd.Flags().GetString("write-spec")
				resume, _     = cmd.Flags().GetString("resume")
				noBrowser, _  = cmd.Flags().GetBool("no-browser")
				timeout, _    = cmd.Flags().GetDuration("timeout")
			)

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "definition", V: definition},
					{K: "writeSpec", V: specDir != ""},
					{K: "resume", V: resume != ""},
					{K: "noBrowser", V: noBrowser},
				}...)
			}()

			if err := validateCreateFlags(definition, name, resume); err != nil {
				return err
			}

			runner, err := newHandoffRunner(cmd.OutOrStdout(), noBrowser, timeout)
			if err != nil {
				return err
			}

			return runner.create(cmd.Context(), definition, name, specDir, resume)
		},
	}

	cmd.Flags().String("definition", "", "Account definition name to create an account for")
	cmd.Flags().String("name", "", "Display name for the new account")
	cmd.Flags().String("write-spec", "", "Directory to write the 'kind: account' spec into")
	cmd.Flags().String("resume", "", "Session id of an authorization already finished in the browser")
	cmd.Flags().Bool("no-browser", false, "Print the authorization link instead of opening a browser")
	cmd.Flags().Duration("timeout", defaultHandoffTimeout, "How long to wait for the browser authorization")

	return cmd
}

func newCmdReauthorizeAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reauthorize <" + accountRefPrefix + "id | external-id>",
		Short: "Re-authorize an OAuth integration account through the browser (experimental)",
		Long: `Re-authorize an OAuth integration account whose grant was revoked or expired.

Takes the account as its spec reference ("` + accountRefPrefix + `google-ads-prod") or as the
bare external id, and resolves it to the remote account the project adopted.
The authorization runs in a browser, exactly as it does for create, and the
account's secret is replaced in place: the remote account id does not change,
so no spec and no destination needs editing.

Routine token refresh is server-side, so this is for a revoked or expired
grant, not for upkeep. An unattended 'apply' never needs it.

` + experimentalNotice,
		Args:    cobra.ExactArgs(1),
		PreRunE: requireOAuthAccounts,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var (
				noBrowser, _ = cmd.Flags().GetBool("no-browser")
				timeout, _   = cmd.Flags().GetDuration("timeout")
			)

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "noBrowser", V: noBrowser},
				}...)
			}()

			runner, err := newHandoffRunner(cmd.OutOrStdout(), noBrowser, timeout)
			if err != nil {
				return err
			}

			return runner.reauthorize(cmd.Context(), args[0])
		},
	}

	cmd.Flags().Bool("no-browser", false, "Print the authorization link instead of opening a browser")
	cmd.Flags().Duration("timeout", defaultHandoffTimeout, "How long to wait for the browser authorization")

	return cmd
}

// validateCreateFlags refuses a definition the CLI cannot create before any
// network call, naming the ones it can.
func validateCreateFlags(definition, name, resume string) error {
	if resume != "" {
		// A resumed session carries its own definition and name, so neither
		// flag is needed and a conflicting one would be ignored silently.
		if definition != "" || name != "" {
			return errors.New("--resume reads the definition and name from the session: drop --definition and --name")
		}
		return nil
	}

	if definition == "" {
		return errors.New("--definition is required (or --resume, to pick up a finished authorization)")
	}
	if name == "" {
		return errors.New("--name is required: it is the account's display name and the spec id is derived from it")
	}
	if !accounts.IsOAuthDefinition(definition) {
		return fmt.Errorf(`no OAuth account definition named %q.
  rudder-cli can create accounts for: %s
  Another definition's account is not created through a browser: write a 'kind: account' spec and run 'rudder-cli apply'.
  linkedin_ads and bingads_offline_conversions have no account definition at all, because their accounts come from the
  legacy role-keyed OAuth flow. Create those in the dashboard, then adopt them with 'rudder-cli import workspace'`,
			definition, strings.Join(accounts.OAuthDefinitionNames(), ", "))
	}
	return nil
}

// handoffRunner is the CLI half of the browser handoff: it owns narration, the
// wait, and the claim that follows.
type handoffRunner struct {
	api      oauthhandoff.API
	accounts *client.Client
	out      io.Writer
	// launchBrowser is false for --no-browser and on a host with no opener.
	launchBrowser bool
	timeout       time.Duration
}

func newHandoffRunner(out io.Writer, noBrowser bool, timeout time.Duration) (*handoffRunner, error) {
	d, err := app.NewDeps()
	if err != nil {
		return nil, err
	}

	c := d.Client()
	return &handoffRunner{
		api:           oauthhandoff.NewClient(c),
		accounts:      c,
		out:           out,
		launchBrowser: !noBrowser && oauthhandoff.CanOpenBrowser(),
		timeout:       timeout,
	}, nil
}

func (r *handoffRunner) create(ctx context.Context, definition, name, specDir, resume string) error {
	// Checked before the session is opened: a refusal here costs a flag
	// correction, while the same refusal after the account exists leaves an
	// OAuth account the CLI has no way to delete.
	if err := checkSpecTarget(specDir, specID(name)); err != nil {
		return err
	}

	sessionID := resume
	if sessionID == "" {
		session, err := r.api.Open(ctx, oauthhandoff.OpenRequest{
			AccountDefinitionName: definition, Name: name,
		})
		if err != nil {
			return err
		}
		sessionID = session.ID

		fmt.Fprintf(r.out, "\nAuthorizing %q (%s).\n", name, definition)
		r.narrate(session)
	} else {
		fmt.Fprintf(r.out, "\nPicking up authorization session %s.\n", sessionID)
	}

	state, err := oauthhandoff.Wait(ctx, r.api, sessionID, r.waitOptions())
	if err != nil {
		return r.explainWaitFailure(err, fmt.Sprintf(
			"  The link stays valid until it expires. Finish in the browser, then run:\n"+
				"    rudder-cli workspace accounts create --resume %s", sessionID))
	}
	fmt.Fprintln(r.out, ui.Success("Account created: "+state.AccountID))

	// The claim needs nothing new from the API: PUT /v2/accounts/{id}/external-id
	// is already open to the CLI and already idempotent.
	externalID := specID(firstNonEmpty(name, state.Name))
	if externalID == "" {
		return fmt.Errorf("the session reported no account name, so there is no spec id to claim %s under", state.AccountID)
	}
	if err := r.accounts.Accounts.SetExternalID(ctx, state.AccountID, externalID); err != nil {
		return fmt.Errorf("claiming account %s as %q: %w", state.AccountID, externalID, err)
	}
	fmt.Fprintln(r.out, ui.Success("Claimed as "+externalID))

	if specDir != "" {
		path, err := writeAccountSpec(ctx, specDir, externalID,
			firstNonEmpty(name, state.Name), firstNonEmpty(definition, state.AccountDefinitionName))
		switch {
		// The account exists and is claimed, so the command has succeeded
		// whatever the spec file does. Failing here would tell a script to
		// retry, and the retry would create a second account.
		case err != nil:
			fmt.Fprintln(r.out, ui.Warning(err.Error()))
			fmt.Fprintf(r.out, "   The spec was not written, but the account is created and claimed. Its fields are:\n"+
				"     id: %s\n     name: %s\n     account_definition_name: %s\n",
				externalID, firstNonEmpty(name, state.Name), firstNonEmpty(definition, state.AccountDefinitionName))
		default:
			fmt.Fprintf(r.out, "   Wrote %s\n", path)
		}
	}

	fmt.Fprintf(r.out, "\nReference it from a destination spec:\n   rudder_account_id: \"%s%s\"\n",
		accountRefPrefix, externalID)
	return nil
}

func (r *handoffRunner) reauthorize(ctx context.Context, ref string) error {
	account, err := r.resolveAccount(ctx, ref)
	if err != nil {
		return err
	}

	session, err := r.api.Open(ctx, oauthhandoff.OpenRequest{AccountID: account.ID})
	if err != nil {
		return err
	}

	fmt.Fprintf(r.out, "\nRe-authorizing %q (%s).\n", account.Name, account.ID)
	r.narrate(session)

	if _, err := oauthhandoff.Wait(ctx, r.api, session.ID, r.waitOptions()); err != nil {
		return r.explainWaitFailure(err,
			"  The link stays valid until it expires. Finishing in the browser re-authorizes\n"+
				"  the account, whether or not this command is still watching.")
	}

	fmt.Fprintln(r.out, ui.Success("Re-authorized "+account.ID))
	fmt.Fprint(r.out, "\nThe account id is unchanged, so no spec and no destination needs editing.\n")
	return nil
}

// resolveAccount turns a spec reference or external id into the remote account
// the project adopted.
//
// The adopt path writes the spec id onto the remote account as its externalId,
// so the remote list is the lookup: it needs no project directory and no state
// fetch. A bare argument that matches no external id is tried as a remote
// account id, because that is what a user who copied it from the dashboard has.
func (r *handoffRunner) resolveAccount(ctx context.Context, ref string) (*client.Account, error) {
	external, isRef := strings.CutPrefix(strings.TrimSpace(ref), accountRefPrefix)
	if external == "" {
		return nil, fmt.Errorf("%q names no account", ref)
	}

	remote, err := r.accounts.Accounts.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	if i := slices.IndexFunc(remote, func(a client.Account) bool { return a.ExternalID == external }); i >= 0 {
		return requireOAuthAccount(&remote[i])
	}

	if isRef {
		return nil, fmt.Errorf(`no account is adopted as %q in this workspace.
  'rudder-cli workspace accounts list' shows what is adopted, and 'rudder-cli apply' adopts an account a spec names`, external)
	}

	account, err := r.accounts.Accounts.Get(ctx, external)
	if err != nil {
		return nil, fmt.Errorf("no account is adopted as %q, and no account has that id: %w", external, err)
	}
	return requireOAuthAccount(account)
}

// requireOAuthAccount refuses an account with no OAuth grant to re-authorize.
func requireOAuthAccount(account *client.Account) (*client.Account, error) {
	if !accounts.IsOAuthDefinition(account.Definition.Name) {
		return nil, fmt.Errorf("account %s is a %s account, which holds its own credentials: there is no authorization to renew.\n"+
			"  Edit its spec and run 'rudder-cli apply' instead", account.ID, account.Definition.Name)
	}
	return account, nil
}

// narrate prints the link, opens a browser when there is one, and says what the
// wait is. A failed launch is not a failed command: the link is already printed.
func (r *handoffRunner) narrate(session *oauthhandoff.Session) {
	fmt.Fprintf(r.out, "Sign in and consent here:\n\n  %s\n\n", session.WebURL)

	switch {
	case !r.launchBrowser:
		fmt.Fprintf(r.out, "Open that link in a browser on any machine signed in to RudderStack.\n")
	case oauthhandoff.OpenBrowser(session.WebURL) != nil:
		fmt.Fprintf(r.out, "Could not open a browser. Open the link above.\n")
	default:
		fmt.Fprintf(r.out, "Opened your browser. ")
	}

	fmt.Fprintf(r.out, "Waiting for you to finish, up to %s.\n", r.timeout)
	fmt.Fprintf(r.out, "Press Ctrl-C to cancel; the link stays valid until it expires.\n\n")
}

func (r *handoffRunner) waitOptions() oauthhandoff.WaitOptions {
	opts := oauthhandoff.WaitOptions{Timeout: r.timeout}
	if r.paints() {
		opts.Tick = r.paintWait
	}
	return opts
}

// paints reports whether the elapsed counter has a terminal to repaint over. A
// repainting line is noise in captured or piped output, and the counter is all
// it adds, so anything other than a terminal on stdout gets none.
func (r *handoffRunner) paints() bool {
	return r.out == os.Stdout && ui.IsTerminal()
}

var spinnerFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

func (r *handoffRunner) paintWait(elapsed time.Duration) {
	frame := spinnerFrames[int(elapsed/(120*time.Millisecond))%len(spinnerFrames)]
	fmt.Fprintf(r.out, "\r%c waiting for authorization ... %ds", frame, int(elapsed.Seconds()))
}

// explainWaitFailure turns a failed wait into the next thing to do, rather than
// just the error.
//
// The recovery differs by command, so each caller supplies it: create can pick
// the session up with --resume, while re-authorize has no such flag and needs
// none. Finishing in the browser re-authorizes the account whether or not the
// CLI was still watching, and there is nothing left for it to claim.
func (r *handoffRunner) explainWaitFailure(err error, recovery string) error {
	if r.paints() {
		fmt.Fprintf(r.out, "\n")
	}

	var denied *oauthhandoff.DeniedError
	switch {
	case errors.Is(err, oauthhandoff.ErrTimedOut):
		return fmt.Errorf("timed out after %s waiting for authorization.\n%s", r.timeout, recovery)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("cancelled. The authorization session is untouched.\n%s", recovery)
	case errors.As(err, &denied):
		return fmt.Errorf("%w\n"+
			"  Ask a workspace admin to create the account, then adopt it with 'rudder-cli import workspace'", err)
	default:
		return err
	}
}

// checkSpecTarget reports an unwritable --write-spec target. create calls it
// before the handoff session is opened, so the common case (a second run with
// the same --name) is refused while nothing has been created yet.
//
// An empty externalID means --resume, whose name comes from the session: there
// is nothing to check yet, and the write failure that follows is a warning
// rather than a failure, because by then the account exists.
func checkSpecTarget(dir, externalID string) error {
	if dir == "" || externalID == "" {
		return nil
	}

	path := filepath.Join(dir, externalID+".yaml")
	switch _, err := os.Stat(path); {
	case err == nil:
		return fmt.Errorf("%s already exists, and the spec is never overwritten.\n"+
			"  Pick another --name, drop --write-spec, or move that file aside", path)
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("checking %s: %w", path, err)
	}
	return nil
}

// writeAccountSpec renders the 'kind: account' spec create and adopt share. An
// OAuth account carries no config block: its definition declares no option
// fields and its credentials are not the CLI's to hold.
func writeAccountSpec(ctx context.Context, dir, externalID, name, definition string) (string, error) {
	spec := &specs.Spec{
		Version:  specs.SpecVersionV1,
		Kind:     accounts.AccountSpecKind,
		Metadata: map[string]any{"name": externalID},
		Spec: map[string]any{
			"id":                      externalID,
			"name":                    name,
			"account_definition_name": definition,
		},
	}

	relative := externalID + ".yaml"
	entity := writer.FormattableEntity{Content: spec, RelativePath: relative}
	// writer.Write refuses to overwrite, so an existing spec is reported rather
	// than replaced.
	path := filepath.Join(dir, relative)
	// writer.Write wraps os.OpenFile's own message twice, so the raw error
	// names the path three times. Only the exists case is worth reporting in
	// its own words; anything else keeps the wrapped detail.
	if err := writer.Write(ctx, dir, formatter.Setup(formatter.DefaultYAML), []writer.FormattableEntity{entity}); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists, and the spec is never overwritten", path)
		}
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}

// specID turns a display name into the URN-safe spec id the account is claimed
// under, so a user never has to invent one.
func specID(name string) string {
	id := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, name)

	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}
	return strings.Trim(id, "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
