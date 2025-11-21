package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

/*

NOTE: This file defines interfaces related to chunks of provider functionality.

It starts with the WorkspaceImporter interface, which is responsible for importing workspace resources.
but will later include additional interfaces as needed to encapsulate other provider-related operations,
which are currently scattered across various parts of the codebase.

The idea is that different components of the system will define which subset of provider functionality they require
e.g:

- Importer component will depend on WorkspaceImporter and some StateLoader interface (to be defined later), because
  the importer is also responsible for ensuring the current project is synced

- Syncer component will depend on StateLoader and some CRUD interface (to be defined later), because the syncer is responsible for
	loading the current state and applying changes through CRUD operations

*/

// WorkspaceImporter defines methods for importing workspace resources using a namer for unique IDs.
// Based on the previous NOTE, LoadImportable should probably be part of RemoteResourceLoader interface (to be defined later)
// that handles loading resources from remote sources (like the workspace), either managed or unmanaged.
type WorkspaceImporter interface {
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error)
	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		resolver resolver.ReferenceResolver,
	) ([]writer.FormattableEntity, error)
}
