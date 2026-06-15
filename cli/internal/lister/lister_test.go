package lister

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stdoutMu sync.Mutex

type mockListProvider struct {
	resources    []resources.ResourceData
	err          error
	resourceType string
	filters      Filters
}

func (m *mockListProvider) List(ctx context.Context, resourceType string, filters Filters) ([]resources.ResourceData, error) {
	m.resourceType = resourceType
	m.filters = filters
	return m.resources, m.err
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return buf.String()
}

func TestNew(t *testing.T) {
	t.Parallel()

	provider := &mockListProvider{}
	widths := map[string]int{"id": 20, "name": 25}

	l := New(provider, WithFormat(JSONFormat), WithColumnWidths(widths))

	assert.Equal(t, provider, l.Provider)
	assert.Equal(t, JSONFormat, l.Format)
	assert.Equal(t, widths, l.ColumnWidths)
}

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	provider := &mockListProvider{}
	l := New(provider)

	assert.Equal(t, TableFormat, l.Format)
	assert.Empty(t, l.ColumnWidths)
}

func TestPrintResourcesAsJSON(t *testing.T) {
	resources := []resources.ResourceData{
		{"id": "src-1", "name": "Source One"},
		{"id": "src-2", "name": "Source Two"},
	}

	output := captureStdout(t, func() {
		err := printResourcesAsJSON(resources)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"id":"src-1"`)
	assert.Contains(t, output, `"name":"Source One"`)
	assert.Contains(t, output, `"id":"src-2"`)
}

func TestPrintResourcesAsJSON_Empty(t *testing.T) {
	output := captureStdout(t, func() {
		err := printResourcesAsJSON(nil)
		require.NoError(t, err)
	})

	assert.Empty(t, output)
}

func TestLister_List(t *testing.T) {
	t.Parallel()

	t.Run("JSON format prints resources", func(t *testing.T) {
		provider := &mockListProvider{
			resources: []resources.ResourceData{
				{"id": "tp-1", "name": "Tracking Plan"},
			},
		}
		l := New(provider, WithFormat(JSONFormat))

		output := captureStdout(t, func() {
			err := l.List(context.Background(), "tracking-plan", Filters{"name": "test"})
			require.NoError(t, err)
		})

		assert.Equal(t, "tracking-plan", provider.resourceType)
		assert.Equal(t, Filters{"name": "test"}, provider.filters)
		assert.Contains(t, output, `"id":"tp-1"`)
	})

	t.Run("provider error is returned", func(t *testing.T) {
		t.Parallel()

		listErr := errors.New("list failed")
		provider := &mockListProvider{err: listErr}
		l := New(provider, WithFormat(JSONFormat))

		err := l.List(context.Background(), "source", nil)
		assert.Equal(t, listErr, err)
	})

	t.Run("unknown format returns error", func(t *testing.T) {
		t.Parallel()

		provider := &mockListProvider{
			resources: []resources.ResourceData{{"id": "src-1"}},
		}
		l := New(provider, WithFormat(OutputFormat("invalid")))

		err := l.List(context.Background(), "source", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown output format")
	})
}
