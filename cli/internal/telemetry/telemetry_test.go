package telemetry

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	version      = "1.2.3"
	anonymousID  = "test-anonymous-id"
	writeKey     = "test-write-key"
	dataplaneURL = "test-dataplane-url"
)

func TestTelemetryTrack(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	tel, err := newTelemetry(version, anonymousID, writeKey, dataplaneURL, withTransport(transport))
	require.NoError(t, err)

	err = tel.Track("test-event", map[string]any{"key": "value"})
	require.NoError(t, err)

	if len(transport.requests) == 0 {
		t.Errorf("Expected at least one request to be sent, got 0")
	}

	expectedStrings := []string{
		`"anonymousId":"test-anonymous-id"`,
		`"app":{"name":"rudder-cli","version":"1.2.3"}`,
		`"properties":{"key":"value"}`,
	}

	encodedWriteKey := base64.StdEncoding.EncodeToString([]byte(writeKey + ":"))
	expectedHeaders := map[string]string{
		"Authorization": "Basic " + encodedWriteKey,
	}

	validateRequest(t, transport.requests[0], expectedStrings, expectedHeaders)
}

func TestTelemetryTrackTimeout(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	transport.delay = 2 * time.Millisecond
	tel, err := newTelemetry(version, anonymousID, writeKey, dataplaneURL, withTransport(transport), WithTimeout(time.Millisecond))
	require.NoError(t, err)

	err = tel.Track("test-event-timeout", map[string]any{"key": "value"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout")
}

type mockTransport struct {
	delay    time.Duration
	requests []*http.Request
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		requests: []*http.Request{},
	}
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Return a proper 200 OK response
	return &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func validateRequest(t *testing.T, req *http.Request, expectedStrings []string, expectedHeaders map[string]string) {
	t.Helper()

	// Get the request body
	body, err := req.GetBody()
	require.NoError(t, err)

	// Decompress the gzip body
	gzipReader, err := gzip.NewReader(body)
	require.NoError(t, err)
	defer gzipReader.Close()

	data, err := io.ReadAll(gzipReader)
	require.NoError(t, err)

	for _, str := range expectedStrings {
		assert.Contains(t, string(data), str, "Expected string %q in request body", str)
	}

	for key, expectedValue := range expectedHeaders {
		actualValue := req.Header.Get(key)
		assert.Equal(t, expectedValue, actualValue, "Header %q mismatch", key)
	}
}
