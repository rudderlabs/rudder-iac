package importremote

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// WorkspaceImporter defines methods for importing workspace resources using a namer for unique IDs.
type WorkspaceImporter interface {
	LoadImportableResources(ctx context.Context) (*resources.ResourceCollection, error)
	AssignExternalIDs(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer) error
	NormalizeForImport(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer, resolver resolver.ReferenceResolver) ([]FormattableEntity, error)
}

// FormattableEntity represents an importable entity with content, path, and optional template.
type FormattableEntity struct {
	Content      any
	RelativePath string
	Template     string
}
