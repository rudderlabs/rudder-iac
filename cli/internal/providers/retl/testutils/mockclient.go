package testutils

import (
	"context"
	"fmt"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// mockRETLStore mocks the RETL client for testing
type mockRETLStore struct {
	retlClient.RETLStore
	sourceID string
	// Adding mock functions for RETL source operations
	CreateRetlSourceFunc func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error)
	UpdateRetlSourceFunc func(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error)
	DeleteRetlSourceFunc func(ctx context.Context, id string) error
	GetRetlSourceFunc    func(ctx context.Context, id string) (*retlClient.RETLSource, error)
	ListRetlSourcesFunc  func(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error)
	// Preview functions
	SubmitPreviewFunc    func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error)
	GetPreviewResultFunc func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error)
}

// Mock RETL source operations

func (m *mockRETLStore) CreateRetlSource(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
	if m.CreateRetlSourceFunc != nil {
		return m.CreateRetlSourceFunc(ctx, source)
	}
	return nil, nil
}

func (m *mockRETLStore) UpdateRetlSource(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
	if m.UpdateRetlSourceFunc != nil {
		return m.UpdateRetlSourceFunc(ctx, sourceID, source)
	}
	return nil, nil
}

func (m *mockRETLStore) DeleteRetlSource(ctx context.Context, id string) error {
	if m.DeleteRetlSourceFunc != nil {
		return m.DeleteRetlSourceFunc(ctx, id)
	}
	return nil
}

func (m *mockRETLStore) GetRetlSource(ctx context.Context, id string) (*retlClient.RETLSource, error) {
	if m.GetRetlSourceFunc != nil {
		return m.GetRetlSourceFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRETLStore) ListRetlSources(ctx context.Context, hasExternalID *bool) (*retlClient.RETLSources, error) {
	if m.ListRetlSourcesFunc != nil {
		return m.ListRetlSourcesFunc(ctx, hasExternalID)
	}
	return &retlClient.RETLSources{}, nil
}

// Preview methods
func (m *mockRETLStore) SubmitSourcePreview(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
	if m.SubmitPreviewFunc != nil {
		return m.SubmitPreviewFunc(ctx, request)
	}
	return &retlClient.PreviewSubmitResponse{ID: "req-123"}, nil
}

func (m *mockRETLStore) GetSourcePreviewResult(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
	if m.GetPreviewResultFunc != nil {
		return m.GetPreviewResultFunc(ctx, resultID)
	}
	return &retlClient.PreviewResultResponse{Status: retlClient.Completed}, nil
}

// NewDefaultMockClient creates a new mock client with default behavior
func NewDefaultMockClient() *mockRETLStore {
	return &mockRETLStore{
		sourceID: "test-source-id",
		CreateRetlSourceFunc: func(ctx context.Context, source *retlClient.RETLSourceCreateRequest) (*retlClient.RETLSource, error) {
			return &retlClient.RETLSource{
				ID:                   "test-source-id",
				SourceType:           source.SourceType,
				SourceDefinitionName: source.SourceDefinitionName,
				Name:                 source.Name,
				Config:               source.Config,
				AccountID:            source.AccountID,
				WorkspaceID:          "test-workspace-id",
			}, nil
		},
		UpdateRetlSourceFunc: func(ctx context.Context, sourceID string, source *retlClient.RETLSourceUpdateRequest) (*retlClient.RETLSource, error) {
			return &retlClient.RETLSource{
				SourceType:           "model",
				SourceDefinitionName: "postgres",
				Name:                 source.Name,
				Config:               source.Config,
				AccountID:            source.AccountID,
			}, nil
		},
		DeleteRetlSourceFunc: func(ctx context.Context, id string) error {
			return nil
		},
		GetRetlSourceFunc: func(ctx context.Context, id string) (*retlClient.RETLSource, error) {

			if id == "remote-id-not-found" {
				return nil, fmt.Errorf("retl source not found")
			}

			return &retlClient.RETLSource{
				ID:                   "remote-id",
				Name:                 "Imported Model",
				SourceType:           retlClient.ModelSourceType,
				SourceDefinitionName: "postgres",
				AccountID:            "acc123",
				IsEnabled:            true,
				Config:               retlClient.RETLSQLModelConfig{Description: "desc", PrimaryKey: "id", Sql: "SELECT * FROM t"},
			}, nil
		},
		SubmitPreviewFunc: func(ctx context.Context, request *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
			return &retlClient.PreviewSubmitResponse{
				ID: "req-123",
			}, nil
		},
		GetPreviewResultFunc: func(ctx context.Context, resultID string) (*retlClient.PreviewResultResponse, error) {
			return &retlClient.PreviewResultResponse{
				Status: retlClient.Completed,
				Rows:   []map[string]any{{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}},
			}, nil
		},
	}
}
