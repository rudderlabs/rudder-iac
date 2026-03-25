package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/display"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var ErrTestsFailed = errors.New("one or more tests failed")

var (
	testLog = logger.New("transformations", logger.Attr{
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
		output   string
		force    bool
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

			# Test from a specific project directory
			$ rudder-cli transformations test --all -l ./my-project

			# Write results to a custom file path
			$ rudder-cli transformations test --all -o /tmp/results.json

			# Overwrite an existing results file
			$ rudder-cli transformations test --all --force
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate flags first
			if err := validateFlags(args, all, modified, output, force); err != nil {
				return err
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
				}...)
			}()

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

			spinner := ui.NewSpinner("Running tests...")
			spinner.Start()

			runner := testorchestrator.NewRunner(deps.Client(), trProvider, graph, workspace.ID)
			results, err := runner.Run(ctx, mode, targetID)

			spinner.Stop()

			if err != nil {
				return fmt.Errorf("running tests: %w", err)
			}

			outputPath := output
			if outputPath == "" {
				outputPath = "test-results.json"
			}

			if err = writeResultsFile(outputPath, results); err != nil {
				return fmt.Errorf("writing results file: %w", err)
			}

			if results.Status == testorchestrator.RunStatusNoResources {
				ui.Println("No resources to test")
				return nil
			}

			displayer := display.NewResultDisplayer(verbose)
			displayer.Display(results)

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
	cmd.Flags().StringVarP(&output, "output", "o", "", "Path to write test results JSON file (default: test-results.json)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite output file if it already exists")

	return cmd
}

// validateFlags validates the command flags and arguments
func validateFlags(args []string, all, modified bool, output string, force bool) error {
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

	outputPath := output
	if outputPath == "" {
		outputPath = "test-results.json"
	}

	dir := filepath.Dir(outputPath)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", dir)
	}
	if err == nil && !info.IsDir() {
		return fmt.Errorf("output path is not valid: %s is not a directory", dir)
	}

	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath)
		}
	}

	return nil
}

func writeResultsFile(path string, results *testorchestrator.TestResults) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating results file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		return fmt.Errorf("encoding results: %w", err)
	}

	return nil
}
