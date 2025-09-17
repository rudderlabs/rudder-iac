package importremote

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// WorkspaceImporter defines methods for importing workspace resources using a namer for unique IDs.
type WorkspaceImporter interface {
	WorkspaceImport(ctx context.Context, idNamer namer.Namer) ([]FormattableEntity, error)
}

// FormattableEntity represents an importable entity with content, path, and optional template.
type FormattableEntity struct {
	Content      resources.ResourceData
	RelativePath string
	Template     string
}
