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

// DestinationHandler is the BaseHandler instantiation for destinations.
type DestinationHandler = handler.BaseHandler[
	DestinationSpec,
	DestinationResource,
	DestinationState,
	RemoteDestination,
]

// HandlerMetadata is the static metadata describing the destination handler
// for the BaseHandler framework.
var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     DestinationResourceType,
	SpecKind:         DestinationSpecKind,
	SpecMetadataName: DestinationMetadataName,
}

// HandlerImpl owns destination CRUD against the API client and uses the
// definitions registry to convert config between snake_case (local/spec) and
// camelCase (API) at the boundary.
type HandlerImpl struct {
	client   *client.Client
	registry *definitions.Registry
}

// NewHandler builds a *DestinationHandler wired to the given client and registry.
func NewHandler(c *client.Client, registry *definitions.Registry) *DestinationHandler {
	h := &HandlerImpl{client: c, registry: registry}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *DestinationSpec {
	return &DestinationSpec{}
}

// ExtractResourcesFromSpec decodes a parsed spec into a DestinationResource,
// parsing the scalar "#transformation:<id>" reference into a PropertyRef whose
// resolver reads TransformationState.ID. Config stays snake_case.
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
		ref, err := parseTransformationRef(spec.Transformation)
		if err != nil {
			return nil, err
		}
		resource.Transformation = ref
	}

	return map[string]*DestinationResource{spec.ID: resource}, nil
}

// Create provisions the destination remotely, then links a transformation if the
// spec-side PropertyRef was resolved by the apply framework.
func (h *HandlerImpl) Create(ctx context.Context, data *DestinationResource) (*DestinationState, error) {
	apiConfig, err := h.localConfigToAPI(data.Type, data.DefinitionVersion, data.Config)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
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

	transformationID, err := h.syncTransformationLink(
		ctx,
		created.ID,
		data.Transformation,
		"", // no previous transformation ID
	)
	if err != nil {
		return nil, fmt.Errorf("syncing transformation link: %w", err)
	}

	return &DestinationState{ID: created.ID, TransformationID: transformationID}, nil
}

// Update rejects an immutable type change, repushes config (converted via the
// registry), and reconciles the transformation link against the previous state.
// The framework resolves newData.Transformation before Update is called, so the
// resolved ID is read from the ref's Value.
func (h *HandlerImpl) Update(
	ctx context.Context,
	newData *DestinationResource,
	oldData *DestinationResource,
	oldState *DestinationState,
) (*DestinationState, error) {
	if newData.Type != oldData.Type {
		return nil, fmt.Errorf("destination type change is not supported: old %q, new %q", oldData.Type, newData.Type)
	}

	apiConfig, err := h.localConfigToAPI(newData.Type, newData.DefinitionVersion, newData.Config)
	if err != nil {
		return nil, err
	}

	if _, err := h.client.Destinations.Update(ctx, &client.Destination{
		ID:        oldState.ID,
		Name:      newData.DisplayName,
		Type:      newData.Type,
		IsEnabled: newData.Enabled,
		Config:    apiConfig,
	}); err != nil {
		return nil, fmt.Errorf("updating destination: %w", err)
	}

	newTransformationID, err := h.syncTransformationLink(
		ctx,
		oldState.ID,
		newData.Transformation,
		oldState.TransformationID,
	)
	if err != nil {
		return nil, err
	}

	return &DestinationState{
		ID:               oldState.ID,
		TransformationID: newTransformationID,
	}, nil
}

// Delete disconnects any linked transformation first, then deletes the destination.
func (h *HandlerImpl) Delete(ctx context.Context, _ string, _ *DestinationResource, oldState *DestinationState) error {
	if oldState.TransformationID != "" {
		if _, err := h.client.Destinations.DisconnectTransformation(
			ctx,
			oldState.ID,
		); err != nil {
			return fmt.Errorf("disconnecting transformation from destination: %w", err)
		}
	}

	if err := h.client.Destinations.Delete(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting destination: %w", err)
	}

	return nil
}

