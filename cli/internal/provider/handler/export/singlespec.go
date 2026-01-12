package export

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type SingleSpecExportHandler[Spec any, Remote handler.RemoteResource] interface {
	Metadata() handler.HandlerMetadata
	MapRemoteToSpec(data map[string]*Remote, inputResolver resolver.ReferenceResolver) (*SpecExportData[Spec], error)
}

type SingleSpecExportStrategy[Spec any, Remote handler.RemoteResource] struct {
	Handler SingleSpecExportHandler[Spec, Remote]
}

func (s *SingleSpecExportStrategy[Spec, Remote]) FormatForExport(
	remotes map[string]*Remote,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	importResources := make([]specs.ImportIds, 0, len(remotes))
	workspaceID := ""
	for localID, remote := range remotes {
		remoteMetadata := (*remote).Metadata()
		if workspaceID == "" {
			workspaceID = remoteMetadata.WorkspaceID
		} else {
			if workspaceID != remoteMetadata.WorkspaceID {
				return nil, fmt.Errorf("cannot export resources from multiple workspaces into a single spec file")
			}
		}

		importResources = append(importResources, specs.ImportIds{
			LocalID:  localID,
			RemoteID: remoteMetadata.ID,
		})
	}

	exportData, err := s.Handler.MapRemoteToSpec(remotes, inputResolver)
	if err != nil {
		return nil, fmt.Errorf("mapping remote to spec: %w", err)
	}

	specMetadata := specs.Metadata{
		Name: s.Handler.Metadata().SpecMetadataName,
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{
				{
					WorkspaceID: workspaceID,
					Resources:   importResources,
				},
			},
		},
	}

	metadataMap, err := specMetadata.ToMap()
	if err != nil {
		return nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	specData, err := exportData.ToMap()
	if err != nil {
		return nil, fmt.Errorf("converting spec data to map: %w", err)
	}

	spec := &specs.Spec{
		Version:  specs.SpecVersionV0_1,
		Kind:     s.Handler.Metadata().SpecKind,
		Metadata: metadataMap,
		Spec:     specData,
	}

	result := []writer.FormattableEntity{
		{
			Content:      spec,
			RelativePath: exportData.RelativePath,
		},
	}

	return result, nil
}
