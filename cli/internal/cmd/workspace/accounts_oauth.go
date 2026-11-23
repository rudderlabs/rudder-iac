package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/oauthhandoff"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// accountRefPrefix is the reference form a destination spec uses for its
// integration account, which is also what reauthorize takes as its argument so
// a user never has to look up the opaque remote id.
const accountRefPrefix = "#" + accounts.AccountSpecKind + ":"

const defaultHandoffTimeout = 5 * time.Minute

// maxSpecBytes caps how much of a file in the --write-spec directory is read
// while looking for ids already in use. A spec is a few hundred bytes; the cap
// is here so a huge or endless file in that directory cannot stall the command.
const maxSpecBytes = 1 << 20

// experimentalNotice is on both commands' help, because neither can work until
// the control plane grows the handoff API that docs/oauth-account-handoff.md
// specifies. It is not a flag that merely hides a finished feature.
const experimentalNotice = `This command is EXPERIMENTAL and cannot work against production yet: the
browser handoff API it calls is not released, and the consent page is behind
the webapp's own 'enable-cli-account-handoff' flag. See
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

// adoptHint is what to run to bring a new account into a project. It is the
// ordinary import path, and it is what create prints whenever it has not
// written a spec itself.
const adoptHint = `
Nothing manages this account yet. To bring it into a project:
   rudder-cli import workspace   writes its spec
   rudder-cli apply              adopts it, claiming the spec id on the account
`

// createHints is what create says when the wait ends badly.
//
// None of them names a session id, because the recovery never needs one:
// consent either created an account, which 'import workspace' finds because
// nothing has claimed it, or it did not, and the command can simply be run
// again.
var createHints = waitHints{
	unresolved: "  If you finished in the browser, the account exists and nothing manages it: 'rudder-cli import workspace'\n" +
		"  writes its spec and 'rudder-cli apply' adopts it. If you did not, run this command again.",
	denied: "Nothing was created. Run the command again to retry. If the page refused outright, this workspace may not\n" +
		"  have the CLI handoff enabled yet; if you cannot consent, ask a workspace admin to create the account in the\n" +
		"  dashboard and give the destination spec its remote account id.",
	expired: "  A consent that finished after the link expired still created an account. Check with\n" +
		"  'rudder-cli workspace accounts list', and adopt it with 'rudder-cli import workspace' if it is there.\n" +
		"  Otherwise run this command again.",
}

// reauthorizeHints is the same for reauthorize, which creates nothing: a
// consent that lands late re-authorizes the account anyway, so there is never
// anything left to adopt.
var reauthorizeHints = waitHints{
	unresolved: "  The link stays valid until it expires. Finishing in the browser re-authorizes\n" +
		"  the account, whether or not this command is still watching.",
	denied:  "The account is unchanged. Run the command again to retry, or ask a workspace admin to re-authorize it in the dashboard.",
	expired: "  The account is unchanged. Run the command again to get a fresh link.",
}

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
own dashboard session, and waits.

The account it creates belongs to no project until a spec names it, exactly as
an account made in the dashboard does. With --write-spec the command writes
that spec, pointing it at the new account, and the next 'rudder-cli apply'
adopts it. Without the flag, 'rudder-cli import workspace' writes the spec
instead.

` + experimentalNotice,
		Args:    cobra.NoArgs,
		Hidden:  true,
		PreRunE: requireOAuthAccounts,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			var (
				definition, _ = cmd.Flags().GetString("definition")
				name, _       = cmd.Flags().GetString("name")
				specDir, _    = cmd.Flags().GetString("write-spec")
				noBrowser, _  = cmd.Flags().GetBool("no-browser")
				timeout, _    = cmd.Flags().GetDuration("timeout")
			)

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "definition", V: definition},
					{K: "writeSpec", V: specDir != ""},
					{K: "noBrowser", V: noBrowser},
				}...)
			}()

			// The server trims the display name it stores, so the spec is
			// written from the trimmed one too rather than drifting from it.
			name = strings.TrimSpace(name)
			if err := validateCreateFlags(definition, name, timeout); err != nil {
				return err
			}

			runner, err := newHandoffRunner(cmd.OutOrStdout(), noBrowser, timeout)
			if err != nil {
				return err
			}

			// Ctrl-C cancels this context instead of killing the process, so
			// the wait ends through explainWaitFailure and says what consent
			// may have left behind.
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return runner.create(ctx, definition, name, specDir)
		},
	}

	cmd.Flags().String("definition", "", "Account definition name to create an account for")
	cmd.Flags().String("name", "", "Display name for the new account")
	cmd.Flags().String("write-spec", "", "Directory to write the 'kind: account' spec into")
	cmd.Flags().Bool("no-browser", false, "Print the authorization link instead of opening a browser")
	cmd.Flags().Duration("timeout", defaultHandoffTimeout, "How long to wait for the browser authorization")

	return cmd
}

