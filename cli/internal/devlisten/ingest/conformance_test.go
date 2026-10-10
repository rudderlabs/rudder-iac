package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Rows marked "deviation" differ from RudderStack's answer on purpose.

var comparedHeaders = []string{
	"Content-Type", "X-Content-Type-Options", "Allow", "Vary",
	"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials",
	"Access-Control-Allow-Methods", "Access-Control-Allow-Headers", "Access-Control-Max-Age",
}

var (
	okHeaders   = map[string]string{"Content-Type": "text/plain; charset=utf-8", "Vary": "Origin"}
	failHeaders = map[string]string{"Content-Type": "text/plain; charset=utf-8", "X-Content-Type-Options": "nosniff", "Vary": "Origin"}
	// onlyDev is the allowlist of the rows that check the write key.
	onlyDev = []string{"dev"}
)

type conformanceCase struct {
	name string
	// writeKeys is the allowlist; empty accepts every key.
	writeKeys []string
	request   *http.Request
	status    int
	body      string
	headers   map[string]string
}

func TestConformance(t *testing.T) {
	t.Parallel()

	groups := map[string][]conformanceCase{
		"core":       coreCases(),
		"auth":       authCases(),
		"gzip":       gzipCases(t),
		"boundary":   boundaryCases(t),
		"sdk shapes": sdkShapeCases(t),
		"bootstrap":  bootstrapCases(),
		"health":     healthCases(),
	}
	for group, cases := range groups {
		for _, tc := range cases {
			t.Run(group+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				h, _ := newTestHandler(tc.writeKeys...)
				h.startedAt = startedAt

				rec := serve(h, tc.request)

				require.Equal(t, tc.status, rec.Code)
				require.Equal(t, tc.body, rec.Body.String())
				got := map[string]string{}
				for _, name := range comparedHeaders {
					if v := rec.Header().Values(name); len(v) > 0 {
						got[name] = strings.Join(v, ", ")
					}
				}
				require.Equal(t, tc.headers, got)
			})
		}
	}
}

func req(method, target, body string, setup ...func(*http.Request)) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	for _, f := range setup {
		f(r)
	}
	return r
}

func basic(key string) func(*http.Request) {
	return func(r *http.Request) { r.SetBasicAuth(key, "") }
}

func header(name, value string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(name, value) }
}

