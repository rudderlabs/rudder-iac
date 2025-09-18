package experimental

import (
	"reflect"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available experimental flags",
		Long:  "Display all available experimental flags with their current status and environment variable names",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.GetConfig()

			// Define table columns
			columns := []table.Column{
				{Title: "Flag", Width: 20},
				{Title: "Status", Width: 12},
				{Title: "Environment Variable", Width: 40},
			}

			// Get the experimental config struct for values
			experimental := cfg.ExperimentalFlags
			expType := reflect.TypeOf(experimental)
			expValue := reflect.ValueOf(experimental)

			var rows []table.Row

			// Iterate through all fields in the ExperimentalConfig struct
			for i := 0; i < expType.NumField(); i++ {
				field := expType.Field(i)
				value := expValue.Field(i)

				// Get the mapstructure tag as the flag name
				flagName := field.Tag.Get("mapstructure")
				if flagName == "" {
					continue
				}

				// Convert boolean to string
				status := "false"
				if value.Bool() {
					status = ui.Color("true", ui.Yellow)
				}

				// Get the environment variable name
				envVarName := config.GetEnvironmentVariableName(flagName)

				rows = append(rows, table.Row{flagName, status, envVarName})
			}

			ui.RenderTable(columns, rows)

			return nil
		},
	}

	return cmd
}
