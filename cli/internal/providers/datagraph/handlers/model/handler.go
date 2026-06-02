package model

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type ModelHandler = handler.BaseHandler[struct{}, dgModel.ModelResource, dgModel.ModelState, dgModel.RemoteModel]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "data-graph-model",
	SpecKind:         "data-graph", // Models are inline in data-graph specs
	SpecMetadataName: "data-graph",
}

// HandlerImpl implements the HandlerImpl interface for model resources
// Note: Models don't have their own spec kind - they're inline in data-graph specs
// The provider handles all spec parsing and resource extraction
type HandlerImpl struct {
	client dgClient.DataGraphClient
}

// NewHandler creates a new BaseHandler for model resources
func NewHandler(client dgClient.DataGraphClient) *ModelHandler {
	h := &HandlerImpl{client: client}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *struct{} {
	return &struct{}{}
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *struct{}) (map[string]*dgModel.ModelResource, error) {
	// Models don't have standalone specs - they're inline in data-graph specs
	// The provider handles all resource extraction
	return nil, fmt.Errorf("model handler does not support standalone spec extraction - models are inline in data-graph specs")
}

// listAllModelsForDataGraph fetches all models (entity and event) for a data graph
func (h *HandlerImpl) listAllModelsForDataGraph(ctx context.Context, dataGraphID string, hasExternalID *bool) ([]*dgModel.RemoteModel, error) {
	var allModels []*dgModel.RemoteModel
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListModels(ctx, &dgClient.ListModelsRequest{DataGraphID: dataGraphID, Page: page, PageSize: perPage, HasExternalID: hasExternalID})
		if err != nil {
			return nil, fmt.Errorf("listing models: %w", err)
		}

		for i := range resp.Data {
			allModels = append(allModels, &dgModel.RemoteModel{
				Model: &resp.Data[i],
			})
		}

		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allModels, nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*dgModel.RemoteModel, error) {
	hasExternalID := true

	// First, get all data graphs with external IDs
	dataGraphs, err := h.listAllDataGraphs(ctx, &hasExternalID)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all models with external IDs
	var allModels []*dgModel.RemoteModel
	for _, dg := range dataGraphs {
		models, err := h.listAllModelsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading models for data graph %s: %w", dg.ID, err)
		}
		if err := h.populateColumnMetadata(ctx, dg.ID, models); err != nil {
			return nil, fmt.Errorf("loading column metadata for data graph %s: %w", dg.ID, err)
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*dgModel.RemoteModel, error) {
	hasExternalID := false

	// Only fetch unmanaged data graphs — models under managed DGs are not importable
	dataGraphs, err := h.listAllDataGraphs(ctx, &hasExternalID)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all models without external IDs
	var allModels []*dgModel.RemoteModel
	for _, dg := range dataGraphs {
		models, err := h.listAllModelsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading importable models for data graph %s: %w", dg.ID, err)
		}
		if err := h.populateColumnMetadata(ctx, dg.ID, models); err != nil {
			return nil, fmt.Errorf("loading column metadata for data graph %s: %w", dg.ID, err)
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// populateColumnMetadata fetches the per-column metadata rows for each model
// and attaches them (sorted by Name) to the corresponding RemoteModel.
//
// The remote list endpoint already filters orphans against the cached schema,
// so the rows we attach here mirror what the user can author declaratively in
// yaml. This is the seam that makes a subsequent re-apply of an unchanged
// yaml-with-columns a no-op: MapRemoteToState lifts these rows into the
// resource's Columns field, the differ compares against the local resource's
// Columns, and the syncer short-circuits when they match.
//
// Error policy:
//   - HTTP 404 from /column-metadata is treated as "no rows yet" (the
//     endpoint exists for the model but holds no entries, or the model was
//     just created in a transient race). The model is left with empty
//     Columns.
//   - Any other error — including 5xx and transport failures — fails the
//     load so a real infrastructure issue isn't silently masked as a clean
//     remote state and then mis-diagnosed as a stale-yaml diff.
func (h *HandlerImpl) populateColumnMetadata(ctx context.Context, dataGraphID string, models []*dgModel.RemoteModel) error {
	for _, m := range models {
		resp, err := h.client.ListColumnMetadata(ctx, dataGraphID, m.ID)
		if err != nil {
			if isNotFound(err) {
				m.Columns = nil
				continue
			}
			return fmt.Errorf("listing column metadata for model %s: %w", m.ID, err)
		}

		if len(resp.Columns) == 0 {
			m.Columns = nil
			continue
		}

		rows := make([]dgClient.ColumnMetadataRow, len(resp.Columns))
		copy(rows, resp.Columns)
		slices.SortFunc(rows, func(a, b dgClient.ColumnMetadataRow) int {
			return cmp.Compare(a.Name, b.Name)
		})
		m.Columns = rows
	}
	return nil
}

// isNotFound reports whether err wraps a 404 response from the API client.
// Used to distinguish "no rows for this model" (treat as empty) from any
// other failure (which must propagate).
func isNotFound(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.HTTPStatusCode == http.StatusNotFound
}

// listAllDataGraphs fetches all data graphs with pagination
func (h *HandlerImpl) listAllDataGraphs(ctx context.Context, hasExternalID *bool) ([]*dgClient.DataGraph, error) {
	var allDataGraphs []*dgClient.DataGraph
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListDataGraphs(ctx, &dgClient.ListDataGraphsRequest{
			Page:          page,
			PageSize:      perPage,
			HasExternalID: hasExternalID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing data graphs: %w", err)
		}

		// Add all data graphs from current page
		for i := range resp.Data {
			allDataGraphs = append(allDataGraphs, &resp.Data[i])
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allDataGraphs, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *dgModel.RemoteModel, urnResolver handler.URNResolver) (*dgModel.ModelResource, *dgModel.ModelState, error) {
	// Skip resources without external IDs
	if remote.ExternalID == "" {
		return nil, nil, nil
	}

	// Resolve the data graph's URN from its remote ID
	dataGraphURN, err := urnResolver.GetURNByID(datagraph.HandlerMetadata.ResourceType, remote.DataGraphID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving data graph URN: %w", err)
	}

	// Create PropertyRef to the data graph using the resolved URN
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	resource := &dgModel.ModelResource{
		ID:           remote.ExternalID,
		DisplayName:  remote.Name,
		Type:         remote.Type,
		Table:        remote.TableRef,
		Description:  remote.Description,
		DataGraphRef: dataGraphRef,
		PrimaryID:    remote.PrimaryID,
		Root:         remote.Root,
		Timestamp:    remote.Timestamp,
		Columns:      columnsFromRemote(remote.Columns),
	}

	state := &dgModel.ModelState{
		ID: remote.ID,
	}

	return resource, state, nil
}

// columnsFromRemote lifts the server's column-metadata rows into the
// resource's diff-friendly map slice. The server's UpdatedAt is intentionally
// omitted: it's an authoring-visible artifact that would surface as a spurious
// diff against the local yaml on every apply. Returns nil for empty input so
// the resulting resource matches the "no columns:" yaml shape exactly.
func columnsFromRemote(rows []dgClient.ColumnMetadataRow) []map[string]any {
	if len(rows) == 0 {
		return nil
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		out[i] = map[string]any{
			"name":         row.Name,
			"display_name": row.DisplayName,
			"description":  row.Description,
		}
	}
	return out
}

func (h *HandlerImpl) Create(ctx context.Context, data *dgModel.ModelResource) (*dgModel.ModelState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value

	var remote *dgClient.Model
	var err error

	switch data.Type {
	case "entity":
		req := &dgClient.CreateModelRequest{
			DataGraphID: dataGraphRemoteID,
			Type:        "entity",
			Name:        data.DisplayName,
			Description: data.Description,
			TableRef:    data.Table,
			ExternalID:  data.ID,
			PrimaryID:   data.PrimaryID,
			Root:        data.Root,
		}
		remote, err = h.client.CreateModel(ctx, req)
	case "event":
		req := &dgClient.CreateModelRequest{
			DataGraphID: dataGraphRemoteID,
			Type:        "event",
			Name:        data.DisplayName,
			Description: data.Description,
			TableRef:    data.Table,
			ExternalID:  data.ID,
			Timestamp:   data.Timestamp,
		}
		remote, err = h.client.CreateModel(ctx, req)
	default:
		return nil, fmt.Errorf("invalid model type: %s", data.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("creating %s model: %w", data.Type, err)
	}

	// Create has no remote state, so no removals to compute — nil remoteColumns
	// short-circuits the diff and only upserts the local entries.
	if err := h.applyColumnMetadata(ctx, dataGraphRemoteID, remote.ID, data.Columns, nil); err != nil {
		return nil, err
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *dgModel.ModelResource, oldData *dgModel.ModelResource, oldState *dgModel.ModelState) (*dgModel.ModelState, error) {
	dataGraphRemoteID := oldData.DataGraphRef.Value

	var remote *dgClient.Model
	var err error

	switch newData.Type {
	case "entity":
		req := &dgClient.UpdateModelRequest{
			DataGraphID: dataGraphRemoteID,
			ModelID:     oldState.ID,
			Type:        "entity",
			Name:        newData.DisplayName,
			Description: newData.Description,
			TableRef:    newData.Table,
			PrimaryID:   newData.PrimaryID,
			Root:        newData.Root,
		}
		remote, err = h.client.UpdateModel(ctx, req)
	case "event":
		req := &dgClient.UpdateModelRequest{
			DataGraphID: dataGraphRemoteID,
			ModelID:     oldState.ID,
			Type:        "event",
			Name:        newData.DisplayName,
			Description: newData.Description,
			TableRef:    newData.Table,
			Timestamp:   newData.Timestamp,
		}
		remote, err = h.client.UpdateModel(ctx, req)
	default:
		return nil, fmt.Errorf("invalid model type: %s", newData.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("updating %s model: %w", newData.Type, err)
	}

	// Update threads oldData.Columns through as the pre-apply remote state so
	// the handler can compute removals (names present remotely but no longer
	// in yaml) and send them in the same PATCH as the upserts. MapRemoteToState
	// is the seam that populates oldData.Columns from the server's rows.
	if err := h.applyColumnMetadata(ctx, dataGraphRemoteID, remote.ID, newData.Columns, oldData.Columns); err != nil {
		return nil, err
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

// applyColumnMetadata reconciles the local yaml columns block against the
// pre-apply remote state and sends one PATCH per model carrying both upserts
// and removals. The yaml is declarative: any column name present remotely but
// missing from local yaml goes into deleteColumns; the remaining entries go
// into columns as (name, displayName) pairs. The server applies the diff
// atomically — sets and clears land in the same transaction. The model commit
// is not rolled back if this call fails; the wrapped error surfaces to the
// apply user with the original cause intact.
func (h *HandlerImpl) applyColumnMetadata(
	ctx context.Context,
	dataGraphID, modelID string,
	localColumns []map[string]any,
	remoteColumns []map[string]any,
) error {
	entries := make([]dgClient.ColumnMetadataEntry, 0, len(localColumns))
	localNames := make(map[string]struct{}, len(localColumns))
	for _, col := range localColumns {
		name, _ := col["name"].(string)
		displayName, _ := col["display_name"].(string)
		description, _ := col["description"].(string)
		// Declarative mapping: a field present in yaml is set; a field absent
		// (empty here) is cleared by sending JSON null (nil pointer). The local
		// validator guarantees at least one is non-empty.
		entry := dgClient.ColumnMetadataEntry{Name: name}
		if displayName != "" {
			dn := displayName
			entry.DisplayName = &dn
		}
		if description != "" {
			desc := description
			entry.Description = &desc
		}
		entries = append(entries, entry)
		if name != "" {
			localNames[name] = struct{}{}
		}
	}

	var deleteColumns []string
	for _, col := range remoteColumns {
		name, ok := col["name"].(string)
		if !ok || name == "" {
			continue
		}
		if _, kept := localNames[name]; kept {
			continue
		}
		deleteColumns = append(deleteColumns, name)
	}
	// Sort for deterministic ordering — important for both wire-payload
	// equivalence in tests and idempotency of repeated applies.
	slices.Sort(deleteColumns)

	if len(entries) == 0 && len(deleteColumns) == 0 {
		return nil
	}

	if _, err := h.client.BatchUpsertColumnMetadata(ctx, dataGraphID, modelID, dgClient.BatchUpsertColumnMetadataRequest{
		Columns:       entries,
		DeleteColumns: deleteColumns,
	}); err != nil {
		return fmt.Errorf("batch-upsert column metadata: %w", err)
	}

	return nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *dgModel.ModelResource, remoteID string) (*dgModel.ModelState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value

	// Set external ID on the remote resource and get the updated model
	remote, setErr := h.client.SetModelExternalID(ctx, &dgClient.SetModelExternalIDRequest{
		DataGraphID: dataGraphRemoteID,
		ModelID:     remoteID,
		ExternalID:  data.ID,
	})
	if setErr != nil {
		return nil, fmt.Errorf("setting external ID: %w", setErr)
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *dgModel.ModelResource, oldState *dgModel.ModelState) error {
	dataGraphRemoteID := oldData.DataGraphRef.Value

	err := h.client.DeleteModel(ctx, &dgClient.DeleteModelRequest{
		DataGraphID: dataGraphRemoteID,
		ModelID:     oldState.ID,
	})
	if err != nil {
		return fmt.Errorf("deleting %s model: %w", oldData.Type, err)
	}

	return nil
}

// FormatForExport is a no-op — export is handled at the provider level for composite specs
func (h *HandlerImpl) FormatForExport(
	collection map[string]*dgModel.RemoteModel,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	return nil, nil
}

// CreateModelReference creates a PropertyRef that points to a model's remote ID
func CreateModelReference(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(
		urn,
		func(state *dgModel.ModelState) (string, error) {
			return state.ID, nil
		},
	)
}
