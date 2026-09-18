package retlsource

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/spf13/cobra"
)

// sourceTypes are the RETL source kinds preview and validate can target. Table
// sources only reach the graph when their experimental kind is registered, so
// with the flag off the lookup finds exactly what it did before.
var sourceTypes = []string{sqlmodel.ResourceType, table.ResourceType}

// findSource returns the RETL source with the given id. IDs are only unique per
// kind, so an id shared by two kinds is an error rather than a silent pick.
func findSource(graph *resources.Graph, externalID string) (*resources.Resource, error) {
	var found []*resources.Resource
	for _, resourceType := range sourceTypes {
		if r, ok := graph.GetResource(resources.URN(externalID, resourceType)); ok {
			found = append(found, r)
		}
	}
	switch len(found) {
	case 0:
		return nil, fmt.Errorf("resource with external id '%s' not found in project", externalID)
	case 1:
		return found[0], nil
	}
	types := make([]string, 0, len(found))
	for _, r := range found {
		types = append(types, r.Type())
	}
	// Listed rather than named pairwise: a third rETL source kind (audience) is
	// coming, and "used by both" would then report two of three collisions.
	return nil, fmt.Errorf("external id '%s' is used by %s in the project", externalID, strings.Join(types, ", "))
}

func NewCmdRetlSources() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-sources",
		Short: "Manage RETL sources",
		Long:  "Manage RETL sources in your RudderStack workspace",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newCmdPreview())
	cmd.AddCommand(newCmdValidate())

	return cmd
}
