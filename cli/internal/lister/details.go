package lister

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func printListWithDetails(rs []resources.ResourceData) error {
	fmt.Println(ui.RenderDetailsList(rs))
	return nil
}
