package export

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type SingleSpecExportHandler[Spec any, Remote handler.RemoteResource] interface {
	Metadata() handler.HandlerMetadata
	MapRemoteToSpec(data map[string]*Remote, inputResolver resolver.ReferenceResolver) (*SpecExportData[Spec], error)
}

type SingleSpecExportStrategy[Spec any, Remote handler.RemoteResource] struct {
	Handler      SingleSpecExportHandler[Spec, Remote]
	secretFields []handler.SecretField
}

func (s *SingleSpecExportStrategy[Spec, Remote]) SetSecretFields(fields []handler.SecretField) {
	s.secretFields = fields
}

func (s *SingleSpecExportStrategy[Spec, Remote]) FormatForExport(
	remotes map[string]*Remote,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	importResources, entries, workspaceID, err := s.buildImportData(remotes)
	if err != nil {
		return nil, nil, err
	}

	exportData, err := s.Handler.MapRemoteToSpec(remotes, inputResolver)
	if err != nil {
		return nil, nil, fmt.Errorf("mapping remote to spec: %w", err)
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
		return nil, nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	if err := exportData.AttachSecretVariables(s.Handler.Metadata().ResourceType, singleRemoteLocalID(remotes), s.secretFields); err != nil {
		return nil, nil, fmt.Errorf("attaching secret variables: %w", err)
	}

	specData, err := exportData.ToMap()
	if err != nil {
		return nil, nil, fmt.Errorf("converting spec data to map: %w", err)
	}

	spec := &specs.Spec{
		Version:  specs.SpecVersionV1,
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

	return result, entries, nil
}

func (s *SingleSpecExportStrategy[Spec, Remote]) buildImportData(remotes map[string]*Remote) ([]specs.ImportIds, []importmanifest.ImportEntry, string, error) {
	importResources := make([]specs.ImportIds, 0, len(remotes))
	entries := make([]importmanifest.ImportEntry, 0, len(remotes))
	workspaceID := ""
	for localID, remote := range remotes {
		remoteMetadata := (*remote).Metadata()
		if workspaceID != "" && workspaceID != remoteMetadata.WorkspaceID {
			return nil, nil, "", fmt.Errorf("cannot export resources from multiple workspaces into a single spec file")
		}
		if workspaceID == "" {
			workspaceID = remoteMetadata.WorkspaceID
		}

		urn := resources.URN(localID, s.Handler.Metadata().ResourceType)
		importResources = append(importResources, specs.ImportIds{URN: urn, RemoteID: remoteMetadata.ID})
		entries = append(entries, importmanifest.ImportEntry{
			WorkspaceID: remoteMetadata.WorkspaceID,
			URN:         urn,
			RemoteID:    remoteMetadata.ID,
		})
	}
	return importResources, entries, workspaceID, nil
}

func singleRemoteLocalID[Remote handler.RemoteResource](remotes map[string]*Remote) string {
	if len(remotes) != 1 {
		return ""
	}
	for localID := range remotes {
		return localID
	}
	return ""
}