// MapRemoteToState converts a remote destination into the spec-side resource
// and the persisted state. Managed resources with an unregistered type are
// treated as corruption (error); the transformation link is resolved back to a
// URN via the urnResolver, gracefully degrading when the linked transformation
// is not CLI-managed (mirrors the event-stream source pattern).
func (h *HandlerImpl) MapRemoteToState(
	remote *RemoteDestination,
	urnResolver handler.URNResolver,
) (*DestinationResource, *DestinationState, error) {
	if remote.ExternalID == "" {
		return nil, nil, fmt.Errorf("managed destination %s has empty external ID", remote.ID)
	}

	if !h.registry.IsSupported(remote.Type) {
		return nil, nil, fmt.Errorf("managed destination %s has unregistered type %q", remote.ID, remote.Type)
	}

	version := int64(remote.Version)
	localConfig, err := h.apiConfigToLocal(remote.Type, version, remote.Config)
	if err != nil {
		return nil, nil, err
	}

	transformationRef, transformationID, err := h.transformationRef(remote, urnResolver)
	if err != nil {
		return nil, nil, err
	}

	resource := &DestinationResource{
		ID:                remote.ExternalID,
		DisplayName:       remote.Name,
		Type:              remote.Type,
		Enabled:           remote.IsEnabled,
		DefinitionVersion: version,
		Transformation:    transformationRef,
		Config:            localConfig,
	}

	state := &DestinationState{
		ID:               remote.ID,
		TransformationID: transformationID,
	}

	return resource, state, nil
}

// LoadRemoteResources returns only managed destinations (ExternalID set). An
// unregistered type on a managed resource indicates corrupted state and errors.
func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteDestination, error) {
	all, err := h.client.Destinations.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	result := make([]*RemoteDestination, 0, len(all))
	for i := range all {
		d := &all[i]
		// TODO: Move the filtering logic to the API client. Remove
		// this check and comment once we have API filtering support.
		if d.ExternalID == "" {
			continue
		}

		if !h.registry.IsSupported(d.Type) {
			return nil, fmt.Errorf(
				"managed destination %s has unregistered type %q",
				d.ID,
				d.Type,
			)
		}
		result = append(result, &RemoteDestination{Destination: d})
	}
	return result, nil
}

// LoadImportableResources returns unmanaged destinations (no ExternalID) and
// silently skips unregistered types — import can only target types the CLI knows.
func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteDestination, error) {
	all, err := h.client.Destinations.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing destinations: %w", err)
	}

	result := make([]*RemoteDestination, 0, len(all))
	for i := range all {
		d := &all[i]
		// TODO: Move the filtering logic to the API client. Remove
		// this check and comment once we have API filtering support.
		if d.ExternalID != "" {
			continue
		}
		if !h.registry.IsSupported(d.Type) {
			// Only destinations which are supported by the registry
			// in the CLI are considered importable.
			continue
		}
		result = append(result, &RemoteDestination{Destination: d})
	}
	return result, nil
}

// Import is deferred to RUD-2865.
func (h *HandlerImpl) Import(_ context.Context, _ *DestinationResource, _ string) (*DestinationState, error) {
	return nil, fmt.Errorf("import not implemented yet")
}

// FormatForExport is deferred to RUD-2865.
func (h *HandlerImpl) FormatForExport(
	_ map[string]*RemoteDestination,
	_ namer.Namer,
	_ resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	return nil, fmt.Errorf("export not implemented yet")
}

