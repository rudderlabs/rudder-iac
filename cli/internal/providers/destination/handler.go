package destination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers"
	tmodel "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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
	h := &HandlerImpl{
		client:   c,
		registry: registry,
	}
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
// resolver reads TransformationState.ID. Config stays snake_case; registered
// secret keys are wrapped as *secret.String so the differ's secret-aware
// branch owns comparison.
func (h *HandlerImpl) ExtractResourcesFromSpec(_ string, spec *DestinationSpec) (map[string]*DestinationResource, error) {
	registered, err := h.registry.Get(spec.Type, spec.DefinitionVersion)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	resource := &DestinationResource{
		ID:                spec.ID,
		DisplayName:       spec.DisplayName,
		Type:              spec.Type,
		Enabled:           spec.Enabled,
		DefinitionVersion: spec.DefinitionVersion,
		Config:            wrapKnownSecrets(spec.Config, registered.SecretKeys()),
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
	registered, err := h.registry.Get(data.Type, data.DefinitionVersion)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	apiConfig, err := h.localConfigToAPI(data.Type, data.DefinitionVersion, data.Config)
	if err != nil {
		return nil, fmt.Errorf("converting local config to API: %w", err)
	}

	created, err := h.client.Destinations.Create(ctx, &client.Destination{
		Name:       data.DisplayName,
		Type:       registered.APIType,
		IsEnabled:  data.Enabled,
		Config:     apiConfig,
		ExternalID: data.ID,
		Version:    data.DefinitionVersion,
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

	registered, err := h.registry.Get(newData.Type, newData.DefinitionVersion)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	apiConfig, err := h.localConfigToAPI(newData.Type, newData.DefinitionVersion, newData.Config)
	if err != nil {
		return nil, err
	}

	if _, err := h.client.Destinations.Update(ctx, &client.Destination{
		ID:        oldState.ID,
		Version:   newData.DefinitionVersion,
		Name:      newData.DisplayName,
		Type:      registered.APIType,
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
		if err := h.client.Destinations.DisconnectTransformation(
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

	version := int64(remote.Version)
	registered, err := h.registry.GetByAPIType(remote.Type, version)
	if err != nil {
		return nil, nil, fmt.Errorf("managed destination %s has unregistered type %q and version %d", remote.ID, remote.Type, remote.Version)
	}

	localConfig, err := h.apiConfigToLocal(
		registered.Type,
		version,
		remote.Config,
	)
	if err != nil {
		return nil, nil, err
	}

	// API responses omit secret values; mark every registered secret key as
	// unknown so the differ flags them SecretOnly rather than phantom drift.
	localConfig = wrapUnknownSecrets(localConfig, registered.SecretKeys())

	transformationRef, transformationID, err := h.transformationRef(remote, urnResolver)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving transformation reference: %w", err)
	}

	resource := &DestinationResource{
		ID:                remote.ExternalID,
		DisplayName:       remote.Name,
		Type:              registered.Type,
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

		if _, err := h.registry.GetByAPIType(d.Type, d.Version); err != nil {
			return nil, fmt.Errorf(
				"managed destination %s has unregistered type %q and version %d",
				d.ID,
				d.Type,
				d.Version,
			)
		}
		result = append(result, &RemoteDestination{Destination: d})
	}
	return result, nil
}

// LoadImportableResources returns unmanaged destinations (no ExternalID) and
// silently skips destinations whose (Type, Version) pair isn't registered —
// import can only target definitions the CLI knows how to convert.
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
		if _, err := h.registry.GetByAPIType(d.Type, d.Version); err != nil {
			// Only destinations whose exact (apiType, version) is registered
			// in the CLI are considered importable.
			continue
		}
		result = append(result, &RemoteDestination{Destination: d})
	}
	return result, nil
}

// Import adopts an existing remote destination into IaC management: it pushes
// the spec's config and transformation link via Update (DRY - same reconciliation
// path as a regular apply), then sets the external ID last so a failed Update
// never leaves a partially-adopted resource behind.
func (h *HandlerImpl) Import(ctx context.Context, data *DestinationResource, remoteId string) (*DestinationState, error) {
	remote, err := h.client.Destinations.Get(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting destination during import: %w", err)
	}

	// The single-resource Get endpoint doesn't embed the transformation link
	// (unlike the list endpoint used by LoadImportableResources), so it's
	// fetched separately. No existing link is treated the same as an error
	// here since the API has no "not found" sentinel for this sub-resource.
	var transformationID string
	connectedTransformation, err := h.client.Destinations.GetTransformation(ctx, remoteId)
	if err != nil {
		if !errors.Is(err, client.ErrResourceNotFound) {
			// If the transformation is not found,
			// we will set empty transformation ID
			// in the state
			return nil, fmt.Errorf("getting transformation during import: %w", err)
		}
	}

	if connectedTransformation != nil {
		transformationID = connectedTransformation.TransformationID
	}

	registered, err := h.registry.GetByAPIType(remote.Type, remote.Version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition during import: %w", err)
	}

	// Translate API type to local type so Update's immutable-type check
	// compares local names (e.g. "s3" vs "s3"), not "s3" vs "S3".
	oldData := &DestinationResource{Type: registered.Type}
	oldState := &DestinationState{
		ID:               remoteId,
		TransformationID: transformationID,
	}

	newState, err := h.Update(ctx, data, oldData, oldState)
	if err != nil {
		return nil, fmt.Errorf("updating destination during import: %w", err)
	}

	if err := h.client.Destinations.SetExternalID(ctx, remoteId, data.ID); err != nil {
		return nil, fmt.Errorf("setting external ID for destination during import: %w", err)
	}

	return newState, nil
}

// FormatForExport converts unmanaged remote destinations into importable YAML
// specs: config is converted to local snake_case, registered secret keys are
// masked with per-resource placeholders, and a linked transformation resolves
// to a "#transformation:<id>" reference (failing the export if the link can't
// be resolved — mirrors the source handler, no silent fallback to a raw ID).
func (h *HandlerImpl) FormatForExport(
	collection map[string]*RemoteDestination,
	_ namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	if len(collection) == 0 {
		return nil, nil, nil
	}

	var (
		entities []writer.FormattableEntity
		entries  []importmanifest.ImportEntry
	)

	for externalID, remote := range collection {
		specMap, err := h.toExportSpecMap(externalID, remote, inputResolver)
		if err != nil {
			return nil, nil, err
		}

		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: remote.WorkspaceID,
			Resources: []specs.ImportIds{
				{
					URN:      resources.URN(externalID, DestinationResourceType),
					RemoteID: remote.ID,
				},
			},
		}
		entries = append(entries, handlers.ImportEntriesFromWorkspace(workspaceMetadata)...)

		spec, err := handlers.ToImportSpec(
			DestinationSpecKind,
			DestinationMetadataName,
			workspaceMetadata,
			specMap,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("creating spec for destination %s: %w", remote.ID, err)
		}

		entities = append(entities, writer.FormattableEntity{
			Content: spec,
			RelativePath: filepath.Join(
				"destinations",
				fmt.Sprintf("%s.yaml", externalID),
			),
		})
	}

	return entities, entries, nil
}

// toExportSpecMap builds the "spec" section of an importable destination's
// YAML: local config with secrets masked, plus an optional transformation ref.
func (h *HandlerImpl) toExportSpecMap(externalID string, remote *RemoteDestination, inputResolver resolver.ReferenceResolver) (map[string]any, error) {
	registered, err := h.registry.GetByAPIType(remote.Type, remote.Version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition for %s: %w", remote.ID, err)
	}

	localConfig, err := h.apiConfigToLocal(registered.Type, remote.Version, remote.Config)
	if err != nil {
		return nil, fmt.Errorf("converting destination %s config to local: %w", remote.ID, err)
	}

	if err := maskSecrets(localConfig, externalID, registered.SecretKeys()); err != nil {
		return nil, fmt.Errorf("masking destination %s secrets: %w", remote.ID, err)
	}

	specMap := map[string]any{
		"id":                 externalID,
		"display_name":       remote.Name,
		"type":               registered.Type,
		"enabled":            remote.IsEnabled,
		"definition_version": remote.Version,
		"config":             localConfig,
	}

	if remote.Transformation != nil && remote.Transformation.ID != "" {
		ref, err := inputResolver.ResolveToReference(
			ttypes.TransformationResourceType,
			remote.Transformation.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("resolving transformation reference for destination %s: %w", remote.ID, err)
		}
		specMap["transformation"] = ref
	}

	return specMap, nil
}

// maskSecrets replaces each registered secret key present in config with the
// marshaled form of secret.NewUnknown(WithVariableName(...)): a "{{ .VAR }}"
// token when enableVarSubstitution is on, otherwise the masked literal.
// Only keys present in config are touched — absent secrets are not invented.
func maskSecrets(config map[string]any, externalID string, secretKeys []string) error {
	if config == nil || len(secretKeys) == 0 {
		return nil
	}

	prefix := strings.ToUpper(strings.ReplaceAll(externalID, "-", "_"))
	for _, key := range secretKeys {
		varName := fmt.Sprintf(
			"%s_%s",
			prefix,
			strings.ToUpper(key),
		)

		s := secret.NewUnknown(secret.WithVariableName(varName))
		token, err := marshalSecretToken(s)
		if err != nil {
			return fmt.Errorf("masking secret key %q: %w", key, err)
		}

		config[key] = token
	}
	return nil
}

// marshalSecretToken JSON-marshals a secret.String to its export string form
// (variable reference or masked literal).
func marshalSecretToken(s secret.String) (string, error) {
	bytes, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	var token string
	if err := json.Unmarshal(bytes, &token); err != nil {
		return "", err
	}
	return token, nil
}

// wrapKnownSecrets wraps every registered secret key as *secret.String with
// the known local value (empty string when the key is absent). Pointer form
// survives the differ's struct→map decode.
func wrapKnownSecrets(config map[string]any, secretKeys []string) map[string]any {
	if len(secretKeys) == 0 {
		return config
	}
	if config == nil {
		config = map[string]any{}
	}
	for _, key := range secretKeys {
		raw := ""
		if v, ok := config[key]; ok {
			if s, ok := v.(string); ok {
				raw = s
			}
		}
		s := secret.New(raw)
		config[key] = &s
	}
	return config
}

// wrapUnknownSecrets marks every registered secret key as an unknown
// *secret.String. Used when mapping remote state: the API never returns
// secret values, so presence is undetectable.
func wrapUnknownSecrets(config map[string]any, secretKeys []string) map[string]any {
	if len(secretKeys) == 0 {
		return config
	}
	if config == nil {
		config = map[string]any{}
	}
	for _, key := range secretKeys {
		s := secret.NewUnknown()
		config[key] = &s
	}
	return config
}

// revealSecrets returns a shallow copy of config with registered secret keys
// replaced by their Reveal() string. Must run before LocalToAPI / json.Marshal
// so the real value reaches the wire instead of a masked form.
func revealSecrets(config map[string]any, secretKeys []string) map[string]any {
	if config == nil || len(secretKeys) == 0 {
		return config
	}
	out := maps.Clone(config)
	for _, key := range secretKeys {
		v, ok := out[key]
		if !ok {
			continue
		}
		switch s := v.(type) {
		case *secret.String:
			if s == nil {
				out[key] = ""
				continue
			}
			out[key] = s.Reveal()
		case secret.String:
			out[key] = s.Reveal()
		}
	}
	return out
}

// localConfigToAPI resolves the registered definition and converts snake_case
// config to the camelCase form the API expects, returning JSON-serializable bytes.
func (h *HandlerImpl) localConfigToAPI(destType string, version int64, local map[string]any) (json.RawMessage, error) {
	registered, err := h.registry.Get(destType, version)
	if err != nil {
		return nil, fmt.Errorf("getting destination definition: %w", err)
	}

	// Reveal before conversion: LocalToAPI json.Marshals the map, and a
	// surviving secret.String would emit its masked form to the API.
	revealed := revealSecrets(
		local,
		registered.SecretKeys(),
	)

	apiConfig, err := registered.LocalToAPI(revealed)
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
// (ErrRemoteResourceExternalIdNotFound) both the ref and the remote ID are
// dropped so the CLI never persists or touches a link it doesn't own —
// mirrors the event-stream source handler, which does not persist a foreign
// tracking plan ID into state.
func (h *HandlerImpl) transformationRef(remote *RemoteDestination, urnResolver handler.URNResolver) (*resources.PropertyRef, string, error) {
	if remote.Transformation == nil || remote.Transformation.ID == "" {
		return nil, "", nil
	}

	transformationID := remote.Transformation.ID
	urn, err := urnResolver.GetURNByID(
		ttypes.TransformationResourceType,
		transformationID,
	)

	if err != nil {
		if err == resources.ErrRemoteResourceExternalIdNotFound {
			// Linked via UI/API and not managed by the CLI yet: drop both the ref
			// and the ID so a later unrelated Update doesn't disconnect the link.
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("resolving transformation URN: %w", err)
	}

	return createTransformationRef(urn), transformationID, nil
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
		if err := h.client.Destinations.DisconnectTransformation(ctx, destinationID); err != nil {
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
