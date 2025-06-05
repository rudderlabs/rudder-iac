package cmd

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/experimental/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var experimentalCmd = &cobra.Command{
	Use:    "experimental",
	Short:  "Experimental commands",
	Long:   "Experimental commands that are under development and may change in future versions",
	Hidden: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !viper.GetBool("experimental") {
			return fmt.Errorf("experimental commands are disabled. Enable with --config experimental=true or set RUDDERSTACK_CLI_EXPERIMENTAL=true")
		}
		return nil
	},
}

func init() {
	// Add schema command to experimental
	experimentalCmd.AddCommand(schema.NewCmdSchema())
}
