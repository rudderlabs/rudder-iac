package importer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// ManagedDir is where import workspace --managed writes its specs.
const ManagedDir = "managed"

var ErrProjectNotEmpty = errors.New("--managed exports into a project with no specs; pass --skip-existing to keep the specs it has")

// ManagedExportOptions configures ManagedExport. SkipExisting allows a project
// that already has specs and writes only the managed resources it lacks.
type ManagedExportOptions struct {
	SkipExisting bool
}

type ManagedExportProvider interface {
	provider.Exporter
	// LoadManaged returns the managed resources and the names of the providers
	// that cannot export theirs.
	LoadManaged(ctx context.Context) (*resources.RemoteResources, []string, error)
}

// ManagedExport writes the workspace's CLI-managed resources (those with an
// externalId) as specs under ManagedDir, each keyed by its externalId, so that
// plan on the result shows no changes. It recovers what the specs say, not how
// the owner laid them out: file layout, comments, variables and secrets are lost.
func ManagedExport(ctx context.Context, project Project, p ManagedExportProvider, opts ManagedExportOptions) error {
	graph, err := project.ResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}
	// Local specs would collide with the externalIds the export writes, unless
	// SkipExisting leaves those resources out.
	if !opts.SkipExisting && len(graph.Resources()) > 0 {
		return ErrProjectNotEmpty
	}

	managed, skipped, err := p.LoadManaged(ctx)
	if err != nil {
		return fmt.Errorf("loading managed resources: %w", err)
	}
	managed = managed.Without(func(_ string, r *resources.RemoteResource) bool { return r.ExternalID == "" })
	if len(skipped) > 0 {
		ui.PrintWarning("Managed resources of these providers are not exported: " + strings.Join(skipped, ", "))
	}
	if managed.Len() == 0 {
		fmt.Println("No managed resources to export")
		return nil
	}

	toWrite := managed
	if opts.SkipExisting {
		toWrite = managed.Without(func(resourceType string, r *resources.RemoteResource) bool {
			_, local := graph.GetResource(resources.URN(r.ExternalID, resourceType))
			return local
		})
		if toWrite.Len() == 0 {
			fmt.Println("Every managed resource is already in the project")
			return nil
		}
	}

	idNamer, err := initNamer(graph)
	if err != nil {
		return fmt.Errorf("initializing namer: %w", err)
	}
	// Every reference points at another managed resource. One being written
	// resolves through toWrite; one the project already has resolves through
	// its local spec, as a plain import resolves references to managed resources.
	resolver, err := initResolver(managed, toWrite, graph)
	if err != nil {
		return fmt.Errorf("setting up ref resolver: %w", err)
	}

	// Import-manifest entries are dropped: externalIds already link these resources.
	entities, _, err := p.FormatForExport(toWrite, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("formatting managed resources: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)
	return writeSpecs(ctx, filepath.Join(project.Location(), ManagedDir), formatters, entities)
}