func coreCases() []conformanceCase {
	return []conformanceCase{
		{
			name:    "track",
			request: req("POST", "/v1/track", `{"userId":"u1","event":"A"}`, basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "merge is a single-event route",
			request: req("POST", "/v1/merge", `{"userId":"u1"}`, basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "import has batch semantics",
			request: req("POST", "/v1/import", `{"batch":[{"userId":"u1"}]}`, basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "double slash",
			request: req("POST", "//v1/track", `{"userId":"u1"}`, basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			// net/http gives a request with Content-Length 0 the body http.NoBody.
			name:    "empty body",
			request: req("POST", "/v1/track", "", basic("dev"), func(r *http.Request) { r.Body = http.NoBody }),
			status:  400, body: "request body is nil\n", headers: failHeaders,
		},
		{
			name:    "empty reader with Content-Length 0",
			request: req("POST", "/v1/track", "", basic("dev")),
			status:  400, body: "invalid json\n", headers: failHeaders,
		},
		{
			name:    "chunked empty body",
			request: req("POST", "/v1/track", "", basic("dev"), func(r *http.Request) { r.ContentLength = -1 }),
			status:  400, body: "invalid json\n", headers: failHeaders,
		},
		{
			name:    "invalid json",
			request: req("POST", "/v1/track", "{", basic("dev")),
			status:  400, body: "invalid json\n", headers: failHeaders,
		},
		{
			name:    "array on a single-event route",
			request: req("POST", "/v1/track", "[]", basic("dev")),
			status:  400, body: "event is not a valid rudder event\n", headers: failHeaders,
		},
		{
			name:    "batch without batch",
			request: req("POST", "/v1/batch", "{}", basic("dev")),
			status:  400, body: "event is not a valid rudder event\n", headers: failHeaders,
		},
		{
			name:    "batch key in another case",
			request: req("POST", "/v1/batch", `{"BATCH":[{"userId":"u1"}]}`, basic("dev")),
			status:  400, body: "event is not a valid rudder event\n", headers: failHeaders,
		},
		{
			name:    "batch with a non-object",
			request: req("POST", "/v1/batch", `{"batch":[{"userId":"u1"},1]}`, basic("dev")),
			status:  400, body: "event is not a valid rudder event\n", headers: failHeaders,
		},
		{
			name:    "empty batch",
			request: req("POST", "/v1/batch", `{"batch":[]}`, basic("dev")),
			status:  400, body: "empty batch payload\n", headers: failHeaders,
		},
		{
			name:    "no identity",
			request: req("POST", "/v1/track", `{"event":"A","userId":""}`, basic("dev")),
			status:  400, body: "request neither has anonymousId nor userId\n", headers: failHeaders,
		},
		{
			name:    "extract needs no identity",
			request: req("POST", "/v1/track", `{"type":"extract"}`, basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "wrong method",
			request: req("GET", "/v1/track", "", basic("dev")),
			status:  405, body: "", headers: map[string]string{"Allow": "POST", "Vary": "Origin"},
		},
		{
			name:    "unknown path",
			request: req("POST", "/v2/track", `{"userId":"u1"}`, basic("dev")),
			status:  404, body: "unknown path\n", headers: failHeaders,
		},
	}
}

// Auth rows run with the allowlist, so the write-key checks apply.
func authCases() []conformanceCase {
	return []conformanceCase{
		{
			name: "no key", writeKeys: onlyDev,
			request: req("POST", "/v1/track", `{"userId":"u1"}`),
			status:  401, body: "failed to read writekey from header\n", headers: failHeaders,
		},
		{
			name: "key outside the allowlist", writeKeys: onlyDev,
			request: req("POST", "/v1/track", `{"userId":"u1"}`, basic("other")),
			status:  401, body: "invalid write key\n", headers: failHeaders,
		},
		{
			name: "beacon without writeKey", writeKeys: onlyDev,
			request: req("POST", "/beacon/v1/batch", `{"batch":[{"userId":"u1"}]}`),
			status:  401, body: "failed to read writekey from query params\n", headers: failHeaders,
		},
		{
			name: "auth runs before the body checks", writeKeys: onlyDev,
			request: req("POST", "/v1/track", "{"),
			status:  401, body: "failed to read writekey from header\n", headers: failHeaders,
		},
		{
			name:    "no allowlist accepts a missing key (deviation)",
			request: req("POST", "/v1/track", `{"userId":"u1"}`),
			status:  200, body: "ok", headers: okHeaders,
		},
	}
}

func gzipped(body []byte) *http.Request {
	return req("POST", "/v1/track", string(body), header("Content-Encoding", "gzip"))
}

// storedGzip is cut after tail, whose four bytes then decode as ISIZE.
func storedGzip(tail string) []byte {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.NoCompression)
	_, _ = zw.Write([]byte(`{"userId":"u1","x":"` + tail + `"}`))
	_ = zw.Close()
	b := buf.Bytes()
	return b[:bytes.LastIndex(b, []byte(tail))+len(tail)]
}

// forgedGzip decodes past the size limit but declares an ISIZE of 100.
func forgedGzip(t testing.TB) []byte {
	b := gz(t, `{"userId":"u1","x":"`+strings.Repeat("a", 5_000_000)+`"}`)
	binary.LittleEndian.PutUint32(b[len(b)-4:], 100)
	return b
}

func gzipCases(t *testing.T) []conformanceCase {
	return []conformanceCase{
		{
			name:    "valid",
			request: gzipped(gz(t, `{"userId":"u1"}`)),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name: "bad header, before auth", writeKeys: onlyDev,
			request: gzipped([]byte("not gzip")),
			status:  400, body: "failed to uncompress request body\n", headers: failHeaders,
		},
		{
			name: "cut, ISIZE above the limit, before auth", writeKeys: onlyDev,
			request: gzipped(storedGzip("abcd")),
			status:  413, body: "request body exceeds 2,048,000 bytes\n", headers: failHeaders,
		},
		{
			name:    "cut, ISIZE below the limit",
			request: gzipped(storedGzip("\x00\x00\x00\x00")),
			status:  400, body: "failed to read body from request\n", headers: failHeaders,
		},
		{
			name: "cut, ISIZE below the limit, auth first", writeKeys: onlyDev,
			request: gzipped(storedGzip("\x00\x00\x00\x00")),
			status:  401, body: "failed to read writekey from header\n", headers: failHeaders,
		},
		{
			name: "forged small ISIZE (deviation: 413 before auth)", writeKeys: onlyDev,
			request: gzipped(forgedGzip(t)),
			status:  413, body: "request body exceeds 2,048,000 bytes\n", headers: failHeaders,
		},
	}
}

// sized is a track body of exactly n bytes.
func sized(n int, identity bool) string {
	head := `{"userId":"u1","p":"`
	if !identity {
		head = `{"event":"A","p":"`
	}
	return head + strings.Repeat("a", n-len(head)-2) + `"}`
}

// batchOf is a batch of n tiny events.
func batchOf(n int) string {
	return `{"batch":[` + strings.TrimSuffix(strings.Repeat(`{"userId":"u"},`, n), ",") + `]}`
}

func boundaryCases(t *testing.T) []conformanceCase {
	const tooLarge = "request body exceeds 2,048,000 bytes\n"
	return []conformanceCase{
		{
			name:    "2,048,000 bytes",
			request: req("POST", "/v1/track", sized(maxReqSize, true), basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "2,048,000 bytes without identity",
			request: req("POST", "/v1/track", sized(maxReqSize, false), basic("dev")),
			status:  400, body: "request neither has anonymousId nor userId\n", headers: failHeaders,
		},
		{
			name: "2,048,001 bytes (deviation: before auth)", writeKeys: onlyDev,
			request: req("POST", "/v1/track", sized(maxReqSize+1, true)),
			status:  413, body: tooLarge, headers: failHeaders,
		},
		{
			name:    "gzip that decodes to 2,048,000 bytes",
			request: req("POST", "/v1/track", string(gz(t, sized(maxReqSize, true))), basic("dev"), header("Content-Encoding", "gzip")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name: "gzip that decodes to 2,048,001 bytes (deviation: before auth)", writeKeys: onlyDev,
			request: req("POST", "/v1/track", string(gz(t, sized(maxReqSize+1, true))), header("Content-Encoding", "gzip")),
			status:  413, body: tooLarge, headers: failHeaders,
		},
		{
			name:    "audiencelist above the limit (deviation)",
			request: req("POST", "/v1/audiencelist", `{"type":"audiencelist",`+sized(maxReqSize, true)[1:], basic("dev")),
			status:  413, body: tooLarge, headers: failHeaders,
		},
		{
			name:    "query over the limit",
			request: req("POST", "/v1/track?q="+strings.Repeat("a", maxReqSize), `{"userId":"u1"}`, basic("dev")),
			status:  413, body: "Request Entity Too Large\n", headers: failHeaders,
		},
		{
			name:    "batch of 10,000 events",
			request: req("POST", "/v1/batch", batchOf(maxBatchEvents), basic("dev")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "batch of 10,001 events (deviation)",
			request: req("POST", "/v1/batch", batchOf(maxBatchEvents+1), basic("dev")),
			status:  413, body: "batch has more than 10,000 events\n", headers: failHeaders,
		},
		{
			name:    "import of 10,001 events (deviation)",
			request: req("POST", "/v1/import", batchOf(maxBatchEvents+1), basic("dev")),
			status:  413, body: "batch has more than 10,000 events\n", headers: failHeaders,
		},
	}
}

func sdkShapeCases(t *testing.T) []conformanceCase {
	return []conformanceCase{
		{
			name: "node gzip without Content-Type",
			request: req("POST", "/v1/batch", string(gz(t, `{"batch":[{"userId":"u1"}],"sentAt":"2026-09-30T12:00:00Z"}`)),
				basic("dev"), header("Content-Encoding", "gzip")),
			status: 200, body: "ok", headers: okHeaders,
		},
		{
			name:    "beacon as text/plain",
			request: req("POST", "/beacon/v1/batch?writeKey=dev", `{"batch":[{"anonymousId":"a1"}]}`, header("Content-Type", "text/plain;charset=UTF-8")),
			status:  200, body: "ok", headers: okHeaders,
		},
		{
			name:    "pixel with dotted keys and a quoted anonymousId",
			request: req("GET", "/pixel/v1/page?writeKey=dev&anonymousId=%22a1%22&context.page.path=%2F&name=", ""),
			status:  200, body: pixelGIF, headers: map[string]string{"Content-Type": "image/gif", "Vary": "Origin"},
		},
		{
			name:    "pixel with too many keys (deviation)",
			request: req("GET", "/pixel/v1/track?writeKey=dev&event=e&anonymousId=a"+strings.Repeat("&k=1", maxPixelKeys-2), ""),
			status:  400, body: "pixel query has more than 1000 keys\n", headers: failHeaders,
		},
		{
			name:    "pixel key too deep (deviation)",
			request: req("GET", "/pixel/v1/track?writeKey=dev&event=e&anonymousId=a&"+strings.Repeat("a.", maxPixelKeyDepth)+"b=1", ""),
			status:  400, body: "pixel query key has more than 32 levels\n", headers: failHeaders,
		},
		{
			name:    "pixel at both limits",
			request: req("GET", "/pixel/v1/track?writeKey=dev&event=e&anonymousId=a&"+strings.Repeat("a.", maxPixelKeyDepth-1)+"b=1"+strings.Repeat("&k=1", maxPixelKeys-4), ""),
			status:  200, body: pixelGIF, headers: map[string]string{"Content-Type": "image/gif", "Vary": "Origin"},
		},
		{
			name:    "pixel wrong method",
			request: req("POST", "/pixel/v1/track?writeKey=dev", ""),
			status:  405, body: "", headers: map[string]string{"Allow": "GET", "Vary": "Origin"},
		},
		{
			name: "preflight",
			request: req("OPTIONS", "/v1/batch", "",
				header("Origin", "https://app.example"),
				header("Access-Control-Request-Method", "POST"),
				header("Access-Control-Request-Headers", "content-type,authorization")),
			status: 204, body: "",
			headers: map[string]string{
				"Vary":                             "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":      "https://app.example",
				"Access-Control-Allow-Methods":     "POST",
				"Access-Control-Allow-Headers":     "content-type,authorization",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Max-Age":           "900",
			},
		},
		{
			name:    "cross-origin post",
			request: req("POST", "/v1/track", `{"userId":"u1"}`, basic("dev"), header("Origin", "https://app.example")),
			status:  200, body: "ok",
			headers: map[string]string{
				"Content-Type":                     "text/plain; charset=utf-8",
				"Vary":                             "Origin",
				"Access-Control-Allow-Origin":      "https://app.example",
				"Access-Control-Allow-Credentials": "true",
			},
		},
	}
}

var (
	// startedAt is not UTC, so the rows show that updatedAt is.
	startedAt = time.Date(2026, 9, 30, 15, 4, 5, 123_000_000, time.FixedZone("UTC+3", 3*3600))

	// sourceConfigDev is the /sourceConfig answer for the key dev, byte for byte.
	sourceConfigDev = `{"source":{"config":{"statsCollection":{"errors":{"enabled":false},"metrics":{"enabled":false}}},` +
		`"dataplanes":{},"destinations":[],"enabled":true,"id":"dev-ef260e9aa3c6","name":"rudder-cli dev listen",` +
		`"updatedAt":"2026-09-30T12:04:05.123Z","workspaceId":"dev-workspace","writeKey":"dev"},"updatedAt":"2026-09-30T12:04:05.123Z"}`

	jsonHeaders = map[string]string{"Content-Type": "application/json; charset=utf-8", "Vary": "Origin"}
)

// Browser and mobile SDKs fetch /sourceConfig before they send, so they
// start only on these answers.
func bootstrapCases() []conformanceCase {
	return []conformanceCase{
		{
			name:    "sourceConfig",
			request: req("GET", "/sourceConfig", "", basic("dev")),
			status:  200, body: sourceConfigDev, headers: jsonHeaders,
		},
		{
			name:    "sourceConfig with a trailing slash and a query, as analytics-js sends it",
			request: req("GET", "/sourceConfig/?p=npm&v=3.31.4&build=modern&writeKey=dev&lockIntegrationsVersion=true", "", basic("dev")),
			status:  200, body: sourceConfigDev, headers: jsonHeaders,
		},
		{
			name:    "sourceConfig with a double slash",
			request: req("GET", "//sourceConfig?p=android&v=1.28.1", "", basic("dev")),
			status:  200, body: sourceConfigDev, headers: jsonHeaders,
		},
		{
			name:    "sourceConfig with a listed key",
			request: req("GET", "/sourceConfig?p=swift", "", basic("dev")), writeKeys: onlyDev,
			status: 200, body: sourceConfigDev, headers: jsonHeaders,
		},
		{
			name:    "sourceConfig without a key",
			request: req("GET", "/sourceConfig", ""), writeKeys: onlyDev,
			status: 401, body: `{"message":"Writekey not found in basic auth"}`, headers: jsonHeaders,
		},
		{
			name:    "sourceConfig with a key outside the allowlist",
			request: req("GET", "/sourceConfig", "", basic("other")), writeKeys: onlyDev,
			status: 400, body: `{"message":"Invalid write key"}`, headers: jsonHeaders,
		},
		{
			name:    "ad-block probe",
			request: req("HEAD", "/sourceConfig/?view=ad", ""), writeKeys: onlyDev,
			status: 204, body: "", headers: map[string]string{"Vary": "Origin"},
		},
		{
			name:    "HEAD sourceConfig",
			request: req("HEAD", "/sourceConfig", "", basic("dev")),
			status:  200, body: "", headers: jsonHeaders,
		},
		{
			name:    "sourceConfig wrong method",
			request: req("POST", "/sourceConfig", "", basic("dev")),
			status:  405, body: "", headers: map[string]string{"Allow": "GET, HEAD", "Vary": "Origin"},
		},
		{
			name: "sourceConfig preflight",
			request: req("OPTIONS", "/sourceConfig/?p=cdn", "",
				header("Origin", "http://localhost:3000"),
				header("Access-Control-Request-Method", "GET"),
				header("Access-Control-Request-Headers", "authorization")),
			status: 204, body: "",
			headers: map[string]string{
				"Vary":                             "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Methods":     "GET",
				"Access-Control-Allow-Headers":     "authorization",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Max-Age":           "900",
			},
		},
		{
			name:    "rsaMetrics",
			request: req("POST", "/rsaMetrics", `{"errors":[]}`, header("Origin", "http://localhost:3000")),
			status:  200, body: "{}",
			headers: map[string]string{
				"Content-Type":                     "application/json; charset=utf-8",
				"Vary":                             "Origin",
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name:    "cluster-info skips CORS",
			request: req("GET", "/cluster-info", "", header("Origin", "http://localhost:3000")),
			status:  200, body: `{"nodeCount":1}`, headers: map[string]string{"Content-Type": "application/json; charset=utf-8"},
		},
		{
			name:    "webhook with the proxy disabled",
			request: req("POST", "/v1/webhook?writeKey=dev", `{}`),
			status:  501, body: "Proxy is disabled\n",
			headers: map[string]string{"Content-Type": "text/plain; charset=utf-8", "X-Content-Type-Options": "nosniff"},
		},
		{
			name: "webhook preflight reaches the proxy",
			request: req("OPTIONS", "/v1/webhook", "",
				header("Origin", "http://localhost:3000"),
				header("Access-Control-Request-Method", "POST")),
			status: 501, body: "Proxy is disabled\n",
			headers: map[string]string{"Content-Type": "text/plain; charset=utf-8", "X-Content-Type-Options": "nosniff"},
		},
	}
}

func healthCases() []conformanceCase {
	return []conformanceCase{
		{name: "root", request: req("GET", "/", ""), status: 200, body: `{"status":"ready"}`, headers: jsonHeaders},
		{name: "health", request: req("GET", "/health", ""), status: 200, body: `{"status":"ready"}`, headers: jsonHeaders},
		{name: "readiness", request: req("GET", "/internal/readiness", ""), status: 200, body: `{"status":"ready"}`, headers: jsonHeaders},
		{name: "liveness", request: req("GET", "/internal/liveness", ""), status: 200, body: "", headers: map[string]string{"Vary": "Origin"}},
		{
			name: "robots", request: req("GET", "/robots.txt", ""),
			status: 200, body: "User-agent: * \nDisallow: / \n", headers: map[string]string{"Content-Type": "text/plain; charset=utf-8", "Vary": "Origin"},
		},
		{name: "version", request: req("GET", "/version", ""), status: 200, body: `{"Version":"` + testVersion + `"}`, headers: jsonHeaders},
		{name: "wrong method", request: req("POST", "/health", ""), status: 405, body: "", headers: map[string]string{"Allow": "GET", "Vary": "Origin"}},
	}
}
