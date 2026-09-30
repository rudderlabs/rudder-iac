package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

func gz(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestBeaconWithoutWriteKeyParamIsRejected(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	// The oracle key check applies under an allowlist only.
	g.AllowWriteKeys([]string{"dev"})

	got := send(t, g, post("/beacon/v1/batch", `{"batch":[{"userId":"u1"}]}`))

	require.Equal(t, http.StatusUnauthorized, got.status)
	require.Equal(t, "failed to read writekey from query params\n", got.body)
	require.Equal(t, "beacon", onlyRecord(t, st).Transport)
}

func TestBeaconTakesWriteKeyFromQuery(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := httptest.NewRequest(http.MethodPost, "/beacon/v1/batch?writeKey=wk", strings.NewReader(`{"batch":[{"userId":"u1"}]}`))
	r.Header.Set("Content-Type", "text/plain")

	got := send(t, g, r)

	require.Equal(t, http.StatusOK, got.status)
	rec := onlyRecord(t, st)
	require.Equal(t, "...", rec.WriteKey, "a short foreign key keeps no characters")
	require.NotEmpty(t, rec.WriteKeySha256)
	require.Len(t, rec.Events, 1)
}

func TestGzipBodyIsDecodedBeforeProcessing(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	compressed := gz(t, `{"batch":[{"type":"track","event":"A","userId":"u1"}]}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/batch", bytes.NewReader(compressed))
	r.SetBasicAuth("dev", "")
	r.Header.Set("Content-Encoding", "gzip")

	got := send(t, g, r)

	require.Equal(t, http.StatusOK, got.status)
	rec := onlyRecord(t, st)
	require.Equal(t, "gzip", rec.Request.BodyEncoding)
	require.Equal(t, `{"batch":[{"type":"track","event":"A","userId":"u1"}]}`, rec.Request.Body)
	require.Equal(t, compressed, rec.Request.BodyBase64)
	require.Equal(t, len(compressed), rec.Request.BodyBytes)
	require.Equal(t, str("A"), rec.Events[0].Event)
}

func TestCorruptGzipFailsBeforeAuth(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := httptest.NewRequest(http.MethodPost, "/v1/track", strings.NewReader("not gzip"))
	r.Header.Set("Content-Encoding", "gzip")

	got := send(t, g, r)

	require.Equal(t, http.StatusBadRequest, got.status)
	require.Equal(t, "failed to uncompress request body\n", got.body)
	require.Equal(t, "decode", onlyRecord(t, st).Rejection.Stage)
}

func TestBrokenGzipStreamFailsAtBodyReadAfterAuth(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	// The oracle key check applies under an allowlist only.
	g.AllowWriteKeys([]string{"dev"})
	compressed := gz(t, `{"userId":"u1","event":"A"}`)
	compressed[12] ^= 0xff

	r := httptest.NewRequest(http.MethodPost, "/v1/track", bytes.NewReader(compressed))
	r.Header.Set("Content-Encoding", "gzip")
	require.Equal(t, "failed to read writekey from header\n", send(t, g, r).body)

	r = httptest.NewRequest(http.MethodPost, "/v1/track", bytes.NewReader(compressed))
	r.Header.Set("Content-Encoding", "gzip")
	r.SetBasicAuth("dev", "")
	require.Equal(t, "failed to read body from request\n", send(t, g, r).body)
	last := st.Since(1).Records[0]
	require.Empty(t, last.Request.Body, "the partial output is not stored")
	require.Equal(t, compressed, last.Request.BodyBase64)
}

// A truncated stream ends in bytes that are not ISIZE; the oracle reads them
// as ISIZE anyway and answers 413 when they exceed the limit
// (SVC internal/middleware/uncompress.go, package doc).
func TestTruncatedGzipIsReadAsOversized(t *testing.T) {
	t.Parallel()
	g, _ := newTestGateway()
	compressed := gz(t, `{"userId":"u1","event":"A"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/track", bytes.NewReader(compressed[:len(compressed)-6]))
	r.Header.Set("Content-Encoding", "gzip")

	got := send(t, g, r)

	require.Equal(t, http.StatusRequestEntityTooLarge, got.status)
	require.Equal(t, "request size exceeds max limit\n", got.body)
}

func TestBodyRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, route, body, want string
		status                  int
		stage                   string
	}{
		{"empty body", "/v1/track", "", "request body is nil\n", 400, "body"},
		{"invalid json", "/v1/track", "{", "invalid json\n", 400, "parse"},
		{"array on single route", "/v1/track", "[]", "event is not a valid rudder event\n", 400, "parse"},
		{"batch without batch key", "/v1/batch", "{}", "event is not a valid rudder event\n", 400, "parse"},
		{"batch with non object", "/v1/batch", `{"batch":[1]}`, "event is not a valid rudder event\n", 400, "parse"},
		{"empty batch", "/v1/batch", `{"batch":[]}`, "empty batch payload\n", 400, "batch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g, st := newTestGateway()

			got := send(t, g, post(tc.route, tc.body))

			require.Equal(t, tc.status, got.status)
			require.Equal(t, tc.want, got.body)
			rec := onlyRecord(t, st)
			require.Equal(t, tc.stage, rec.Rejection.Stage)
			require.Empty(t, rec.Events)
		})
	}
}

