package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// browserTrack carries the context an analytics-js browser event sends: SDK
// auto-collected keys, session keys, traits and one app-set key.
func browserTrack(idx int, name string) store.Event {
	msg := `{"type":"track","event":"` + name + `","userId":"u1","properties":{"n":1},` +
		`"context":{"app":{"name":"x"},"library":{"name":"rs"},"page":{"path":"/chat"},"sessionId":1,` +
		`"traits":{"plan":"pro"},"appEnvironment":"dev"}}`
	return store.Event{Idx: idx, Type: sp("track"), Event: sp(name), UserID: sp("u1"), MessageID: sp("m"),
		Message:         json.RawMessage(msg),
		EnrichedMessage: json.RawMessage(`{"request_ip":"127.0.0.1","rudderId":"r-1","type":"track"}`)}
}

func firstItem(t *testing.T, page map[string]any) map[string]any {
	t.Helper()
	events := page["events"].([]any)
	require.NotEmpty(t, events)
	return events[0].(map[string]any)
}

func TestCompactIsTheDefaultView(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?event=A")

	require.Equal(t, "compact", page["view"])
	item := firstItem(t, page)
	require.NotContains(t, item, "message")
	require.NotContains(t, item, "enrichedMessage")
	require.Equal(t, map[string]any{"appEnvironment": "dev"}, item["context"], "only the app-set residual stays")
	require.Equal(t, map[string]any{"plan": "pro"}, item["traits"])
	require.Equal(t, map[string]any{
		"fields":  []any{"message", "enrichedMessage"},
		"context": []any{"app", "library", "page", "sessionId", "traits"},
		"next":    "rudder-cli dev events list --since 0 --event 'A' --include context --json",
	}, page["omitted"])
	require.Nil(t, page["truncated"])
}

func TestCompactNamesViewFullWhenNothingWasStripped(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	st.Append(ingestion(store.Event{Idx: 0, Type: sp("track"), Message: json.RawMessage(`{"type":"track"}`)}))

	_, page := get(t, srv.URL+"/_dev/v1/events?since=1")

	omitted := page["omitted"].(map[string]any)
	require.Equal(t, []any{}, omitted["context"])
	require.Equal(t, "rudder-cli dev events list --since 1 --view full --json", omitted["next"])
	require.Nil(t, firstItem(t, page)["context"], "an empty residual is null")
}

func TestIncludeContextRestoresTheStrippedKeys(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?include=context")

	item := firstItem(t, page)
	ctx := item["context"].(map[string]any)
	require.Contains(t, ctx, "library")
	require.Contains(t, ctx, "traits")
	require.Contains(t, ctx, "appEnvironment")
	omitted := page["omitted"].(map[string]any)
	require.Equal(t, []any{}, omitted["context"])
	require.Equal(t, "rudder-cli dev events list --since 0 --include 'context' --view full --json", omitted["next"])
}

func TestIncludeEnrichmentAddsServerValues(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?include=enrichment")

	item := firstItem(t, page)
	require.Equal(t, "127.0.0.1", item["request_ip"])
	require.Equal(t, "r-1", item["rudderId"])
	require.NotContains(t, item, "enrichedMessage")
}

func TestViewFullHasNoOmittedBlockAndIgnoresInclude(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?view=full&include=context")

	require.Equal(t, "full", page["view"])
	require.Nil(t, page["omitted"])
	item := firstItem(t, page)
	require.Contains(t, item, "message")
	require.Contains(t, item, "enrichedMessage")
	require.NotContains(t, item, "context")
}

func TestViewSummaryIsOneShortLinePerEvent(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?view=summary")

	require.Equal(t, map[string]any{
		"seq": float64(1), "idx": float64(0), "type": "track", "event": "A", "userId": "u1", "statusCode": float64(200),
	}, firstItem(t, page))
	require.Equal(t, "rudder-cli dev events list --since 0 --json", page["omitted"].(map[string]any)["next"],
		"the summary list names the compact call")
}

func TestFieldsSelectsNestedPaths(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?fields=properties&fields=message.context.page.path&fields=message.nope")

	require.Equal(t, "fields", page["view"])
	require.Equal(t, map[string]any{
		"seq": float64(1), "idx": float64(0), "type": "track", "event": "A",
		"properties": map[string]any{"n": float64(1)},
		"message":    map[string]any{"context": map[string]any{"page": map[string]any{"path": "/chat"}}, "nope": nil},
	}, firstItem(t, page))
}

