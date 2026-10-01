package initcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	essource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var log = logger.New("init")

// cliClientEnv names whatever drove the run so telemetry can attribute it.
const cliClientEnv = "RUDDERSTACK_CLI_CLIENT"

// Options are the wizard's answers as flags, so an agent can run init without a
// terminal. Every question has one.
type Options struct {
	Location    string
	Flow        string
	SourceType  string
	Destination string
	WebhookURL  string
	Yes         bool
}

func NewCmdInit() *cobra.Command {
	var opts Options

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set up event delivery from scratch and send your first event",
		Long: heredoc.Doc(`
			Ask three questions, write the project that answers them, apply it, and
			commit the result, ending with the one command that sends your own event
			through the pipeline you just created.

			What it creates is one event stream source, one destination, and the
			connection between them. The specs are written as YAML into the target
			directory and applied to the workspace your access token belongs to.

			Every question has a flag, so an agent can answer without prompting. With
			--yes, init takes the defaults for everything it can and fails rather than
			prompting for an answer that has no default.

			There is no revert command: the generated specs are committed to git, so
			the way back is 'git revert' followed by 'rudder-cli apply'.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli init
			$ rudder-cli init --location ./my-app
			$ rudder-cli init --source-type javascript --webhook-url https://example.com/hook --yes
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("init", err, []telemetry.KV{
					{K: "location", V: opts.Location},
					{K: "flow", V: opts.Flow},
					{K: "sourceType", V: opts.SourceType},
					{K: "destination", V: opts.Destination},
					{K: "yes", V: opts.Yes},
				}...)
			}()

			err = run(cmd.Context(), opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.Location, "location", "l", ".", "Directory to write the project into")
	cmd.Flags().StringVar(&opts.Flow, "flow", FlowEventStream, "Which pipeline to set up")
	cmd.Flags().StringVar(&opts.SourceType, "source-type", "", "What is sending events (skips the question; inferred from package.json when present)")
	cmd.Flags().StringVar(&opts.Destination, "destination", "", "Where the data should go")
	cmd.Flags().StringVar(&opts.WebhookURL, "webhook-url", "", "URL the events should be delivered to")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Take every default and never prompt; fail when a required answer has no default")

	return cmd
}

func run(ctx context.Context, opts Options) error {
	if opts.Flow != FlowEventStream {
		return fmt.Errorf("unknown flow %q, only %q is supported", opts.Flow, FlowEventStream)
	}

	// Telemetry attributes the run to whatever drove it: a human shell, an agent,
	// CI. The env var is how that caller says so.
	log.Debug("init", "location", opts.Location, "client", os.Getenv(cliClientEnv))

	// Checked before anything is asked or fetched, so re-running init in a
	// directory it already owns costs nothing and fails immediately.
	names := NamesFor(opts.Location)
	if err := EnsureNotInitialised(opts.Location, names); err != nil {
		return err
	}

	answers, err := Resolve(opts.Location, opts, DefaultPrompter())
	if err != nil {
		return err
	}

	deps, err := app.NewDeps()
	if err != nil {
		return fmt.Errorf("initialising dependencies: %w", err)
	}

	workspace, err := deps.Client().Workspaces.GetByAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("fetching workspace information: %w", err)
	}
	ui.Printf("%s %s %s\n",
		ui.Color("Workspace:", ui.ColorBlue),
		ui.Bold(workspace.Name),
		ui.GreyedOut("("+workspace.ID+")"),
	)

	paths, err := Generate(ctx, opts.Location, names, answers)
	if err != nil {
		return err
	}

	// The specs have to exist on disk before they can be planned, so a failure
	// from here on leaves them behind. Saying so turns a confusing "file already
	// exists" on the next init into a resumable step.
	resumable := func(err error) error {
		ui.PrintInfo(fmt.Sprintf("The specs are already written to %s. After fixing the above, resume with 'rudder-cli apply -l %s'.", opts.Location, opts.Location))
		return err
	}

	graph, err := loadGraph(ctx, deps, workspace, opts.Location)
	if err != nil {
		return resumable(err)
	}

	plan, err := dryRun(ctx, deps.CompositeProvider(), workspace, graph)
	if err != nil {
		return resumable(err)
	}

	if err := GuardDeletions(plan, ui.IsTerminal(), ui.Confirm); err != nil {
		return resumable(err)
	}

	if err := apply(ctx, deps.CompositeProvider(), workspace, graph); err != nil {
		return resumable(err)
	}

	commit(opts.Location, paths)

	return report(ctx, deps, workspace, names, answers, opts.Location, paths)
}

func loadGraph(ctx context.Context, deps app.Deps, workspace *client.Workspace, location string) (*resources.Graph, error) {
	// ponytail: no --var-file. The generated project substitutes nothing, so
	// there is no vars file to pass; an existing project in the directory that
	// needs one is out of scope for a first-run wizard.
	projectOpts, err := app.NewProjectOptions(nil)
	if err != nil {
		return nil, err
	}
	projectOpts = append(projectOpts, project.WithWorkspaceID(workspace.ID))

	p := deps.NewProject(projectOpts...)
	if err := p.Load(location); err != nil {
		return nil, fmt.Errorf("loading and validating project: %w", err)
	}

	graph, err := p.ResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("getting resource graph: %w", err)
	}
	return graph, nil
}

// planCapture keeps the plan the syncer reports. Sync does not return it, and
// the deletion guard needs to read it, so the reporter is the seam.
type planCapture struct {
	syncer.SyncReporter
	plan *planner.Plan
}

func (c *planCapture) ReportPlan(plan *planner.Plan) {
	c.plan = plan
	c.SyncReporter.ReportPlan(plan)
}

// dryRun runs the full remote load, map and plan, and stops before any
// mutation (syncer.go:144), which is what makes it safe to run first purely to
// read the plan.
func dryRun(ctx context.Context, prov syncer.SyncProvider, workspace *client.Workspace, graph *resources.Graph) (*planner.Plan, error) {
	capture := &planCapture{SyncReporter: app.SyncReporter()}

	s, err := syncer.New(prov, workspace,
		syncer.WithDryRun(true),
		syncer.WithAskConfirmation(false),
		syncer.WithReporter(capture),
		syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer),
	)
	if err != nil {
		return nil, err
	}

	if err := s.Sync(ctx, graph); err != nil {
		return nil, fmt.Errorf("planning changes: %w", err)
	}
	return capture.plan, nil
}

// apply owns its confirmation rather than letting the syncer ask. The syncer's
// own prompt auto-declines without a terminal and then returns nil, which would
// make a non-interactive run apply nothing and still report success.
func apply(ctx context.Context, prov syncer.SyncProvider, workspace *client.Workspace, graph *resources.Graph) error {
	s, err := syncer.New(prov, workspace,
		syncer.WithDryRun(false),
		syncer.WithAskConfirmation(false),
		syncer.WithReporter(app.SyncReporter()),
		syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer),
	)
	if err != nil {
		return err
	}

	if err := s.Sync(ctx, graph); err != nil {
		return fmt.Errorf("syncing resources: %w", err)
	}
	return nil
}

// commit puts the generated specs under version control, which is what stands
// in for a revert command: undoing an init is 'git revert' then 'apply'. A
// missing git is a warning rather than a failed run, since the resources
// already exist.
func commit(location string, paths []string) {
	if _, err := exec.LookPath("git"); err != nil {
		ui.PrintWarning("git not found, so the generated specs were not committed.")
		return
	}

	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = location
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v: %w: %s", args, err, out)
		}
		return nil
	}

	if err := git("rev-parse", "--git-dir"); err != nil {
		if err := git("init"); err != nil {
			ui.PrintWarning(fmt.Sprintf("could not initialise a git repository: %v", err))
			return
		}
	}

	add := []string{"add", "--"}
	for _, p := range paths {
		rel, err := filepath.Rel(location, p)
		if err != nil {
			rel = p
		}
		add = append(add, rel)
	}
	if err := git(add...); err != nil {
		ui.PrintWarning(fmt.Sprintf("could not stage the generated specs: %v", err))
		return
	}

	if err := git("commit", "-m", "Describe event delivery as code, so undoing it is a git revert"); err != nil {
		ui.PrintWarning(fmt.Sprintf("could not commit the generated specs: %v", err))
	}
}

func report(
	ctx context.Context,
	deps app.Deps,
	workspace *client.Workspace,
	names Names,
	answers Answers,
	location string,
	paths []string,
) error {
	ui.PrintSuccess("Event delivery is live.")

	ui.Printf("\n%s\n", ui.Bold("Created"))
	ui.Printf("  source       %s (%s)\n", names.Source, answers.SourceType)
	ui.Printf("  destination  %s -> %s\n", names.Destination, answers.WebhookURL)
	ui.Printf("  connection   %s\n", names.Connection)

	ui.Printf("\n%s\n", ui.Bold("Specs"))
	for _, p := range paths {
		ui.Printf("  %s\n", p)
	}

	dataPlaneURL := ""
	if workspace.DataPlaneURL != nil {
		dataPlaneURL = *workspace.DataPlaneURL
	}

	writeKey, err := writeKeyFor(ctx, deps, names.Source)
	if err != nil {
		// The pipeline exists either way, so a missing write key is reported
		// rather than failing the run, but it is reported loudly, because
		// nothing can be sent without it.
		ui.PrintWarning(fmt.Sprintf("could not read the source's write key: %v", err))
	}

	ui.Printf("\n%s\n", ui.Bold("Send your first event"))
	ui.Printf("  write key       %s\n", writeKey)
	ui.Printf("  data plane URL  %s\n", dataPlaneURL)
	ui.Printf("\n  curl -u %s: %s/v1/track \\\n", writeKey, dataPlaneURL)
	ui.Printf("    -H 'Content-Type: application/json' \\\n")
	ui.Printf("    -d '{\"userId\":\"demo\",\"event\":\"Signed Up\"}'\n")

	log.Info("init completed", "location", location, "source", names.Source)
	return nil
}

// writeKeyFor reads the applied source's write key.
//
// The write key is not in the applied state: the event stream source handler's
// output carries only the remote id (handler.go toResourceData), and the event
// stream source API type has no writeKey field at all. So the remote id is read
// back out of the freshly loaded state and exchanged for the key through
// /v2/sources/{id}, which does return it.
func writeKeyFor(ctx context.Context, deps app.Deps, localID string) (string, error) {
	remote, err := deps.CompositeProvider().LoadResourcesFromRemote(ctx)
	if err != nil {
		return "", fmt.Errorf("loading resources from remote: %w", err)
	}

	st, err := deps.CompositeProvider().MapRemoteToState(remote)
	if err != nil {
		return "", fmt.Errorf("mapping remote resources to state: %w", err)
	}

	rs := st.GetResource(resources.URN(localID, essource.ResourceType))
	if rs == nil {
		return "", fmt.Errorf("source %q is not in the applied state", localID)
	}

	remoteID, ok := rs.Output[essource.IDKey].(string)
	if !ok || remoteID == "" {
		return "", fmt.Errorf("source %q has no remote id in its state", localID)
	}

	source, err := deps.Client().Sources.Get(ctx, remoteID)
	if err != nil {
		return "", fmt.Errorf("fetching source %s: %w", remoteID, err)
	}
	if source == nil || source.WriteKey == "" {
		return "", fmt.Errorf("source %s returned no write key", remoteID)
	}

	return source.WriteKey, nil
}
