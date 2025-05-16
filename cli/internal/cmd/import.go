package cmd

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/importer"
	"github.com/spf13/cobra"
)

var (
	outputDir string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import tracking plans from various sources",
	Long:  `Import tracking plans from different sources into RudderStack.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		deps, err := app.NewDeps()
		if err != nil {
			return fmt.Errorf("initialising dependencies: %w", err)
		}

		i := importer.New(outputDir, deps.Providers().DataCatalog)
		return i.Import(context.Background(), "property")
	},
}

func init() {
	importCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for imported tracking plans (default: current directory)")
}