func TestNonIdentifiableEventRejectsTheWholeBatch(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	got := send(t, g, post("/v1/batch", `{"batch":[{"userId":"u1"},{"anonymousId":" ​ "}]}`))

	require.Equal(t, http.StatusBadRequest, got.status)
	require.Equal(t, "request neither has anonymousId nor userId\n", got.body)
	rec := onlyRecord(t, st)
	idx := 1
	require.Equal(t, &store.Rejection{Stage: "identity", Reason: "request neither has anonymousId nor userId", Idx: &idx}, rec.Rejection)
	require.Len(t, rec.Events, 2)
	require.NotNil(t, rec.Events[0].EnrichedMessage)
	require.Nil(t, rec.Events[1].EnrichedMessage)
}

func TestEnrichmentKeepsGivenMessageIDAndRequestIP(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	send(t, g, post("/v1/identify", `{"userId":"u1","anonymousId":"a1","messageId":"m-1","request_ip":"10.0.0.9","receivedAt":"old"}`))

	ev := onlyRecord(t, st).Events[0]
	require.Equal(t, str("m-1"), ev.MessageID)
	require.Equal(t, str("identify"), ev.Type)
	require.Contains(t, string(ev.EnrichedMessage), `"request_ip":"10.0.0.9"`)
	require.Contains(t, string(ev.EnrichedMessage), `"receivedAt":"2026-09-29T12:00:00.123456Z"`)
}

func TestAuthorizationIsRedactedInTheRecord(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	send(t, g, post("/v1/track", `{"userId":"u1"}`))

	require.Equal(t, []string{"REDACTED"}, onlyRecord(t, st).Request.Headers["Authorization"])
}

func TestCredentialHeadersAreRedactedAndListed(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	req := post("/v1/track", `{"userId":"u1"}`)
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("Proxy-Authorization", "Basic x")
	req.Header.Set("X-Api-Key", "k")
	req.Header.Set("X-Session-Token", "t")
	req.Header.Set("X-Client-Secret", "s")
	req.Header.Set("User-Agent", "analytics-node/3.0.9")

	send(t, g, req)

	rec := onlyRecord(t, st).Request
	require.Equal(t, []string{"Authorization", "Cookie", "Proxy-Authorization", "X-Api-Key", "X-Client-Secret",
		"X-Session-Token"}, rec.RedactedHeaders)
	for _, name := range rec.RedactedHeaders {
		require.Equal(t, []string{"REDACTED"}, rec.Headers[name], name)
	}
	require.Equal(t, []string{"analytics-node/3.0.9"}, rec.Headers["User-Agent"])
}