func newCmdReauthorizeAccount() *cobra.Command {
	cmd := &cobra.Command{
		// Both forms are quoted: a shell reads a bare #account:id as the start
		// of a comment and passes nothing on.
		Use:   `reauthorize ("` + accountRefPrefix + `id" | "account-id")`,
		Short: "Re-authorize an OAuth integration account through the browser (experimental)",
		Long: `Re-authorize an OAuth integration account whose grant was revoked or expired.

Takes the account as its spec reference ("` + accountRefPrefix + `google-ads-prod") or as the
bare external id, and resolves it to the remote account the project adopted. A
remote account id works too, which is what a user who copied one out of the
dashboard has. The authorization runs in a browser, exactly as it does for
create, and the account's secret is replaced in place: the remote account id
does not change, so no spec and no destination needs editing.

Routine token refresh is server-side, so this is for a revoked or expired
grant, not for upkeep. An unattended 'apply' never needs it.

` + experimentalNotice,
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
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

			if timeout <= 0 {
				return errors.New("--timeout must be greater than zero")
			}

			runner, err := newHandoffRunner(cmd.OutOrStdout(), noBrowser, timeout)
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return runner.reauthorize(ctx, args[0])
		},
	}

	cmd.Flags().Bool("no-browser", false, "Print the authorization link instead of opening a browser")
	cmd.Flags().Duration("timeout", defaultHandoffTimeout, "How long to wait for the browser authorization")

	return cmd
}

// validateCreateFlags refuses a definition the CLI cannot create before any
// network call, naming the ones it can.
func validateCreateFlags(definition, name string, timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("--timeout must be greater than zero")
	}
	if definition == "" {
		return errors.New("--definition is required")
	}
	if name == "" {
		return errors.New("--name is required: it is the account's display name and the spec id is derived from it")
	}
	// Checked here because the same refusal after consent leaves an account
	// this run can no longer write a spec for.
	if baseExternalID(name) == "" {
		return fmt.Errorf("--name %q has no letters or digits to build a spec id from", name)
	}
	if !accounts.IsOAuthDefinition(definition) {
		return fmt.Errorf(`no OAuth account definition named %q.
  rudder-cli can create accounts for: %s
  A warehouse account (SOURCE_POSTGRES and the like) is not created through a browser: write a 'kind: account'
  spec and run 'rudder-cli apply'.
  linkedin_ads and bingads_offline_conversions accounts come from the legacy role-keyed OAuth flow and carry no
  account definition at all, so rudder-cli can neither create nor adopt one: create it in the dashboard and give
  the destination spec its remote account id`,
			definition, strings.Join(accounts.OAuthDefinitionNames(), ", "))
	}
	return nil
}

// handoffRunner is the CLI half of the browser handoff: it owns narration and
// the wait, and writes the spec that adopts what the consent created.
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

// specPlan is everything writing the spec needs that the consent itself does
// not supply. It is settled before the session opens: once an account exists, a
// refusal costs a consent the user has to give again, while the same refusal
// beforehand costs a flag correction.
type specPlan struct {
	externalID  string
	workspaceID string
	// takenBy names what the name's own id collided with, empty when it was
	// free, so a suffixed id can say why it is suffixed.
	takenBy string
}

