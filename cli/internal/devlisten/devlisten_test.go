package devlisten

import (
	"bytes"
	"compress/gzip"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func send(t *testing.T, method, url, writeKey string, body []byte, header http.Header) string {
	t.Helper()
	r, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	maps.Copy(r.Header, header)
	r.SetBasicAuth(writeKey, "")
	resp, err := http.DefaultClient.Do(r)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(got)
}

// The store holds the body and each event as the bytes the SDK sent, with the
// key masked in the record, the target and the response.
func TestHandlerCapturesIntoTheStore(t *testing.T) {
	t.Parallel()
	const key = "fake-write-key-for-tests"
	h, s := NewHandler(nil, "1.12.0")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	events := []string{
		`{ "type":"track", "type":"track", "event":"Order Completed", "userId":"u1", "properties":{"total":1.0,"n":1e2,"s":"é"} }`,
		`{"type":"identify","anonymousId":"a1","traits":{"k":1,"k":2}}`,
	}
	batch := `{"batch":[` + strings.Join(events, " ,\n") + `],"sentAt":"2026-09-30T12:00:00.000Z"}`
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err := zw.Write([]byte(batch))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	require.Equal(t, "ok", send(t, http.MethodPost, srv.URL+"/v1/batch", key, gz.Bytes(),
		http.Header{"Content-Encoding": {"gzip"}, "Cookie": {"session=secret"}}))
	require.Contains(t, send(t, http.MethodGet, srv.URL+"/sourceConfig/?writeKey="+key, key, nil, nil), key)
	require.Equal(t, `{"status":"ready"}`, send(t, http.MethodGet, srv.URL+"/health", "", nil, nil))

	records, _ := s.Since(0)
	require.Len(t, records, 2, "health checks are not captured")

	ingestion := records[0]
	require.Equal(t, "ingestion", ingestion.Kind)
	require.Equal(t, "fake...ests", ingestion.WriteKey.Key)
	require.Equal(t, batch, string(ingestion.Request.Body))
	require.Equal(t, gz.Len(), ingestion.Request.BodyBytes)
	require.Equal(t, "gzip", ingestion.Request.Headers["Content-Encoding"])
	require.NotContains(t, ingestion.Request.Headers, "Authorization")
	require.NotContains(t, ingestion.Request.Headers, "Cookie")
	require.Len(t, ingestion.Events, len(events))
	for i, want := range events {
		require.Equal(t, want, string(ingestion.Events[i].Message))
		require.NotNil(t, ingestion.Events[i].Enrichment)
	}

	control := records[1]
	require.Equal(t, "control", control.Kind)
	require.Equal(t, "/sourceConfig/", control.Route)
	require.NotContains(t, control.Request.Target, key)
	require.NotContains(t, control.Response.Body, key)
	require.Contains(t, control.Response.Body, `"writeKey":"fake...ests"`)
}
