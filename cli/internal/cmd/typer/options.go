package typer

import (
	"fmt"
	"reflect"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCmdOptions() *cobra.Command {
	var platform string

	cmd := &cobra.Command{
		Use:   "options",
		Short: "Show available options for a platform",
		Long:  "Show all available platform-specific options for code generation",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !config.GetConfig().ExperimentalFlags.RudderTyper {
				return fmt.Errorf("typer commands are disabled")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := generator.GeneratorForPlatform(platform)
			if err != nil {
				return err
			}

			defaults := gen.DefaultOptions()
			printOptionsTable(defaults)

			return nil
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "", "Platform to show options for (kotlin)")
	cmd.MarkFlagRequired("platform")

	return cmd
}

// printOptionsTable prints platform options in a readable table format
func printOptionsTable(defaults any) {
	// Define table columns
	columns := []table.Column{
		{Title: "Option", Width: 20},
		{Title: "Description", Width: 80},
		{Title: "Default", Width: 30},
	}

	defaultsType := reflect.TypeOf(defaults)
	defaultsValue := reflect.ValueOf(defaults)

	var rows []table.Row

	// Iterate through all fields in the options struct
	for i := 0; i < defaultsType.NumField(); i++ {
		field := defaultsType.Field(i)
		value := defaultsValue.Field(i)

		// Get the mapstructure tag as the option name
		optionName := field.Tag.Get("mapstructure")
		if optionName == "" {
			continue
		}

		description := field.Tag.Get("description")
		defaultValue := fmt.Sprintf("%v", value.Interface())

		rows = append(rows, table.Row{optionName, description, defaultValue})
	}

	fmt.Println()
	ui.RenderTable(columns, rows)
	fmt.Println()
}
