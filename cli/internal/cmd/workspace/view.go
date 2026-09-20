package workspace

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/lister"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// externalIDKey is the field every managed resource carries to name itself in a
// spec. A view addresses resources the way an author wrote them, not by the
// remote id, which is what sync, stop and syncs view already do.
const externalIDKey = "externalId"

// findByExternalID picks one resource out of a kind's listing. The providers
// expose no get-by-external-id, and adding one to every provider to save a
// listing would be a larger change than the feature is worth; a workspace's
// listing of a single kind is small.
//
// ponytail: lists and filters client-side; add a provider-level lookup if a
// workspace ever holds enough of one kind for this to matter.
func findByExternalID(
	ctx context.Context,
	p lister.ListProvider,
	resourceType, externalID, label string,
) ([]resources.ResourceData, error) {
	rows, err := p.List(ctx, resourceType, nil)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row[externalIDKey] == externalID {
			return []resources.ResourceData{row}, nil
		}
	}
	return nil, fmt.Errorf("no %s with external id %q in this workspace", label, externalID)
}

// oneResource adapts a single row back to the lister, so a view renders through
// the same formatter as a list and --json emits the same object shape.
type oneResource []resources.ResourceData

func (o oneResource) List(context.Context, string, lister.Filters) ([]resources.ResourceData, error) {
	return o, nil
}

// viewFormat maps the --json flag onto the detailed renderer, which unlike the
// table needs no TTY.
func viewFormat(jsonOutput bool) lister.OutputFormat {
	if jsonOutput {
		return lister.JSONFormat
	}
	return lister.DetailedFormat
}
