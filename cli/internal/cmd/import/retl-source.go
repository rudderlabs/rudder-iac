package importcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	retlSourceImportLog = logger.New("import", logger.Attr{
		Key:   "cmd",
		Value: "retl-source",
	})
)

func NewCmdRetlSource() *cobra.Command {
	var (
		localID     string
		remoteID    string
		location    string
		sqlLocation string
		err         error
	)

	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Import remote RETL SQL Model to local configuration",
		Long: heredoc.Doc(`
			Import a remote RETL SQL Model source into a local YAML configuration file.
			This command fetches the remote SQL Model using the provided remote ID,
			creates a local YAML configuration with the specified local ID, and embeds
			import metadata for tracking.
			
			Optionally, you can specify a separate location for SQL files using --sql-location.
			When provided, the SQL content will be saved as a separate .sql file and the
			YAML configuration will reference it using the 'file' field instead of inline 'sql'.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli import retl-sources --local-id my-model --remote-id abc123
			$ rudder-cli import retl-sources -i analytics-model -r def789 -l ./models
			$ rudder-cli import retl-sources --local-id analytics-model --remote-id def789 --location ./models --sql-location ./sql
			$ rudder-cli import retl-sources -i analytics-model -r def789 -l ./models -s ./sql
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			retlSourceImportLog.Debug("import retl-source", "localID", localID, "remoteID", remoteID, "location", location, "sqlLocation", sqlLocation)
			retlSourceImportLog.Debug("importing remote RETL SQL Model to local configuration")

			defer func() {
				telemetry.TrackCommand("import retl-source", err, []telemetry.KV{
					{K: "localID", V: localID},
					{K: "remoteID", V: remoteID},
					{K: "location", V: location},
					{K: "sqlLocation", V: sqlLocation},
				}...)
			}()

			// Get dependencies
			deps, err := app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			// Cast to RETL provider to access Import method
			retlProvider, ok := deps.Providers().RETL.(*retl.Provider)
			if !ok {
				return fmt.Errorf("failed to cast RETL provider")
			}

			entity, err := retlProvider.FetchImportData(cmd.Context(), sqlmodel.ResourceType, specs.ImportIds{
				RemoteID: remoteID,
				LocalID:  localID,
			})
			if err != nil {
				return fmt.Errorf("importing RETL SQL Model: %w", err)
			}

			// Handle SQL file separation if requested
			if sqlLocation != "" {
				// Extract the spec from the entity
				spec := entity.Content.(*specs.Spec)

				// Write SQL to separate file
				err = writeSQLToFile(spec.Spec, sqlLocation, localID)
				if err != nil {
					return fmt.Errorf("writing SQL file: %w", err)
				}

				// Create a relative path from spec location to SQL file location
				relPath, err := filepath.Rel(location, sqlLocation)
				if err != nil {
					return fmt.Errorf("calculating relative path: %w", err)
				}

				// Always use forward slashes for consistency across OS
				relPath = filepath.ToSlash(relPath)

				// Modify the resource data to reference the SQL file instead of inline SQL
				spec.Spec[sqlmodel.FileKey] = fmt.Sprintf("%s/%s.sql", relPath, localID)
				delete(spec.Spec, sqlmodel.SQLKey)
			}

			// Write the entity using the new write function
			formatters := formatter.Setup(formatter.DefaultYAML)
			err = writer.Write(cmd.Context(), location, formatters, []writer.FormattableEntity{entity})
			if err != nil {
				return fmt.Errorf("writing YAML files: %w", err)
			}

			retlSourceImportLog.Info("Successfully imported RETL SQL Model", "localID", localID, "remoteID", remoteID)
			fmt.Printf("%s Successfully imported RETL SQL Model '%s' from remote ID '%s'\n",
				ui.Color("✓", ui.Green), localID, remoteID)
			fmt.Printf("Configuration saved to: %s/%s.yaml\n", location, localID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&localID, "local-id", "i", "", "Local identifier for the imported SQL Model (required)")
	cmd.Flags().StringVarP(&remoteID, "remote-id", "r", "", "Remote RETL source ID to import (required)")
	cmd.Flags().StringVarP(&location, "location", "l", ".", "Directory where to save the YAML configuration file (default: current directory)")
	cmd.Flags().StringVarP(&sqlLocation, "sql-location", "s", "", "Directory where to save SQL files separately (optional, if not provided SQL will be inline in YAML)")

	// Mark required flags
	cmd.MarkFlagRequired("local-id")
	cmd.MarkFlagRequired("remote-id")

	return cmd
}

func writeSQLToFile(resourceData map[string]interface{}, sqlLocation string, localID string) error {
	if _, err := os.Stat(sqlLocation); os.IsNotExist(err) {
		return fmt.Errorf("SQL directory does not exist: %s", sqlLocation)
	}

	sql, ok := resourceData[sqlmodel.SQLKey].(string)
	if !ok {
		return fmt.Errorf("SQL key not found in resource data")
	}
	err := os.WriteFile(fmt.Sprintf("%s/%s.sql", sqlLocation, localID), []byte(sql), 0644)
	if err != nil {
		return fmt.Errorf("writing SQL file: %w", err)
	}
	return nil
}
