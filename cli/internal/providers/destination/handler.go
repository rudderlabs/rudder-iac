package destination

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	tmodel "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type DestinationHandler = handler.BaseHandler[
	DestinationSpec,
	DestinationResource,
	DestinationState,
	RemoteDestination,
]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     DestinationResourceType,
	SpecKind:         DestinationSpecKind,
	SpecMetadataName: DestinationMetadataName,
}

type HandlerImpl struct {
	client   *client.Client
	registry *definitions.Registry
}

func NewHandler(c *client.Client, registry *definitions.Registry) *DestinationHandler {
	return handler.NewHandler(&HandlerImpl{
		client:   c,
		registry: registry,
	})
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *DestinationSpec {
	return &DestinationSpec{}
}

func (h *HandlerImpl) ExtractResourcesFromSpec(_ string, spec *DestinationSpec) (map[string]*DestinationResource, error) {
	resource := &DestinationResource{
		ID:                spec.ID,
		DisplayName:       spec.DisplayName,
		Type:              spec.Type,
		Enabled:           spec.Enabled,
		DefinitionVersion: spec.DefinitionVersion,
		Config:            spec.Config,
	}

	if spec.Transformation != "" {
		transformationRef, err := parseTransformationRef(spec.Transformation)
		if err != nil {
			return nil, fmt.Errorf("parsing transformation reference: %w", err)
		}
		resource.Transformation = transformationRef
	}

	return map[string]*DestinationResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteDestination, error) {
	destinations, err := h.client.Destinations.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	result := make([]*RemoteDestination, 0, len(destinations))
	for i := range destinations {
		dest := destinations[i]
		if dest.ExternalID == "" {
			continue
		}
		if !h.registry.IsSupported(dest.Type) {
			return nil, fmt.Errorf("destination %q has unregistered type %q", dest.ID, dest.Type)
		}
		result = append(result, &RemoteDestination{Destination: &dest})
	}

	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteDestination, error) {
	destinations, err := h.client.Destinations.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	result := make([]*RemoteDestination, 0, len(destinations))
	for i := range destinations {
		dest := destinations[i]
		if dest.ExternalID != "" {
			continue
		}
		if !h.registry.IsSupported(dest.Type) {
			continue
		}
		result = append(result, &RemoteDestination{Destination: &dest})
	}

	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *RemoteDestination, urnResolver handler.URNResolver) (*DestinationResource, *DestinationState, error) {
	if remote.ExternalID == "" {
		return nil, nil, nil
	}

	if !h.registry.IsSupported(remote.Type) {
		return nil, nil, fmt.Errorf("destination %q has unregistered type %q", remote.ID, remote.Type)
	}

	definitionVersion, err := h.latestDefinitionVersion(remote.Type)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving definition version for type %q: %w", remote.Type, err)
	}

	registered, err := h.registry.Get(remote.Type, definitionVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("getting destination definition: %w", err)
	}

	apiConfig, err := unmarshalConfig(remote.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}

	localConfig, err := registered.APIToLocal(apiConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("converting API config to local: %w", err)
	}

	transformationRef, transformationID, err := h.mapTransformationRef(remote, urnResolver)
	if err != nil {
		return nil, nil, err
	}

	resource := &DestinationResource{
		ID:                remote.ExternalID,
		DisplayName:       remote.Name,
		Type:              remote.Type,
		Enabled:           remote.IsEnabled,
		DefinitionVersion: definitionVersion,
		Transformation:    transformationRef,
		Config:            localConfig,
	}

	state := &DestinationState{
		ID:               remote.ID,
		TransformationID: transformationID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *DestinationResource) (*DestinationState, error) {
	apiConfig, err := h.localConfigToAPI(data.Type, data.DefinitionVersion, data.Config)
	if err != nil {
		return nil, err
	}

	created, err := h.client.Destinations.Create(ctx, &client.Destination{
		Name:       data.DisplayName,
		Type:       data.Type,
		IsEnabled:  data.Enabled,
		Config:     apiConfig,
		ExternalID: data.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating destination: %w", err)
	}

	transformationID, err := h.connectTransformationIfPresent(ctx, created.ID, data.Transformation)
	if err != nil {
		return nil, err
	}

	return &DestinationState{
		ID:               created.ID,
		TransformationID: transformationID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *DestinationResource, oldData *DestinationResource, oldState *DestinationState) (*DestinationState, error) {
	if newData.Type != oldData.Type {
		return nil, fmt.Errorf("destination type is immutable: cannot change from %q to %q", oldData.Type, newData.Type)
	}

	apiConfig, err := h.localConfigToAPI(newData.Type, newData.DefinitionVersion, newData.Config)
	if err != nil {
		return nil, err
	}

	_, err = h.client.Destinations.Update(ctx, &client.Destination{
		ID:        oldState.ID,
		Name:      newData.DisplayName,
		Type:      newData.Type,
		IsEnabled: newData.Enabled,
		Config:    apiConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("updating destination: %w", err)
	}

	transformationID, err := h.syncTransformationLink(ctx, oldState.ID, oldState.TransformationID, newData.Transformation)
	if err != nil {
		return nil, err
	}

	return &DestinationState{
		ID:               oldState.ID,
		TransformationID: transformationID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, _ string, _ *DestinationResource, oldState *DestinationState) error {
	if oldState.TransformationID != "" {
		if _, err := h.client.Destinations.DisconnectTransformation(ctx, oldState.ID); err != nil {
			return fmt.Errorf("disconnecting transformation from destination: %w", err)
		}
	}

	if err := h.client.Destinations.Delete(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting destination: %w", err)
	}

	return nil
}

func (h *HandlerImpl) Import(_ context.Context, _ *DestinationResource, _ string) (*DestinationState, error) {
	return nil, fmt.Errorf("import not implemented yet")
}

func (h *HandlerImpl) FormatForExport(
	_ map[string]*RemoteDestination,
	_ namer.Namer,
	_ resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	return nil, fmt.Errorf("export not implemented yet")
}

func (h *HandlerImpl) localConfigToAPI(destType string, version int64, local map[string]any) (json.RawMessage, error) {
	registered, err := h.registry.Get(destType, version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	apiConfig, err := registered.LocalToAPI(local)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	configBytes, err := json.Marshal(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("marshalling destination config: %w", err)
	}

	return configBytes, nil
}

func (h *HandlerImpl) latestDefinitionVersion(destType string) (int64, error) {
	versions, err := h.registry.Versions(destType)
	if err != nil {
		return 0, err
	}
	return versions[len(versions)-1], nil
}

func (h *HandlerImpl) mapTransformationRef(
	remote *RemoteDestination,
	urnResolver handler.URNResolver,
) (*resources.PropertyRef, string, error) {
	if remote.Transformation == nil || remote.Transformation.ID == "" {
		return nil, "", nil
	}

	transformationID := remote.Transformation.ID
	urn, err := urnResolver.GetURNByID(ttypes.TransformationResourceType, transformationID)
	if err != nil {
		if err == resources.ErrRemoteResourceExternalIdNotFound {
			return nil, transformationID, nil
		}
		return nil, "", fmt.Errorf("resolving transformation URN: %w", err)
	}

	return &resources.PropertyRef{
		URN:      urn,
		Property: "id",
	}, transformationID, nil
}

func (h *HandlerImpl) connectTransformationIfPresent(ctx context.Context, destinationID string, ref *resources.PropertyRef) (string, error) {
	transformationID, ok := resolvedTransformationID(ref)
	if !ok {
		return "", nil
	}

	if _, err := h.client.Destinations.ConnectTransformation(ctx, destinationID, transformationID); err != nil {
		return "", fmt.Errorf("connecting transformation to destination: %w", err)
	}

	return transformationID, nil
}

func (h *HandlerImpl) syncTransformationLink(
	ctx context.Context,
	destinationID string,
	oldTransformationID string,
	newRef *resources.PropertyRef,
) (string, error) {
	newTransformationID, hasNewRef := resolvedTransformationID(newRef)

	switch {
	case hasNewRef && oldTransformationID == "":
		if _, err := h.client.Destinations.ConnectTransformation(ctx, destinationID, newTransformationID); err != nil {
			return "", fmt.Errorf("connecting transformation to destination: %w", err)
		}
		return newTransformationID, nil
	case hasNewRef && oldTransformationID != "" && newTransformationID != oldTransformationID:
		if _, err := h.client.Destinations.ConnectTransformation(ctx, destinationID, newTransformationID); err != nil {
			return "", fmt.Errorf("connecting transformation to destination: %w", err)
		}
		return newTransformationID, nil
	case hasNewRef && oldTransformationID != "" && newTransformationID == oldTransformationID:
		return oldTransformationID, nil
	case !hasNewRef && oldTransformationID != "":
		if _, err := h.client.Destinations.DisconnectTransformation(ctx, destinationID); err != nil {
			return "", fmt.Errorf("disconnecting transformation from destination: %w", err)
		}
		return "", nil
	default:
		return "", nil
	}
}

func parseTransformationRef(ref string) (*resources.PropertyRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("transformation reference is empty")
	}

	refBody := strings.TrimPrefix(ref, "#")
	parts := strings.SplitN(refBody, ":", 2)
	if len(parts) != 2 || parts[0] != ttypes.TransformationSpecKind || parts[1] == "" {
		return nil, fmt.Errorf("invalid transformation reference %q: expected format #transformation:<id>", ref)
	}

	urn := resources.URN(parts[1], ttypes.TransformationResourceType)
	return createTransformationRef(urn), nil
}

func createTransformationRef(urn string) *resources.PropertyRef {
	ref := handler.CreatePropertyRef[tmodel.TransformationState](
		urn,
		func(state *tmodel.TransformationState) (string, error) {
			if state.ID == "" {
				return "", fmt.Errorf("transformation state has empty ID")
			}
			return state.ID, nil
		},
	)
	ref.Property = "id"
	return ref
}

func resolvedTransformationID(ref *resources.PropertyRef) (string, bool) {
	if ref == nil || !ref.IsResolved || ref.Value == "" {
		return "", false
	}
	return ref.Value, true
}

func unmarshalConfig(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	return config, nil
}