// localConfigToAPI resolves the registered definition and converts snake_case
// config to the camelCase form the API expects, returning JSON-serializable bytes.
func (h *HandlerImpl) localConfigToAPI(destType string, version int64, local map[string]any) (json.RawMessage, error) {
	registered, err := h.registry.Get(destType, version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	apiConfig, err := registered.LocalToAPI(local)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	bytes, err := json.Marshal(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("marshalling destination config: %w", err)
	}
	return bytes, nil
}

// apiConfigToLocal is the inverse of localConfigToAPI, used by MapRemoteToState.
func (h *HandlerImpl) apiConfigToLocal(destType string, version int64, apiConfig json.RawMessage) (map[string]any, error) {
	registered, err := h.registry.Get(destType, version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	var apiMap map[string]any
	if err := json.Unmarshal(apiConfig, &apiMap); err != nil {
		return nil, fmt.Errorf("unmarshalling destination config: %w", err)
	}
	if apiMap == nil {
		apiMap = map[string]any{}
	}

	local, err := registered.APIToLocal(apiMap)
	if err != nil {
		return nil, fmt.Errorf("converting API config to local: %w", err)
	}
	return local, nil
}

// transformationRef resolves the remote transformation link back to a
// spec-side PropertyRef. When the linked transformation is not CLI-managed
// (ErrRemoteResourceExternalIdNotFound) the ref is dropped but the remote ID is
// still tracked in state so the link re-applies on every run — the established
// tradeoff from the event-stream source handler.
func (h *HandlerImpl) transformationRef(remote *RemoteDestination, urnResolver handler.URNResolver) (*resources.PropertyRef, string, error) {
	if remote.Transformation == nil || remote.Transformation.ID == "" {
		return nil, "", nil
	}

	transformationID := remote.Transformation.ID
	urn, err := urnResolver.GetURNByID(ttypes.TransformationResourceType, transformationID)
	if err != nil {
		if err == resources.ErrRemoteResourceExternalIdNotFound {
			// It might be a transformation which the upstream user's manually added but
			// not managed by the CLI yet.
			return nil, transformationID, nil
		}
		return nil, "", fmt.Errorf("resolving transformation URN: %w", err)
	}

	return &resources.PropertyRef{
		URN:      urn,
		Property: "id",
	}, transformationID, nil
}

// syncTransformationLink reconciles the link state during Update. The differ
// has already flagged the resource as changed; this method picks the right
// connect/disconnect call based on the new ref vs the previously stored ID.
func (h *HandlerImpl) syncTransformationLink(
	ctx context.Context,
	destinationID string,
	newRef *resources.PropertyRef,
	oldTransformationID string,
) (string, error) {
	newTransformationID, err := resolveTransformationID(newRef)
	if err != nil {
		return "", fmt.Errorf("resolving transformation: %w", err)
	}

	if oldTransformationID == newTransformationID {
		return newTransformationID, nil
	}

	if newTransformationID == "" {
		if _, err := h.client.Destinations.DisconnectTransformation(ctx, destinationID); err != nil {
			return "", fmt.Errorf("disconnecting transformation from destination: %w", err)
		}
		return "", nil
	}

	if _, err := h.client.Destinations.ConnectTransformation(ctx, destinationID, newTransformationID); err != nil {
		return "", fmt.Errorf("connecting transformation to destination: %w", err)
	}

	return newTransformationID, nil
}

// resolveTransformationID extracts the remote ID from a resolved PropertyRef.
func resolveTransformationID(ref *resources.PropertyRef) (string, error) {
	if ref == nil {
		return "", nil
	}

	if !ref.IsResolved || ref.Value == "" {
		return "", fmt.Errorf("transformation reference is not resolved or has empty value")
	}

	return ref.Value, nil
}

// parseTransformationRef parses a scalar "#transformation:<id>" reference into
// a PropertyRef whose Resolve function reads TransformationState.ID.
func parseTransformationRef(ref string) (*resources.PropertyRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("transformation reference is empty")
	}

	if !strings.HasPrefix(ref, "#") {
		return nil, fmt.Errorf("invalid transformation reference %q: expected format #transformation:<id>", ref)
	}

	body := strings.TrimPrefix(ref, "#")
	parts := strings.SplitN(body, ":", 2)
	if len(parts) != 2 || parts[0] != ttypes.TransformationSpecKind || parts[1] == "" {
		return nil, fmt.Errorf("invalid transformation reference %q: expected format #transformation:<id>", ref)
	}

	urn := resources.URN(parts[1], ttypes.TransformationResourceType)
	return createTransformationRef(urn), nil
}

// createTransformationRef wraps handler.CreatePropertyRef and stamps the
// "id" property so the differ's comparePropertyRefs sees a stable shape on
// both the spec and state sides.
func createTransformationRef(urn string) *resources.PropertyRef {
	ref := handler.CreatePropertyRef(
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
