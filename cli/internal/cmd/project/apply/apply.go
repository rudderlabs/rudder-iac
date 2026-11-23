package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/spf13/cobra"
)

var (
	applyLog = logger.New("root", logger.Attr{
		Key:   "cmd",
		Value: "apply",
	})
)

func NewCmdApply() *cobra.Command {
	var (
		deps       app.Deps
		p          project.Project
		workspace  *client.Workspace
		location   string
		dryRun     bool
		confirm    bool
		varFiles   []string
		jsonOutput bool
		// Diagnostics are buffered rather than written straight out: on a clean
		// load apply's document is the plan, and an empty diagnostics document
		// ahead of it would mean two JSON values on stdout and nothing that
		// parses. Only a failed load promotes the buffer to the output.
		diagnostics bytes.Buffer
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply project configuration changes",
		Long: heredoc.Doc(`
			Applies the project configuration changes to the RudderStack workspace associated with your access token.
			This includes creating, updating, or deleting resources based on
			the differences between local configuration and the workspace resources.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli apply --location </path/to/dir or file>
			$ rudder-cli apply --location </path/to/dir or file> --dry-run
			$ rudder-cli apply --location </path/to/dir or file> --confirm=false
			$ rudder-cli apply --dry-run --json
		`),
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			// The JSON reporter always declines a confirmation, so without one of
			// these the command would plan, apply nothing and exit 0.
			if jsonOutput && !dryRun && confirm {
				return errors.New("--json needs --dry-run to preview the plan, or --confirm=false to apply it")
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			workspace, err = deps.Client().Workspaces.GetByAuthToken(context.Background())
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			// stdout holds only the JSON document, so the banner goes to stderr.
			if jsonOutput {
				fmt.Fprintln(cmd.ErrOrStderr(), workspaceBanner(workspace))
			} else {
				ui.Printf("%s\n", workspaceBanner(workspace))
			}

			projectOpts, err := app.NewProjectOptions(varFiles)
			if err != nil {
				return err
			}
			projectOpts = append(projectOpts, project.WithWorkspaceID(workspace.ID))

			// A validation failure during load would otherwise render text
			// diagnostics into the middle of the JSON document.
			if jsonOutput {
				projectOpts = append(projectOpts, project.WithRenderer(
					renderer.NewJSONRenderer(&diagnostics),
				))
			}

			p = deps.NewProject(projectOpts...)

			// Load and validate the project configuration
			if err := p.Load(location); err != nil {
				wrapped := fmt.Errorf("loading and validating project: %w", err)
				if jsonOutput {
					if _, copyErr := diagnostics.WriteTo(cmd.OutOrStdout()); copyErr != nil {
						applyLog.Error("writing json diagnostics", "error", copyErr)
					}
					// The document already holds the failure, so stderr does not
					// repeat it. What the document lacks, such as the --var-file
					// hint, keeps its message.
					if errors.Is(err, project.ErrReported) {
						return &cmderrors.SilentError{Err: wrapped}
					}
				}
				return wrapped
			}

			if jsonOutput {
				// A clean load leaves the plan as the only document on stdout, so
				// warnings and the legacy notice go to stderr, as validate does.
				if hasWarnings(diagnostics.Bytes()) {
					if _, copyErr := diagnostics.WriteTo(cmd.ErrOrStderr()); copyErr != nil {
						applyLog.Error("writing json warnings", "error", copyErr)
					}
				}
				if project.HasLegacySpecs(p.Specs()) {
					fmt.Fprintln(cmd.ErrOrStderr(), ui.Warning(project.LegacySpecDeprecationWarning))
				}
			} else if project.HasLegacySpecs(p.Specs()) {
				ui.PrintDeprecationWarning(project.LegacySpecDeprecationWarning)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyLog.Debug("apply", "location", location, "dryRun", dryRun, "confirm", confirm, "json", jsonOutput)
			applyLog.Debug("identifying changes for the upstream catalog")

			defer func() {
				telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
					{K: "location", V: location},
					{K: "dryRun", V: dryRun},
					{K: "confirm", V: confirm},
					{K: "json", V: jsonOutput},
				}...)
			}()

			// Get resource graph to understand dependencies
			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			reporter := app.SyncReporter()
			if jsonOutput {
				jsonReporter := reporters.NewJSONSyncReporter(cmd.OutOrStdout(), dryRun)
				// err is the named return, so the document records a failure.
				defer func() {
					// A document that could not be written is a failure: exiting 0
					// would leave the caller with none, or a truncated one.
					if flushErr := jsonReporter.Flush(err); flushErr != nil {
						applyLog.Error("flushing json report", "error", flushErr)
						err = errors.Join(err, flushErr)
					}
				}()
				reporter = jsonReporter
			}

			options := []syncer.Option{
				syncer.WithDryRun(dryRun),
				syncer.WithAskConfirmation(confirm),
				syncer.WithReporter(reporter),
				syncer.WithConcurrency(config.GetConfig().Concurrency.Syncer),
			}

			// Create syncer to handle the changes
			s, err := syncer.New(deps.CompositeProvider(), workspace, options...)
			if err != nil {
				return err
			}

			// Apply the changes
			err = s.Sync(context.Background(), graph)
			if err != nil {
				return fmt.Errorf("syncing resources: %w", err)
			}

			if dryRun {
				applyLog.Info("Dry run completed. No changes were applied.")
			} else {
				applyLog.Info("Successfully applied all changes")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only show the changes without applying them")
	cmd.Flags().BoolVar(&confirm, "confirm", true, "Confirm changes before applying them")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output the plan and results as JSON (needs --dry-run or --confirm=false)")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "Path to a variable file ending in .vars.yaml or .vars.yml (repeatable; later files take priority)")

	return cmd
}

// workspaceBanner names the workspace the apply is about to touch, so a run
// against the wrong token is visible before any change is planned.
func workspaceBanner(workspace *client.Workspace) string {
	return fmt.Sprintf(
		"%s %s %s",
		ui.Color("Workspace:", ui.ColorBlue),
		ui.Bold(workspace.Name),
		ui.GreyedOut("("+workspace.ID+")"),
	)
}

// hasWarnings reports whether the JSON diagnostics document holds any warning.
func hasWarnings(document []byte) bool {
	var doc struct {
		Summary struct {
			Warnings int `json:"warnings"`
		} `json:"summary"`
	}
	return json.Unmarshal(document, &doc) == nil && doc.Summary.Warnings > 0
}
