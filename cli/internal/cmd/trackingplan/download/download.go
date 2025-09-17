package download

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

// TrackingPlanDownloader handles downloading tracking plans and dependencies
type TrackingPlanDownloader struct {
	catalog   catalog.DataCatalog
	outputDir string
}

// NewTrackingPlanDownloader creates a new TrackingPlanDownloader
func NewTrackingPlanDownloader(catalog catalog.DataCatalog, outputDir string) *TrackingPlanDownloader {
	return &TrackingPlanDownloader{catalog: catalog, outputDir: outputDir}
}

// Download downloads all tracking plans and their dependencies to JSON files
func (d *TrackingPlanDownloader) Download(ctx context.Context) error {
	// Create output directory structure
	jsonDir := filepath.Join(d.outputDir, "json")
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Download tracking plans
	if err := d.downloadTrackingPlans(ctx, jsonDir); err != nil {
		return fmt.Errorf("failed to download tracking plans: %w", err)
	}

	// Download events
	if err := d.downloadEvents(ctx, jsonDir); err != nil {
		return fmt.Errorf("failed to download events: %w", err)
	}

	// Download properties
	if err := d.downloadProperties(ctx, jsonDir); err != nil {
		return fmt.Errorf("failed to download properties: %w", err)
	}

	// Download custom types
	if err := d.downloadCustomTypes(ctx, jsonDir); err != nil {
		return fmt.Errorf("failed to download custom types: %w", err)
	}

	// Download categories
	if err := d.downloadCategories(ctx, jsonDir); err != nil {
		return fmt.Errorf("failed to download categories: %w", err)
	}

	fmt.Printf("Successfully downloaded tracking plan data to %s\n", jsonDir)
	return nil
}

func (d *TrackingPlanDownloader) downloadTrackingPlans(ctx context.Context, jsonDir string) error {
	// First get the list of tracking plans
	plans, err := d.catalog.ListTrackingPlans(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tracking plans: %w", err)
	}

	// Get detailed tracking plans with event relationships
	var detailedPlans []catalog.TrackingPlanWithIdentifiers
	for _, plan := range plans {
		detailedPlan, err := d.catalog.GetTrackingPlan(ctx, plan.ID)
		if err != nil {
			return fmt.Errorf("failed to get detailed tracking plan %s: %w", plan.ID, err)
		}
		detailedPlans = append(detailedPlans, *detailedPlan)
	}

	return d.writeJSONFile(filepath.Join(jsonDir, "tracking-plans.json"), detailedPlans)
}

func (d *TrackingPlanDownloader) downloadEvents(ctx context.Context, jsonDir string) error {
	// Get all tracking plan IDs first
	plans, err := d.catalog.ListTrackingPlans(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tracking plans: %w", err)
	}

	var trackingPlanIds []string
	for _, plan := range plans {
		trackingPlanIds = append(trackingPlanIds, plan.ID)
	}

	// Download all events with pagination
	var allEvents []catalog.Event
	page := 1
	for {
		response, err := d.catalog.ListEvents(ctx, trackingPlanIds, page)
		if err != nil {
			return fmt.Errorf("failed to list events for page %d: %w", page, err)
		}

		allEvents = append(allEvents, response.Data...)

		// Check if we've reached the end
		if len(response.Data) == 0 || page*response.PageSize >= response.Total {
			break
		}
		page++
	}

	return d.writeJSONFile(filepath.Join(jsonDir, "events.json"), allEvents)
}

func (d *TrackingPlanDownloader) downloadProperties(ctx context.Context, jsonDir string) error {
	// Get all tracking plan IDs first
	plans, err := d.catalog.ListTrackingPlans(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tracking plans: %w", err)
	}

	var trackingPlanIds []string
	for _, plan := range plans {
		trackingPlanIds = append(trackingPlanIds, plan.ID)
	}

	// Download all properties with pagination
	var allProperties []catalog.Property
	page := 1
	for {
		response, err := d.catalog.ListProperties(ctx, trackingPlanIds, page)
		if err != nil {
			return fmt.Errorf("failed to list properties for page %d: %w", page, err)
		}

		allProperties = append(allProperties, response.Data...)

		// Check if we've reached the end
		if len(response.Data) == 0 || page*response.PageSize >= response.Total {
			break
		}
		page++
	}

	return d.writeJSONFile(filepath.Join(jsonDir, "properties.json"), allProperties)
}

func (d *TrackingPlanDownloader) downloadCustomTypes(ctx context.Context, jsonDir string) error {
	// Download all custom types with pagination
	var allCustomTypes []catalog.CustomType
	page := 1
	for {
		response, err := d.catalog.ListCustomTypes(ctx, page)
		if err != nil {
			return fmt.Errorf("failed to list custom types for page %d: %w", page, err)
		}

		allCustomTypes = append(allCustomTypes, response.Data...)

		// Check if we've reached the end
		if len(response.Data) == 0 || page*response.PageSize >= response.Total {
			break
		}
		page++
	}

	return d.writeJSONFile(filepath.Join(jsonDir, "custom-types.json"), allCustomTypes)
}

func (d *TrackingPlanDownloader) downloadCategories(ctx context.Context, jsonDir string) error {
	// Download all categories with pagination
	var allCategories []catalog.Category
	page := 1
	for {
		response, err := d.catalog.ListCategories(ctx, page)
		if err != nil {
			return fmt.Errorf("failed to list categories for page %d: %w", page, err)
		}

		allCategories = append(allCategories, response.Data...)

		// Check if we've reached the end
		if len(response.Data) == 0 || page*response.PageSize >= response.Total {
			break
		}
		page++
	}

	return d.writeJSONFile(filepath.Join(jsonDir, "categories.json"), allCategories)
}

func (d *TrackingPlanDownloader) writeJSONFile(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// NewCmdTPDownload creates a new cobra command for downloading tracking plans
func NewCmdTPDownload() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download tracking plans and dependencies to JSON files",
		Long: heredoc.Doc(`
			Download all tracking plans and their dependencies to JSON files.

			This command downloads tracking plans, events, properties, custom types,
			and categories from the workspace and saves them as JSON files in the
			specified output directory.
		`),
		Example: heredoc.Doc(`
			# Download to default directory (./tracking-plans)
			$ rudder-cli tp download

			# Download to custom directory
			$ rudder-cli tp download --output-dir ./my-data
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			defer func() {
				telemetry.TrackCommand("tp download", err, []telemetry.KV{
					{K: "output_dir", V: outputDir},
				}...)
			}()

			// Validate output directory
			if outputDir == "" {
				outputDir = "./tracking-plans"
			}

			deps, err := app.NewDeps()
			if err != nil {
				return err
			}

			downloader := NewTrackingPlanDownloader(catalog.NewRudderDataCatalog(deps.Client()), outputDir)

			// Show spinner during download
			spinner := ui.NewSpinner("Downloading tracking plan data...")
			spinner.Start()
			err = downloader.Download(cmd.Context())
			spinner.Stop()

			return err
		},
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", "./tracking-plans", "Output directory for downloaded files")

	return cmd
}