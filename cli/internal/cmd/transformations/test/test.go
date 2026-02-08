package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var ErrTestsFailed = errors.New("one or more tests failed")

var (
	testLog = logger.New("transformations-test", logger.Attr{
		Key:   "cmd",
		Value: "test",
	})
)

func NewCmdTest() *cobra.Command {
	var (
		deps     app.Deps
		p        project.Project
		err      error
		location string
		all      bool
		modified bool
		verbose  bool
		show     bool
	)

	cmd := &cobra.Command{
		Use:   "test [id]",
		Short: "Test transformations",
		Long: heredoc.Doc(`
			Tests transformations by executing their code against test input events.

			You can test a single transformation by ID, all transformations, or only
			modified transformations. Test results show pass/fail status with optional
			verbose output showing diffs for failures.
		`),
		Example: heredoc.Doc(`
			# Test a single transformation
			$ rudder-cli transformations test my-transformation-id

			# Test all transformations
			$ rudder-cli transformations test --all

			# Test only modified transformations
			$ rudder-cli transformations test --modified

			# Test with verbose output (shows diffs)
			$ rudder-cli transformations test --all --verbose

			# Show default test events
			$ rudder-cli transformations test default-events --show
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags first
			if err := validateFlags(args, all, modified, show); err != nil {
				return err
			}

			// If showing default events, skip project initialization
			if show {
				return nil
			}

			// Initialize dependencies
			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			// Create project
			p = deps.NewProject()

			// Load and validate the project configuration
			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("transformations test", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "all", V: all},
					{K: "modified", V: modified},
					{K: "verbose", V: verbose},
					{K: "show", V: show},
				}...)
			}()

			// Handle show default events
			if show {
				return showDefaultEvents()
			}

			testLog.Debug("test", "location", location, "all", all, "modified", modified, "verbose", verbose)

			ctx := context.Background()
			// Get workspace information
			workspace, err := deps.Client().Workspaces.GetByAuthToken(ctx)
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			// Get resource graph
			graph, err := p.ResourceGraph()
			if err != nil {
				return fmt.Errorf("getting resource graph: %w", err)
			}

			// Create transformations provider
			trProvider := deps.Providers().Transformations

			// Determine test mode and target
			var mode testorchestrator.Mode
			var targetID string

			if all {
				mode = testorchestrator.ModeAll
			} else if modified {
				mode = testorchestrator.ModeModified
			} else {
				mode = testorchestrator.ModeSingle
				targetID = args[0]
			}

			runner := testorchestrator.NewRunner(deps, trProvider, graph, workspace.ID)
			results, err := runner.Run(ctx, mode, targetID)
			if err != nil {
				return fmt.Errorf("running tests: %w", err)
			}

			// TODO: Format and display results
			// formatter := testorchestrator.NewFormatter(verbose)
			// formatter.Display(results)

			if results.HasFailures() {
				return ErrTestsFailed
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmd.Flags().BoolVar(&all, "all", false, "Test all transformations in the project")
	cmd.Flags().BoolVar(&modified, "modified", false, "Test only new or modified transformations")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed output including diffs for failures")
	cmd.Flags().BoolVar(&show, "show", false, "Show default test events (use with 'default-events' argument)")

	return cmd
}

// validateFlags validates the command flags and arguments
func validateFlags(args []string, all, modified, show bool) error {
	// Special case: show flag requires "default-events" argument
	if show {
		if len(args) == 0 || args[0] != "default-events" {
			return fmt.Errorf("--show flag requires 'default-events' argument")
		}
		return nil
	}

	// Count active modes
	modes := 0
	hasID := len(args) > 0
	if hasID {
		modes++
	}
	if all {
		modes++
	}
	if modified {
		modes++
	}

	// Must have exactly one mode
	if modes == 0 {
		return fmt.Errorf("must specify either an ID, --all, or --modified")
	}
	if modes > 1 {
		return fmt.Errorf("cannot combine test modes: specify only one of ID, --all, or --modified")
	}

	// Only one ID allowed
	if len(args) > 1 {
		return fmt.Errorf("only one transformation/library ID allowed, got %d arguments", len(args))
	}

	return nil
}

// showDefaultEvents displays the embedded default test events
func showDefaultEvents() error {
	events := testorchestrator.GetDefaultEvents()

	ui.Println(ui.Bold("Default Test Events: \n"))

	for eventName, eventData := range events {
		ui.Printf("--- %s ---\n", ui.Color(eventName, ui.ColorYellow))

		// Marshal event data to pretty JSON for display
		jsonBytes, err := json.MarshalIndent(eventData, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling event %s to JSON: %w", eventName, err)
		}

		ui.Println(string(jsonBytes))
		ui.Println()
	}

	return nil
}