func (r *handoffRunner) create(ctx context.Context, definition, name, specDir string) error {
	plan, err := r.planSpec(ctx, name, specDir)
	if err != nil {
		return err
	}

	session, err := r.api.Open(ctx, oauthhandoff.OpenRequest{
		AccountDefinitionName: definition, Name: name,
	})
	if err != nil {
		// A cancel here is a Ctrl-C before anything was opened, which is worth
		// saying plainly rather than reporting a cancelled HTTP request.
		if errors.Is(err, context.Canceled) {
			return errors.New("cancelled. Nothing was created")
		}
		return err
	}

	fmt.Fprintf(r.out, "\nAuthorizing %q (%s).\n", name, definition)
	r.narrate(session)

	state, err := oauthhandoff.Wait(ctx, r.api, session.ID, r.waitOptions())
	r.endPaint()
	if err != nil {
		return r.explainWaitFailure(err, createHints)
	}

	// The id is the server's string, so it goes through the sanitiser before
	// it reaches a terminal.
	fmt.Fprintln(r.out, ui.Success("Account created: "+oauthhandoff.Printable(state.AccountID)))

	if specDir == "" {
		fmt.Fprint(r.out, adoptHint)
		return nil
	}

	path, err := writeAccountSpec(ctx, specDir, plan, name, definition, state.AccountID)
	if err != nil {
		// The account exists and nothing is claimed, so the ordinary import
		// path still adopts it. Failing the command would tell a script to
		// retry, and the retry would run a second consent.
		fmt.Fprintln(r.out, ui.Warning(err.Error()))
		fmt.Fprint(r.out, adoptHint)
		return nil
	}

	fmt.Fprintf(r.out, "   Wrote %s\n", path)
	if plan.takenBy != "" {
		fmt.Fprintf(r.out, "   (%q is taken by %s, so the spec id is %q)\n",
			baseExternalID(name), plan.takenBy, plan.externalID)
	}
	fmt.Fprintf(r.out, `
Adopt it, which claims the id on the account:
   rudder-cli apply

Then reference it from a destination spec:
   rudder_account_id: "%s%s"
`, accountRefPrefix, plan.externalID)
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

	fmt.Fprintf(r.out, "\nRe-authorizing %q (%s).\n",
		oauthhandoff.Printable(account.Name), oauthhandoff.Printable(account.ID))
	r.narrate(session)

	_, err = oauthhandoff.Wait(ctx, r.api, session.ID, r.waitOptions())
	r.endPaint()
	if err != nil {
		return r.explainWaitFailure(err, reauthorizeHints)
	}

	fmt.Fprintln(r.out, ui.Success("Re-authorized "+oauthhandoff.Printable(account.ID)))
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
  'rudder-cli workspace accounts list' shows what the workspace has, and 'rudder-cli import workspace' writes a
  spec for an account no project manages yet`, external)
	}

	account, err := r.accounts.Accounts.Get(ctx, external)
	if err != nil {
		return nil, fmt.Errorf("no account is adopted as %q, and no account has that id: %w", external, err)
	}
	return requireOAuthAccount(account)
}

// requireOAuthAccount refuses an account with no OAuth grant to re-authorize.
//
// A definition the CLI does not know gets its own refusal: it may well be an
// OAuth definition, since integrations-config has several the CLI has no entry
// for, so saying it holds its own credentials would be a guess.
func requireOAuthAccount(account *client.Account) (*client.Account, error) {
	switch {
	case accounts.IsOAuthDefinition(account.Definition.Name):
		return account, nil
	case accounts.IsRegisteredDefinition(account.Definition.Name):
		return nil, fmt.Errorf("account %s is a %s account, which holds its own credentials: there is no authorization to renew.\n"+
			"  Edit its spec and run 'rudder-cli apply' instead", account.ID, account.Definition.Name)
	default:
		return nil, fmt.Errorf("rudder-cli does not know the account definition %q, so it cannot tell what re-authorizing account %s would do.\n"+
			"  Re-authorize it from the RudderStack dashboard",
			oauthhandoff.Printable(account.Definition.Name), account.ID)
	}
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
	// The wait is the caller's; the link's life is the server's. Saying so
	// beats a wait that outlives what it is waiting for.
	if until := time.Until(session.ExpiresAt); !session.ExpiresAt.IsZero() && until < r.timeout {
		fmt.Fprintf(r.out, "The link itself expires in %s, before that.\n", until.Round(time.Second))
	}
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

// endPaint closes the line the counter has been repainting, so whatever is
// printed next does not land on top of the last frame.
func (r *handoffRunner) endPaint() {
	if r.paints() {
		fmt.Fprintln(r.out)
	}
}

// waitHints is what a command says when its wait ends badly. create and
// reauthorize differ in what the consent left behind, so the text cannot be
// shared.
type waitHints struct {
	// unresolved covers a wait that ended with the session still open: a
	// timeout, a cancel, or a failed poll.
	unresolved string
	// denied is the browser half refusing.
	denied string
	// expired is the link outliving the wait.
	expired string
}

// explainWaitFailure turns a failed wait into the next thing to do, rather than
// just the error.
func (r *handoffRunner) explainWaitFailure(err error, hints waitHints) error {
	var (
		denied   *oauthhandoff.DeniedError
		apiError *client.APIError
	)
	switch {
	case errors.Is(err, oauthhandoff.ErrTimedOut):
		return fmt.Errorf("timed out after %s waiting for authorization.\n%s", r.timeout, hints.unresolved)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("cancelled.\n%s", hints.unresolved)
	case errors.As(err, &denied):
		return fmt.Errorf("%w\n  %s", err, hints.denied)
	case errors.Is(err, oauthhandoff.ErrExpired):
		// The session is gone, so there is nothing left to wait for: only what
		// the consent may already have done.
		return fmt.Errorf("%w\n%s", err, hints.expired)
	case errors.As(err, &apiError) && apiError.HTTPStatusCode == http.StatusNotFound:
		return fmt.Errorf("%w\n  This workspace has no authorization session with that id, so nothing was waiting", err)
	default:
		// A failed poll, such as a restart of the API mid-wait. The session is
		// still there, so the unresolved hint applies.
		return fmt.Errorf("%w\n%s", err, hints.unresolved)
	}
}

// planSpec settles the spec id, what it stepped around, and the workspace the
// spec records, all before the browser session opens.
//
// Nothing here runs without --write-spec: there is no file to write, so there
// is no id to pick and no directory to read.
func (r *handoffRunner) planSpec(ctx context.Context, name, specDir string) (specPlan, error) {
	var plan specPlan
	if specDir == "" {
		return plan, nil
	}
	if err := requireSpecDir(specDir); err != nil {
		return plan, err
	}

	local, err := localSpecIDs(specDir)
	if err != nil {
		return plan, err
	}
	claimed, err := r.claimedExternalIDs(ctx)
	if err != nil {
		return plan, err
	}

	// One source per id, the workspace first: an id a spec uses is usually one
	// the workspace has already claimed, and the account is the more useful
	// thing to name.
	taken := make(map[string]string, len(claimed)+len(local))
	for id := range claimed {
		taken[id] = "an account in this workspace"
	}
	for id, file := range local {
		if _, ok := taken[id]; !ok {
			taken[id] = file
		}
	}

	if plan.externalID, err = nameAround(taken, name); err != nil {
		return plan, err
	}
	if base := baseExternalID(name); plan.externalID != base {
		plan.takenBy = taken[base]
	}

	// The spec records which workspace the remote id belongs to, the same way
	// an imported spec does.
	workspace, err := r.accounts.Workspaces.GetByAuthToken(ctx)
	if err != nil {
		return plan, fmt.Errorf("reading this workspace's id for the spec: %w", err)
	}
	plan.workspaceID = workspace.ID

	return plan, nil
}

func (r *handoffRunner) claimedExternalIDs(ctx context.Context) (map[string]bool, error) {
	claimed, err := r.accounts.Accounts.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return nil, fmt.Errorf("listing the workspace's claimed accounts: %w", err)
	}

	ids := make(map[string]bool, len(claimed))
	for _, account := range claimed {
		if account.ExternalID != "" {
			ids[account.ExternalID] = true
		}
	}
	return ids, nil
}

// requireSpecDir refuses a --write-spec target that cannot hold a spec at all:
// a path that names a regular file, or one no directory can be made at.
//
// It says nothing about the spec's own name. An id another spec in the
// directory already uses is planSpec's business, which steps around it.
func requireSpecDir(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("--write-spec %s cannot hold the account spec: %w", dir, err)
	}
	return nil
}

// localSpecIDs is the ids already spoken for under dir, each mapped to the file
// that holds it.
//
// Two things count, because two different checks collide. An account spec's own
// 'spec.id' anywhere under dir is what a duplicate-resource check collides on,
// whatever the file is called. And the name of a YAML file directly in dir is
// what this command's own write collides on, since it writes <id>.yaml there. A
// YAML file in a subdirectory that is not an account spec counts for neither: a
// destination spec called google-ads.yaml says nothing about account ids.
func localSpecIDs(dir string) (map[string]string, error) {
	ids := map[string]string{}
	if dir == "" {
		return ids, nil
	}
	root := filepath.Clean(dir)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		// A FIFO or a device node with a .yaml name would block the read.
		if !entry.Type().IsRegular() || (ext != ".yaml" && ext != ".yml") {
			return nil
		}

		// A file named exactly ".yaml" has no base name. Recording the empty
		// string would hand it to the namer, which rejects it.
		if base := strings.TrimSuffix(entry.Name(), ext); base != "" && filepath.Dir(path) == root {
			ids[base] = path
		}
		if id := accountSpecID(path); id != "" {
			ids[id] = path
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return ids, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	return ids, nil
}

// accountSpecID is the id of the 'kind: account' spec in a file, or empty when
// the file is another kind or cannot be read as a spec. An unreadable file is
// left for the project loader to report: this only widens the set of taken ids.
func accountSpecID(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxSpecBytes))
	if err != nil {
		return ""
	}

	var doc struct {
		Kind string `yaml:"kind"`
		Spec struct {
			ID string `yaml:"id"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil || doc.Kind != accounts.AccountSpecKind {
		return ""
	}
	return doc.Spec.ID
}

