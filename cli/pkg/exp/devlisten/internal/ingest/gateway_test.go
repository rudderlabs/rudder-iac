package ingest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

var testNow = time.Date(2026, 9, 29, 12, 0, 0, 123456000, time.UTC)

func newTestGateway() (*Gateway, *store.Store) {
	st := store.New("0123456789abcdef")
	g := New(st, testNow)
	g.now = func() time.Time { return testNow }
	g.newUUID = func() string { return "11111111-2222-4333-8444-555555555555" }
	return g, st
}

type sent struct {
	status int
	header http.Header
	body   string
}

func send(t *testing.T, g *Gateway, r *http.Request) sent {
	t.Helper()
	rec := httptest.NewRecorder()
	g.ServeHTTP(rec, r)
	body, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)
	return sent{status: rec.Code, header: rec.Header(), body: string(body)}
}

func post(path, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.SetBasicAuth("dev", "")
	r.RemoteAddr = "127.0.0.1:50000"
	return r
}

func onlyRecord(t *testing.T, st *store.Store) store.Record {
	t.Helper()
	records := st.Since(0).Records
	require.Len(t, records, 1)
	return records[0]
}

func str(s string) *string { return &s }

func TestTrackIsAcceptedAndCaptured(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	got := send(t, g, post("/v1/track", `{"event":"Order Completed","userId":"u1","properties":{"total":42}}`))

	require.Equal(t, http.StatusOK, got.status)
	require.Equal(t, "ok", got.body)

	rec := onlyRecord(t, st)
	require.Equal(t, "ingestion", rec.Kind)
	require.Equal(t, "/v1/track", rec.Route)
	require.Equal(t, "http", rec.Transport)
	require.Equal(t, 200, rec.StatusCode)
	require.Equal(t, "accepted", rec.Outcome)
	require.False(t, rec.Failed)
	require.Equal(t, "dev", rec.WriteKey)
	require.Equal(t, "dev-ef260e9aa3c6", rec.SourceID)
	require.Equal(t, testNow, rec.ReceivedAt)
	require.Len(t, rec.Events, 1)

	ev := rec.Events[0]
	require.Equal(t, str("track"), ev.Type)
	require.Equal(t, str("Order Completed"), ev.Event)
	require.Equal(t, str("u1"), ev.UserID)
	require.Nil(t, ev.AnonymousID)
	require.Equal(t, str("11111111-2222-4333-8444-555555555555"), ev.MessageID)
	require.JSONEq(t, `{"event":"Order Completed","userId":"u1","properties":{"total":42}}`, string(ev.Message))

	var enriched map[string]any
	require.NoError(t, json.Unmarshal(ev.EnrichedMessage, &enriched))
	require.Equal(t, map[string]any{
		"type":       "track",
		"event":      "Order Completed",
		"userId":     "u1",
		"properties": map[string]any{"total": float64(42)},
		"messageId":  "11111111-2222-4333-8444-555555555555",
		"receivedAt": "2026-09-29T12:00:00.123456Z",
		"request_ip": "127.0.0.1",
		"rudderId":   "5eff0fc3-0944-48fa-9c04-38f62e3890ca",
	}, enriched)
}

func TestMissingWriteKeyIsRejectedWithOracleBody(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := post("/v1/track", `{"userId":"u1"}`)
	r.Header.Del("Authorization")

	got := send(t, g, r)

	require.Equal(t, http.StatusUnauthorized, got.status)
	require.Equal(t, "failed to read writekey from header\n", got.body)
	require.Equal(t, "text/plain; charset=utf-8", got.header.Get("Content-Type"))
	require.Equal(t, "nosniff", got.header.Get("X-Content-Type-Options"))

	rec := onlyRecord(t, st)
	require.True(t, rec.Failed)
	require.Equal(t, "rejected", rec.Outcome)
	require.Equal(t, &store.Rejection{Stage: "auth", Reason: "failed to read writekey from header"}, rec.Rejection)
	require.Empty(t, rec.Events)
}

func TestBatchCapturesEachEventWithoutTouchingType(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	got := send(t, g, post("/v1/batch", `{"batch":[{"type":"identify","userId":"u1"},{"type":"track","event":"A","anonymousId":"a1"}],"sentAt":"2026-09-29T12:00:00.100Z"}`))

	require.Equal(t, http.StatusOK, got.status)
	rec := onlyRecord(t, st)
	require.Len(t, rec.Events, 2)
	require.Equal(t, 1, rec.Events[1].Idx)
	require.Equal(t, str("identify"), rec.Events[0].Type)
	require.Equal(t, str("track"), rec.Events[1].Type)
	require.Equal(t, str("a1"), rec.Events[1].AnonymousID)
	require.JSONEq(t, `{"type":"track","event":"A","anonymousId":"a1"}`, string(rec.Events[1].Message))
}

// A browser cannot set User-Agent, so the probe page marks its requests by
// being their referrer.
func TestRequestsFromTheProbePageAreProbes(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := post("/v1/track", `{"event":"dev probe","anonymousId":"a1","properties":{"probe":true}}`)
	r.Header.Set("Referer", "http://127.0.0.1:4321/_dev/v1/probe.html")

	send(t, g, r)

	require.True(t, onlyRecord(t, st).Probe)
}
