package provider

import "github.com/rudderlabs/rudder-iac/cli/internal/provider"

type WorkspaceImporter interface {
	provider.UnmanagedRemoteResourceLoader
	provider.Exporter
}
