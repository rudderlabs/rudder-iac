package previewer

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stdoutMu sync.Mutex

type mockPreviewProvider struct {
	rows         []map[string]any
	err          error
	called       bool
	calledID     string
	calledType   string
	calledData   resources.ResourceData
	calledLimit  int
}

func (m *mockPreviewProvider) Preview(_ context.Context, ID string, resourceType string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	m.called = true
	m.calledID = ID
	m.calledType = resourceType
	m.calledData = data
	m.calledLimit = limit
	return m.rows, m.err
}

func TestNewAndOptions(t *testing.T) {
	provider := &mockPreviewProvider{}
	p := New(provider, WithLimit(20), WithJson(true), WithInteractive(true))

	require.NotNil(t, p)
	assert.Equal(t, provider, p.Provider)
	assert.Equal(t, 20, p.Limit)
	assert.True(t, p.Json)
	assert.True(t, p.Interactive)
}

func TestPreview_ProviderError(t *testing.T) {
	expectedErr := errors.New("preview failed")
	provider := &mockPreviewProvider{err: expectedErr}
	p := New(provider, WithLimit(7))

	err := p.Preview(context.Background(), "src-1", "sources", resources.ResourceData{"a": 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.True(t, provider.called)
	assert.Equal(t, "src-1", provider.calledID)
	assert.Equal(t, "sources", provider.calledType)
	assert.Equal(t, resources.ResourceData{"a": 1}, provider.calledData)
	assert.Equal(t, 7, provider.calledLimit)
}

func TestPreview_EmptyRows(t *testing.T) {
	provider := &mockPreviewProvider{rows: []map[string]any{}}
	p := New(provider)

	out := captureStdout(t, func() {
		err := p.Preview(context.Background(), "src-1", "sources", resources.ResourceData{})
		require.NoError(t, err)
	})

	assert.Contains(t, out, "No preview data available")
}

func TestPreview_JsonPath(t *testing.T) {
	provider := &mockPreviewProvider{rows: []map[string]any{{"id": "abc", "count": 2}}}
	p := New(provider, WithJson(true))

	out := captureStdout(t, func() {
		err := p.Preview(context.Background(), "src-1", "sources", resources.ResourceData{})
		require.NoError(t, err)
	})

	assert.Contains(t, out, "\"id\": \"abc\"")
	assert.Contains(t, out, "\"count\": 2")
}

func TestPreview_TablePath(t *testing.T) {
	provider := &mockPreviewProvider{rows: []map[string]any{{"id": "abc", "name": "book"}}}
	p := New(provider, WithInteractive(false))

	out := captureStdout(t, func() {
		err := p.Preview(context.Background(), "src-1", "sources", resources.ResourceData{})
		require.NoError(t, err)
	})

	assert.Contains(t, out, "abc")
	assert.Contains(t, out, "book")
}

func TestPreviewJson_MarshalError(t *testing.T) {
	p := New(&mockPreviewProvider{})
	err := p.previewJson([]map[string]any{{"bad": math.NaN()}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value")
}

func TestPreviewTable_NonInteractivePrintsView(t *testing.T) {
	p := New(&mockPreviewProvider{}, WithInteractive(false))

	out := captureStdout(t, func() {
		err := p.previewTable([]map[string]any{
			{"id": "id-1", "name": "alpha"},
			{"id": "id-2", "name": ""},
		})
		require.NoError(t, err)
	})

	assert.True(t, strings.Contains(out, "id-1") || strings.Contains(out, "id-2"))
	assert.Contains(t, out, "name")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = orig

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(b)
}
