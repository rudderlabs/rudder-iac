package ingest

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnrichmentMatchesRudderStack(t *testing.T) {
	t.Parallel()
	const (
		receivedAt = "2026-09-30T12:00:00.5Z"
		// rudderIDs from rudder-go-kit uuid.GetMD5UUID.
		u1a1 = "f81ebcce-dafc-4607-b906-29e68e64b35f"
		u1   = "5eff0fc3-0944-48fa-9c04-38f62e3890ca"
	)
	withXFF := func(value string) func(*http.Request) { return header("X-Forwarded-For", value) }

	for _, tc := range []struct {
		name    string
		request *http.Request
		want    []*Enrichment
	}{
		{
			name:    "single-event route sets type, messageId and request_ip",
			request: req("POST", "/v1/track", `{"userId":"u1","type":"page"}`),
			want:    []*Enrichment{{MessageID: "generated", ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: u1, Type: "track"}},
		},
		{
			name:    "batch keeps what the SDK sent",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":"u1","anonymousId":"a1","messageId":"m1","request_ip":"10.0.0.1"}]}`),
			want:    []*Enrichment{{ReceivedAt: receivedAt, RudderID: u1a1}},
		},
		{
			name:    "messageId is sanitized",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":"u1","messageId":" m1​"},{"userId":"u1","messageId":"​"}]}`),
			want: []*Enrichment{
				{MessageID: "m1", ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: u1},
				{MessageID: "generated", ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: u1},
			},
		},
		{
			name:    "the last duplicate key wins",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":"x","userId":"u1","anonymousId":"a1","messageId":"m1"}]}`),
			want:    []*Enrichment{{ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: u1a1}},
		},
		{
			name:    "first X-Forwarded-For entry",
			request: req("POST", "/v1/identify", `{"userId":"u1","messageId":"m1"}`, withXFF(" 203.0.113.7 , 10.0.0.1")),
			want:    []*Enrichment{{ReceivedAt: receivedAt, RequestIP: "203.0.113.7", RudderID: u1, Type: "identify"}},
		},
		{
			name:    "long X-Forwarded-For is cut",
			request: req("POST", "/v1/identify", `{"userId":"u1","messageId":"m1"}`, withXFF(strings.Repeat("1", 40_000))),
			want:    []*Enrichment{{ReceivedAt: receivedAt, RequestIP: strings.Repeat("1", maxRequestIPLen), RudderID: u1, Type: "identify"}},
		},
		{
			name:    "pixel",
			request: req("GET", "/pixel/v1/page?writeKey=dev&anonymousId=a1&userId=u1&messageId=m1", ""),
			want:    []*Enrichment{{ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: u1a1, Type: "page"}},
		},
		{
			name:    "numeric identity after a float64 round trip",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":9007199254740993,"messageId":"m1"},{"anonymousId":1.0e21,"messageId":"m2"}]}`),
			want: []*Enrichment{
				{ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: "c2425f96-b965-4d3e-92f3-2dd2cfd4fdc9"},
				{ReceivedAt: receivedAt, RequestIP: "192.0.2.1", RudderID: rudderID("", "1e+21")},
			},
		},
		{
			name:    "rejected request is not enriched",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":"u1"},{"event":"A"}]}`),
			want:    []*Enrichment{nil, nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, sink := newTestHandler()
			h.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 500_000_000, time.UTC) }
			h.newUUID = func() string { return "generated" }

			serve(h, tc.request)

			var got []*Enrichment
			for _, e := range sink.only(t).Events {
				got = append(got, e.Enrichment)
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRequestIPFallsBack(t *testing.T) {
	t.Parallel()
	for remoteAddr, want := range map[string]string{
		"192.0.2.1:1234": "192.0.2.1",
		"[::1]:1234":     "[::1]",
		"pipe":           "0.0.0.0",
	} {
		r := httptest.NewRequest(http.MethodPost, "/v1/track", nil)
		r.RemoteAddr = remoteAddr
		require.Equal(t, want, requestIP(r), remoteAddr)
	}
}

// Not parallel: it measures the heap.
func TestEnrichmentKeepsNoPartOfTheRequest(t *testing.T) {
	heap := func() uint64 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	h, sink := newTestHandler()
	before := heap()

	serve(h, req("POST", "/v1/track", `{"userId":"u1","messageId":" m1 ","p":"`+strings.Repeat("a", 2_000_000)+`"}`,
		header("X-Forwarded-For", strings.Repeat("1", 1<<20))))
	e := sink.only(t).Events[0].Enrichment
	sink.captures = nil

	require.Less(t, int64(heap())-int64(before), int64(256<<10))
	require.Equal(t, "m1", e.MessageID)
	require.Len(t, e.RequestIP, maxRequestIPLen)
}
