// Package retlconnection holds the commands that act on a rETL connection
// rather than describing one. Listing lives under `workspace`, which is
// read-only by convention; starting and stopping a sync mutates, so it sits
// here alongside retl-sources' preview and validate.
package retlconnection

import (
	"context"
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/spf13/cobra"
)

func NewCmdRetlConnections() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retl-connections",
		Short: "Run and inspect RETL connection syncs",
	}

	cmd.AddCommand(newCmdSync())
	cmd.AddCommand(newCmdStop())

	return cmd
}

// resolveConnection turns the external id an author writes in a spec into the
// remote id the API addresses. Every other command takes the spec id, and
// making this one take an opaque 3JYY… id would be the odd one out.
func resolveConnection(ctx context.Context, d app.Deps, externalID string) (string, error) {
	retlProvider := d.Providers().RETL
	if !slices.Contains(retlProvider.SupportedTypes(), connection.ResourceType) {
		return "", fmt.Errorf("RETL connections are experimental: set RUDDERSTACK_CLI_EXPERIMENTAL=true and RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true")
	}

	rows, err := retlProvider.List(ctx, connection.ResourceType, nil)
	if err != nil {
		return "", fmt.Errorf("listing rETL connections: %w", err)
	}

	for _, row := range rows {
		if row[connection.ExternalIDKey] == externalID {
			id, ok := row[connection.IDKey].(string)
			if !ok || id == "" {
				return "", fmt.Errorf("rETL connection %q has no id", externalID)
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("no rETL connection with external id %q in this workspace", externalID)
}