// writeAccountSpec writes the 'kind: account' spec that adopts the account the
// consent just created.
//
// It is the spec 'import workspace' writes for an account nobody manages, down
// to the import metadata: the local id, and the remote account it names. That
// block is what makes the next 'apply' adopt the account instead of trying to
// create it, which the API refuses for an OAuth definition. An OAuth account
// carries no config block, because its definition declares no option fields and
// its credentials are not the CLI's to hold.
func writeAccountSpec(ctx context.Context, dir string, plan specPlan, name, definition, remoteID string) (string, error) {
	relative := plan.externalID + ".yaml"
	// The id is the namer's kebab-case output, so this cannot fire today. It
	// is here because writer.Write joins the name to the directory, and no
	// value should be able to choose the path it lands at.
	if !filepath.IsLocal(relative) {
		return "", fmt.Errorf("%q is not a file name this command will write", relative)
	}

	spec, err := specs.ToImportSpec(accounts.AccountSpecKind, accounts.AccountMetadataName,
		specs.WorkspaceImportMetadata{
			WorkspaceID: plan.workspaceID,
			Resources: []specs.ImportIds{{
				URN:      resources.URN(plan.externalID, accounts.AccountResourceType),
				RemoteID: remoteID,
			}},
		},
		map[string]any{
			"id":                      plan.externalID,
			"name":                    name,
			"account_definition_name": definition,
		})
	if err != nil {
		return "", fmt.Errorf("building the account spec: %w", err)
	}

	entity := writer.FormattableEntity{Content: spec, RelativePath: relative}
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

// baseExternalID is the spec id a display name maps to before any collision is
// resolved. It is the import flow's convention (namer's kebab-case strategy),
// so an account created here and one adopted by 'import workspace' get the same
// id for the same name.
func baseExternalID(name string) string {
	return namer.StrategyKebabCase.Name(name)
}

// nameAround is baseExternalID disambiguated against every id that is already
// spoken for, so a display name whose id is taken gets "-1" appended instead of
// a spec that collides with another resource or another file.
//
// Both sides count: an id the workspace has claimed, and an id a spec in the
// --write-spec directory uses. One rule, so --write-spec changes which file is
// written and never whether the command runs.
//
// A free name is kept unchanged: the namer only resolves a collision.
func nameAround(taken map[string]string, name string) (string, error) {
	scope := make([]namer.ScopeName, 0, len(taken))
	// Sorted, so the suffix a collision gets does not depend on map order.
	for _, id := range slices.Sorted(maps.Keys(taken)) {
		scope = append(scope, namer.ScopeName{Scope: accounts.AccountResourceType, Name: id})
	}

	idNamer := namer.NewExternalIdNamer(namer.StrategyKebabCase)
	if err := idNamer.Load(scope); err != nil {
		return "", fmt.Errorf("loading the ids already in use: %w", err)
	}
	return idNamer.Name(namer.ScopeName{Scope: accounts.AccountResourceType, Name: name})
}