func TestPreflightCopiesRSCors(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := httptest.NewRequest(http.MethodOptions, "/v1/batch", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Access-Control-Request-Method", "POST")
	r.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	got := send(t, g, r)

	require.Equal(t, http.StatusNoContent, got.status)
	require.Empty(t, got.body)
	require.Equal(t, http.Header{
		"Vary":                             {"Origin, Access-Control-Request-Method, Access-Control-Request-Headers"},
		"Access-Control-Allow-Origin":      {"http://localhost:3000"},
		"Access-Control-Allow-Methods":     {"POST"},
		"Access-Control-Allow-Headers":     {"authorization,content-type"},
		"Access-Control-Allow-Credentials": {"true"},
		"Access-Control-Max-Age":           {"900"},
	}, got.header)
	rec := onlyRecord(t, st)
	require.Equal(t, "control", rec.Kind)
	require.Equal(t, got.header, rec.Response.Headers)
}

func TestActualRequestGetsCORSHeaders(t *testing.T) {
	t.Parallel()
	g, _ := newTestGateway()
	r := post("/v1/track", `{"userId":"u1"}`)
	r.Header.Set("Origin", "http://localhost:3000")

	got := send(t, g, r)

	require.Equal(t, "Origin", got.header.Get("Vary"))
	require.Equal(t, "http://localhost:3000", got.header.Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", got.header.Get("Access-Control-Allow-Credentials"))
}

func TestSourceConfig(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	r := httptest.NewRequest(http.MethodGet, "/sourceConfig/?p=npm&v=3.0.0", nil)
	r.SetBasicAuth("dev", "")

	got := send(t, g, r)

	require.Equal(t, http.StatusOK, got.status)
	require.Equal(t, "application/json; charset=utf-8", got.header.Get("Content-Type"))
	require.JSONEq(t, `{"source":{"id":"dev-ef260e9aa3c6","name":"rudder-cli dev listen","writeKey":"dev","enabled":true,
		"config":{},"workspaceId":"dev-workspace","dataplanes":{},"destinations":[],
		"updatedAt":"2026-09-29T12:00:00.123Z"},"updatedAt":"2026-09-29T12:00:00.123Z"}`, got.body)
	rec := onlyRecord(t, st)
	require.Equal(t, "control", rec.Kind)
	require.Equal(t, "dev", rec.WriteKey)
	require.Equal(t, "/sourceConfig", rec.Route, "route drops the trailing slash; target keeps it")
	require.Equal(t, "/sourceConfig/?p=npm&v=3.0.0", rec.Request.Target)
}

func TestSourceConfigWithoutWriteKey(t *testing.T) {
	t.Parallel()
	g, _ := newTestGateway()
	// The oracle key check applies under an allowlist only.
	g.AllowWriteKeys([]string{"dev"})

	got := send(t, g, httptest.NewRequest(http.MethodGet, "/sourceConfig", nil))

	require.Equal(t, http.StatusUnauthorized, got.status)
	require.JSONEq(t, `{"message":"Writekey not found in basic auth"}`, got.body)
}

func TestSourceConfigHead(t *testing.T) {
	t.Parallel()
	g, _ := newTestGateway()

	got := send(t, g, httptest.NewRequest(http.MethodHead, "/sourceConfig?view=ad", nil))

	require.Equal(t, http.StatusNoContent, got.status)
	require.Empty(t, got.body)
}

func TestUnknownPathIsCapturedAsControl(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	got := send(t, g, httptest.NewRequest(http.MethodGet, "/rsa-plugins.js", nil))

	require.Equal(t, http.StatusNotFound, got.status)
	require.Equal(t, "unknown path\n", got.body)
	rec := onlyRecord(t, st)
	require.Equal(t, "control", rec.Kind)
	require.True(t, rec.Failed)
}

func TestWrongMethodOnIngestRoute(t *testing.T) {
	t.Parallel()
	g, _ := newTestGateway()

	got := send(t, g, httptest.NewRequest(http.MethodGet, "/v1/track", nil))

	require.Equal(t, http.StatusMethodNotAllowed, got.status)
	require.Equal(t, "POST", got.header.Get("Allow"))
	require.Empty(t, got.body)
}

func TestDoubleSlashIsRewritten(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	got := send(t, g, post("//v1/track", `{"userId":"u1"}`))

	require.Equal(t, http.StatusOK, got.status)
	require.Equal(t, "/v1/track", onlyRecord(t, st).Route)
}

func TestHealthRoutesAreNotCaptured(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	for _, path := range []string{"/", "/health", "/internal/readiness", "/internal/liveness", "/robots.txt", "/version"} {
		got := send(t, g, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, got.status, path)
	}
	require.Empty(t, st.Since(0).Records)

	g.Stop()
	got := send(t, g, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusServiceUnavailable, got.status)
	require.JSONEq(t, `{"status":"stopping"}`, got.body)
}

// A sender controls ISIZE. A forged small ISIZE must not let the decoded
// stream grow past the body limit, and the partial output is not stored.
func TestGzipBombWithForgedISizeIsCappedAndNotStored(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	compressed := gz(t, `{"userId":"u1","event":"`+strings.Repeat("A", maxReqSize+1)+`"}`)
	binary.LittleEndian.PutUint32(compressed[len(compressed)-4:], 10)
	r := httptest.NewRequest(http.MethodPost, "/v1/track", bytes.NewReader(compressed))
	r.Header.Set("Content-Encoding", "gzip")
	r.SetBasicAuth("dev", "")

	got := send(t, g, r)

	require.Equal(t, http.StatusRequestEntityTooLarge, got.status)
	require.Equal(t, "request size exceeds max limit\n", got.body)
	rec := onlyRecord(t, st)
	require.Empty(t, rec.Request.Body)
	require.Equal(t, compressed, rec.Request.BodyBase64)
	require.Equal(t, "size", rec.Rejection.Stage)
}

// The raw body is read through a limit, so an endless body cannot pin
// memory. The oversized body is not stored.
func TestOversizedRawBodyIsCutAtTheLimitAndNotStored(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	body := io.MultiReader(strings.NewReader(`{"userId":"u1","event":"`), neverEnding('A'))
	r := httptest.NewRequest(http.MethodPost, "/v1/track", body)
	r.SetBasicAuth("dev", "")

	got := send(t, g, r)

	require.Equal(t, http.StatusRequestEntityTooLarge, got.status)
	rec := onlyRecord(t, st)
	require.Empty(t, rec.Request.Body)
	require.False(t, rec.Request.BodyComplete)
	require.Equal(t, maxReqSize+1, rec.Request.BodyBytes)
}

type neverEnding byte

func (b neverEnding) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

func TestRedactionCoversUnderscoreKeysPasswordsAndSessions(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	req := post("/v1/track", `{"userId":"u1"}`)
	req.Header["X-Api_key"] = []string{"k"}
	req.Header.Set("X-Session-Id", "s")
	req.Header.Set("X-Db-Password", "p")
	req.Header.Set("X-Request-Id", "keep")

	send(t, g, req)

	rec := onlyRecord(t, st).Request
	require.Equal(t, []string{"Authorization", "X-Api_key", "X-Db-Password", "X-Session-Id"}, rec.RedactedHeaders)
	require.Equal(t, []string{"keep"}, rec.Headers["X-Request-Id"])
}

// A write key other than dev may be a real one: the record keeps a
// fingerprint, never the key.
func TestForeignWriteKeyIsStoredAsAFingerprint(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	const key = "notrealkey-for-tests-3456"
	req := httptest.NewRequest(http.MethodGet, "/sourceConfig?p=web", nil)
	req.SetBasicAuth(key, "")
	send(t, g, req)
	beacon := httptest.NewRequest(http.MethodPost, "/beacon/v1/batch?writeKey="+key,
		strings.NewReader(`{"batch":[{"userId":"u1"}]}`))
	send(t, g, beacon)

	for _, rec := range st.Since(0).Records {
		require.Equal(t, "notr...3456", rec.WriteKey)
		require.Equal(t, "notr", rec.WriteKeyPrefix)
		require.Equal(t, "3456", rec.WriteKeySuffix)
		require.Equal(t, "1a102365aa902010c7900e09c2bfc9548368bb58e8e3166954b5d979fcb7257f", rec.WriteKeySha256)
		raw, err := json.Marshal(rec)
		require.NoError(t, err)
		require.NotContains(t, string(raw), key)
	}
}

func TestDevWriteKeyStaysClear(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()

	send(t, g, post("/v1/track", `{"userId":"u1"}`))

	rec := onlyRecord(t, st)
	require.Equal(t, "dev", rec.WriteKey)
	require.Empty(t, rec.WriteKeySha256)
}