func TestFieldsConflicts(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)

	for query, param := range map[string]string{
		"fields=properties&view=full":       "fields",
		"fields=properties&include=context": "include",
		"fields=nope.x":                     "fields",
		"include=request":                   "include",
		"view=raw":                          "view",
		"order=desc":                        "order",
		"statusCode=abc":                    "statusCode",
		"maxBytes=-1":                       "maxBytes",
		"userId=a&userId=b":                 "userId",
	} {
		status, body := get(t, srv.URL+"/_dev/v1/events?"+query)
		require.Equal(t, http.StatusBadRequest, status, query)
		errObj := body["error"].(map[string]any)
		require.Equal(t, param, errObj["param"], query)
		require.NotEmpty(t, errObj["next"], query)
	}
}

func TestFiltersRouteStatusCodeAndIdentity(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))
	other := ingestion(store.Event{Idx: 0, Type: sp("identify"), UserID: sp("u2"), AnonymousID: sp("a2"),
		Message: json.RawMessage(`{"type":"identify"}`)})
	other.Route, other.StatusCode = "/v1/identify", 400
	st.Append(other)

	for query, want := range map[string]int{
		"route=/v1/identify":                 1,
		"route=/v1/identify&route=/v1/batch": 2,
		"statusCode=400":                     1,
		"statusCode=200&statusCode=400":      2,
		"userId=u2":                          1,
		"anonymousId=a2":                     1,
		"anonymousId=zz":                     0,
		"order=asc":                          2,
	} {
		_, page := get(t, srv.URL+"/_dev/v1/events?"+query)
		require.Len(t, page["events"], want, query)
	}
}

func TestMaxBytesDropsWholeTrailingRequests(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	for range 5 {
		st.Append(ingestion(browserTrack(0, "A"), browserTrack(1, "A")))
	}

	_, all := get(t, srv.URL+"/_dev/v1/events?maxBytes=0")
	require.Len(t, all["events"], 10)
	require.Nil(t, all["truncated"])

	_, page := get(t, srv.URL+"/_dev/v1/events?maxBytes=2000&event=A")
	events := page["events"].([]any)
	require.NotEmpty(t, events)
	require.Zero(t, len(events)%2, "a request is never split")
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), 2000)
	require.Equal(t, true, page["hasMore"])
	last := events[len(events)-1].(map[string]any)
	require.Equal(t, last["seq"], page["cursor"])
	truncated := page["truncated"].(map[string]any)
	require.Equal(t, "maxBytes", truncated["by"])
	require.Equal(t, float64(len(events)), truncated["kept"])
	require.Equal(t, float64(10-len(events)), truncated["matchedAfter"])
	require.Equal(t, "rudder-cli dev events list --since "+jsonNumber(page["cursor"])+" --event 'A' --max-bytes '2000' --json",
		truncated["next"], "the next page keeps the caller's cap")
}

func TestMaxBytesWithOneOversizedRequest(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(browserTrack(0, "A"), browserTrack(1, "A")))

	_, page := get(t, srv.URL+"/_dev/v1/events?maxBytes=300&since=0")

	require.Equal(t, []any{}, page["events"])
	require.Equal(t, float64(0), page["cursor"], "the cursor does not pass the request that did not fit")
	truncated := page["truncated"].(map[string]any)
	require.Greater(t, truncated["requestBytes"], float64(300))
	require.Equal(t, float64(0), truncated["kept"])
	require.Equal(t, "rudder-cli dev events list --since 0 --fields properties --json", truncated["next"])
}

func TestNextQuotesCallerValues(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	st.Append(ingestion(track(0, "A")))

	for _, name := range []string{"a'b", "$(id)", "x; rudder-cli dev reset", "two\nlines"} {
		_, page := get(t, srv.URL+"/_dev/v1/events?"+url.Values{"event": {name}}.Encode())
		next := page["omitted"].(map[string]any)["next"].(string)
		quoted := "'" + strings.ReplaceAll(name, "'", `'\''`) + "'"
		require.Contains(t, next, "--event "+quoted, name)
	}
}

func jsonNumber(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// When maxBytes keeps fewer events than min, the page still moves the
// cursor past what it kept, so a client that follows it makes progress.
func TestMinAboveKeptMovesTheCursor(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	for range 3 {
		st.Append(ingestion(browserTrack(0, "A")))
	}

	_, page := get(t, srv.URL+"/_dev/v1/events?maxBytes=1500&min=3&wait=5s")

	events := page["events"].([]any)
	require.Less(t, len(events), 3)
	require.NotEmpty(t, events)
	require.Equal(t, false, page["timedOut"])
	require.Equal(t, true, page["hasMore"])
	require.Equal(t, events[len(events)-1].(map[string]any)["seq"], page["cursor"])
}
