package list

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

// TrackingPlanLister handles listing tracking plans
type TrackingPlanLister struct {
	catalog catalog.DataCatalog
}

// NewTrackingPlanLister creates a new TrackingPlanLister
func NewTrackingPlanLister(catalog catalog.DataCatalog) *TrackingPlanLister {
	return &TrackingPlanLister{catalog: catalog}
}

// List retrieves all tracking plans
func (l *TrackingPlanLister) List(ctx context.Context) ([]catalog.TrackingPlan, error) {
	return l.catalog.ListTrackingPlans(ctx)
}

// DisplayTable displays tracking plans in a simple table format
func (l *TrackingPlanLister) DisplayTable(ctx context.Context) error {
	plans, err := l.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tracking plans: %w", err)
	}

	if len(plans) == 0 {
		fmt.Println("No tracking plans found")
		return nil
	}

	// Simple table output
	fmt.Printf("%-20s %-30s %-8s %-50s %s\n", "ID", "NAME", "VERSION", "DESCRIPTION", "CREATED")
	fmt.Println(strings.Repeat("-", 120))

	for _, plan := range plans {
		description := ""
		if plan.Description != nil {
			description = *plan.Description
			// Truncate long descriptions
			if len(description) > 47 {
				description = description[:44] + "..."
			}
		}

		created := plan.CreatedAt.Format("2006-01-02")

		fmt.Printf("%-20s %-30s %-8d %-50s %s\n",
			plan.ID,
			plan.Name,
			plan.Version,
			description,
			created,
		)
	}

	return nil
}

// DisplayJSON displays tracking plans in JSON format
func (l *TrackingPlanLister) DisplayJSON(ctx context.Context) error {
	plans, err := l.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tracking plans: %w", err)
	}

	for _, plan := range plans {
		b, err := json.Marshal(plan)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

// NewCmdTPList creates a new cobra command for listing tracking plans
func NewCmdTPList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracking plans in workspace",
		Long: heredoc.Doc(`
			List all tracking plans in the current workspace.

			This command displays tracking plans with their basic information including
			ID, name, version, description, and creation date.
		`),
		Example: heredoc.Doc(`
			# List tracking plans in table format
			$ rudder-cli tp list

			# List tracking plans in JSON format
			$ rudder-cli tp list --json
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")

			var err error
			defer func() {
				telemetry.TrackCommand("tp list", err, []telemetry.KV{
					{K: "json", V: jsonOutput},
				}...)
			}()

			deps, err := app.NewDeps()
			if err != nil {
				return err
			}

			lister := NewTrackingPlanLister(catalog.NewRudderDataCatalog(deps.Client()))

			if jsonOutput {
				return lister.DisplayJSON(cmd.Context())
			}

			// Show spinner for table format
			spinner := ui.NewSpinner("Fetching tracking plans...")
			spinner.Start()
			err = lister.DisplayTable(cmd.Context())
			spinner.Stop()

			return err
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}