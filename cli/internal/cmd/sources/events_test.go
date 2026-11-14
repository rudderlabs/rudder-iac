package sources

import (
	"go/build"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
)

// recorder is a listener stand-in that records each request target and
// answers with a fixed response.
type recorder struct {
	srv     *httptest.Server
	mu      sync.Mutex
	targets []string
	status  int
	header  http.Header
	body    string
}

func newRecorder(t *testing.T, status int, header http.Header, body string) *recorder {
	t.Helper()
	rec := &recorder{status: status, header: header, body: body}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.targets = append(rec.targets, r.URL.RequestURI())
		rec.mu.Unlock()
		for k, v := range rec.header {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.status)
		_, _ = w.Write([]byte(rec.body))
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (rec *recorder) sent() []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]string(nil), rec.targets...)
}

var streamHeader = http.Header{
	"Content-Type":    {"application/x-ndjson"},
	"X-Dev-Cursor":    {"47"},
	"X-Dev-Has-More":  {"false"},
	"X-Dev-Server-Id": {"9f3ac1d2b7e4c601"},
}

// Each read command sends one request, and each flag is the
// query parameter of the same name.
func TestReadCommandsSendOneRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"events"}, want: "/_dev/v1/events"},
		{args: []string{"events", "--view", "list"}, want: "/_dev/v1/events"},
		{args: []string{"events", "--view", "full", "--json"}, want: "/_dev/v1/events"},
		{args: []string{"events", "--view", "compact"}, want: "/_dev/v1/events?view=compact"},
		{
			args: []string{"events", "--fields", "properties", "--fields", "context.page.path", "--limit", "5"},
			want: "/_dev/v1/events?fields=properties&fields=context.page.path&limit=5",
		},
		{
			args: []string{"events", "--since", "3", "--event", "e", "--type", "page", "--write-key", "k", "--server-id", "s"},
			want: "/_dev/v1/events?event=e&serverId=s&since=3&type=page&writeKey=k",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()
			rec := newRecorder(t, http.StatusOK, streamHeader, "")

			_, stderr, err := execute(append([]string{"sources"}, append(tc.args, "--url", rec.srv.URL)...)...)

			require.NoError(t, err, stderr)
			require.Equal(t, []string{tc.want}, rec.sent())
		})
	}
}

func contains(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}

// Not parallel: the cases without --url need RUDDERSTACK_DEV_URL unset,
// and the help tells users to export it.
func TestReadUsageErrors(t *testing.T) {
	t.Setenv(urlEnv, "")
	const placeholder = "http://127.0.0.1:4321"
	for _, tc := range []struct {
		args    []string
		noURL   bool
		message string
		next    string
	}{
		{
			args: []string{"events", "--view", "table"}, message: `--view must be list, compact or full, got "table"`,
			next: "rudder-cli sources events --url URL --help",
		},
		{
			args:    []string{"events", "--view", "compact", "--fields", "properties"},
			message: "--fields replaces --view; give one of them", next: "rudder-cli sources events --url URL --fields properties",
		},
		{
			args:    []string{"events", "--fields", "properties,event"},
			message: "--fields: repeat the flag for each path, as --fields properties --fields event",
			next:    "rudder-cli sources events --url URL --fields properties --fields event",
		},
		{
			args: []string{"events", "--limit", "0"}, message: "--limit must be 1 to 1000, got 0",
			next: "rudder-cli sources events --url URL --help",
		},
		{
			args: []string{"events", "--limit", "1001"}, message: "--limit must be 1 to 1000, got 1001",
			next: "rudder-cli sources events --url URL --help",
		},
		{
			args:    []string{"events", "--since", "yesterday"},
			message: `--since: "yesterday" is not a cursor, a duration such as 5m or an RFC 3339 time`,
			next:    "rudder-cli sources events --url URL --help",
		},
		{
			args: []string{"events", "--url", "127.0.0.1:4321"}, noURL: true,
			message: `--url must be an http URL such as http://127.0.0.1:4321, got "127.0.0.1:4321"`,
			next:    "rudder-cli sources events --url " + placeholder,
		},
		{
			args: []string{"events"}, noURL: true,
			message: "no listener URL: pass --url or set RUDDERSTACK_DEV_URL to the url of the ready line",
			next:    "rudder-cli sources events --url " + placeholder,
		},
		{
			args: []string{"events", "nope"}, message: `unknown argument "nope" for "rudder-cli sources events"`,
			next: "rudder-cli sources events --help",
		},
		{
			args: []string{"requests"}, noURL: true, message: `unknown command "requests" for "rudder-cli sources"`,
			next: "rudder-cli sources --help",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			rec := newRecorder(t, http.StatusOK, streamHeader, "")
			args := append([]string{"sources"}, tc.args...)
			if !tc.noURL {
				args = append(args, "--url", rec.srv.URL)
			}

			stdout, stderr, err := execute(args...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			require.Empty(t, rec.sent(), "a usage error sends no request")
			next := strings.ReplaceAll(tc.next, "URL", rec.srv.URL)
			require.Equal(t, "Error: "+tc.message+"\nNext: "+next+"\n", stderr)
		})
	}
}

// With --json an error is one JSON object on stderr, and stdout stays empty.
func TestReadErrorsInJSON(t *testing.T) {
	t.Parallel()
	conflict := `{"error":{"status":409,"code":"server_changed","message":"Another listener answers at this URL.",` +
		`"param":null,"details":{"serverId":"aaaa","startedAt":"2026-09-30T12:00:00Z"},"next":"rudder-cli sources events --json"}}` + "\n"
	rec := newRecorder(t, http.StatusConflict, http.Header{"Content-Type": {"application/json"}}, conflict)
	closed := closedURL(t)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{
			args: []string{"events", "--json", "--url", rec.srv.URL, "--server-id", "bbbb"},
			want: `{"error":{"status":409,"code":"server_changed","message":"Another listener answers at this URL.",` +
				`"param":null,"details":{"serverId":"aaaa","startedAt":"2026-09-30T12:00:00Z"},` +
				`"next":"rudder-cli sources events --url ` + rec.srv.URL + ` --json"}}`,
		},
		{
			args: []string{"events", "-j", "--url", rec.srv.URL, "--limit", "0"},
			want: `{"error":{"status":null,"code":"usage","message":"--limit must be 1 to 1000, got 0","param":null,` +
				`"details":null,"next":"rudder-cli sources events --url ` + rec.srv.URL + ` --help"}}`,
		},
		{
			args: []string{"events", "--json", "--url", closed},
			want: `{"error":{"status":null,"code":"server_unreachable","message":"no listener answers at ` + closed +
				`","param":null,"details":null,"next":"rudder-cli sources listen --help"}}`,
		},
	} {
		stdout, stderr, err := execute(append([]string{"sources"}, tc.args...)...)

		var silent *cmderrors.SilentError
		require.ErrorAs(t, err, &silent)
		require.Empty(t, stdout)
		require.Equal(t, tc.want+"\n", stderr, tc.args)
	}

	_, stderr, _ := execute("sources", "events", "--url", rec.srv.URL)
	require.Equal(t, "Error: Another listener answers at this URL.\nNext: rudder-cli sources events --url "+rec.srv.URL+" --json\n", stderr)
}

// A server with an HTML fallback answers 200 on any path. Without the
// listener's header that answer is not output: a pipe such as wc -l would
// count its lines as events.
func TestReadsRefuseA200FromAnotherServer(t *testing.T) {
	t.Parallel()
	page := "<!doctype html>\n<html><body>app</body></html>\n"
	rec := newRecorder(t, http.StatusOK, http.Header{"Content-Type": {"text/html; charset=utf-8"}}, page)
	message := rec.srv.URL + " answered HTTP 200, not as sources listen answers; check the url of the ready line"

	for _, args := range [][]string{
		{"events"},
		{"events", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			stdout, stderr, err := execute(append(append([]string{"sources"}, args...), "--url", rec.srv.URL)...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			if contains(args, "--json") {
				require.Equal(t, `{"error":{"status":null,"code":"server_unreachable","message":"`+message+
					`","param":null,"details":null,"next":"rudder-cli sources listen --help"}}`+"\n", stderr)
				return
			}
			require.Equal(t, "Error: "+message+"\nNext: rudder-cli sources listen --help\n", stderr)
		})
	}
}

func closedURL(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	url := "http://" + ln.Addr().String()
	require.NoError(t, ln.Close())
	return url
}

const streamBody = `{"type":"track","event":"Order Completed","userId":"u1","originalTimestamp":"2026-09-30T12:00:00.000Z","properties":{"total":42,"note":"a` + "\\u001b[31m" + `b"}}
{"type":"page","name":"Home","anonymousId":"a1","sentAt":"2026-09-30T12:00:01.000Z"}
{"userId":"u2","event":"Signed Up"}
`

func TestEventsPrintsTheStream(t *testing.T) {
	t.Parallel()
	header := streamHeader.Clone()
	header.Set("X-Dev-Has-More", "true")
	header.Set("X-Dev-Cursor", "61")
	rec := newRecorder(t, http.StatusOK, header, streamBody)

	stdout, stderr, err := execute("sources", "events", "--url", rec.srv.URL, "--json")

	require.NoError(t, err)
	require.Equal(t, streamBody, stdout, "the server's bytes as they are")
	require.Equal(t, "more events: continue with --since 61\n", stderr)
}

func TestEventsTables(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "list",
			args: []string{"--since", "57"},
			want: `3 events after cursor 57
TIME                      TYPE   EVENT            USER
2026-09-30T12:00:00.000Z  track  Order Completed  u1
2026-09-30T12:00:01.000Z  page   Home             a1
                                 Signed Up        u2
`,
		},
		{
			name: "compact",
			args: []string{"--view", "compact"},
			want: `3 events after cursor 0
TIME                      TYPE   EVENT            USER  PROPERTIES
2026-09-30T12:00:00.000Z  track  Order Completed  u1    {"total":42,"note":"a\u001b[31mb"}
2026-09-30T12:00:01.000Z  page   Home             a1
                                 Signed Up        u2
`,
		},
		{
			name: "fields",
			args: []string{"--fields", "properties.total", "--fields", "event", "--since", "5m"},
			want: `3 events since 5m
PROPERTIES.TOTAL  EVENT
42                Order Completed

                  Signed Up
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := newRecorder(t, http.StatusOK, streamHeader, streamBody)

			stdout, stderr, err := execute(append([]string{"sources", "events", "--url", rec.srv.URL}, tc.args...)...)

			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, tc.want, stdout)
		})
	}
}

func TestEventsTableEdges(t *testing.T) {
	t.Parallel()
	rec := newRecorder(t, http.StatusOK, streamHeader, "")

	stdout, _, err := execute("sources", "events", "--url", rec.srv.URL, "--since", "57")

	require.NoError(t, err)
	require.Equal(t, "0 events after cursor 57; refused requests: curl -fsS '"+rec.srv.URL+
		"/_dev/v1/requests?since=57&failed=true&view=compact'\n", stdout)

	// The requests read the same window, so a time since goes along too.
	stdout, _, err = execute("sources", "events", "--url", rec.srv.URL, "--since", "2026-09-30T12:00:00Z")

	require.NoError(t, err)
	require.Equal(t, "0 events since 2026-09-30T12:00:00Z; refused requests: curl -fsS '"+rec.srv.URL+
		"/_dev/v1/requests?since=2026-09-30T12%3A00%3A00Z&failed=true&view=compact'\n", stdout)

	header := streamHeader.Clone()
	header.Set("X-Dev-Has-More", "true")
	header.Set("X-Dev-Cursor", "61")
	long := `{"type":"track","event":"` + strings.Repeat("x", 100) + `","userId":"u"}` + "\n"
	rec = newRecorder(t, http.StatusOK, header, long)

	stdout, stderr, err := execute("sources", "events", "--url", rec.srv.URL)

	require.NoError(t, err)
	require.Empty(t, stderr, "the table names the next cursor itself")
	require.Equal(t, "1 events after cursor 0; more with --since 61\n"+
		"TIME  TYPE   EVENT                                                         USER\n"+
		"      track  "+strings.Repeat("x", 59)+"…  u\n", stdout)
}

// The CLI is an HTTP client of the listener, with no second path
// into its store or its API code.
func TestReadCommandsImportNoServerInternals(t *testing.T) {
	t.Parallel()
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)

	for _, imp := range pkg.Imports {
		require.NotContains(t, imp, "devlisten/store")
		require.NotContains(t, imp, "devlisten/api")
		require.NotContains(t, imp, "devlisten/ingest")
	}
	require.Contains(t, pkg.Imports, "github.com/rudderlabs/rudder-iac/cli/internal/devlisten")
}

// A captured name comes from any page that can post to the listener, so a
// terminal escape in it prints as text.
func TestEventsEscapesControlCharacters(t *testing.T) {
	t.Parallel()
	rec := newRecorder(t, http.StatusOK, streamHeader, `{"event":"a\u001b[2Jb\u009bc","userId":"u"}`+"\n")

	stdout, _, err := execute("sources", "events", "--url", rec.srv.URL)

	require.NoError(t, err)
	require.Contains(t, stdout, `a\u001b[2Jb\u009bc`)
	require.NotContains(t, stdout, "\x1b")
}

// The flag wins over the environment, and the environment serves a shell
// that sets the URL once.
func TestReadCommandsTakeTheURLFromTheEnvironment(t *testing.T) {
	env := newRecorder(t, http.StatusOK, streamHeader, "")
	flag := newRecorder(t, http.StatusOK, streamHeader, "")
	t.Setenv(urlEnv, env.srv.URL)

	_, _, err := execute("sources", "events")
	require.NoError(t, err)
	_, _, err = execute("sources", "events", "--url", flag.srv.URL)
	require.NoError(t, err)

	require.Equal(t, []string{"/_dev/v1/events"}, env.sent())
	require.Equal(t, []string{"/_dev/v1/events"}, flag.sent())
}

func TestEventsTableFieldsThroughAList(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "multiple values", body: `{"properties":{"products":[{"sku":"a"},{"sku":"b"}]}}`, want: `["a","b"]`},
		{name: "missing keys and non-objects keep their place", body: `{"properties":{"products":[{"sku":"a"},{},"x",null,3]}}`, want: `["a",null,"x",null,3]`},
		{name: "no selected values", body: `{"properties":{"products":[{},"x",null,3]}}`, want: `[null,"x",null,3]`},
		{name: "a list inside a list stays as sent", body: `{"properties":{"products":[[{"sku":"a","x":1}]]}}`, want: `[[{"sku":"a","x":1}]]`},
		{name: "empty list", body: `{"properties":{"products":[]}}`, want: `[]`},
		{name: "missing path", body: `{}`, want: ""},
		{name: "scalar path", body: `{"properties":{"products":{"sku":"a"}}}`, want: "a"},
		{name: "JSON values", body: `{"properties":{"products":[{"sku":null},{"sku":false},{"sku":42},{"sku":"a\"b"}]}}`, want: `[null,false,42,"a\"b"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := newRecorder(t, http.StatusOK, streamHeader, tc.body+"\n")

			stdout, stderr, err := execute("sources", "events", "--url", rec.srv.URL, "--fields", "properties.products.sku")

			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, "1 events after cursor 0\nPROPERTIES.PRODUCTS.SKU\n"+tc.want+"\n", stdout)
		})
	}
}

func TestEventsTableLastDuplicateKeyWins(t *testing.T) {
	t.Parallel()
	const body = `{"originalTimestamp":"old","originalTimestamp":"new","type":"page","type":"track","event":"Old","event":"New","userId":"u1","userId":"u2","properties":{"old":1},"properties":{"sku":"old","sku":"new"}}` + "\n"
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "standard columns",
			args: []string{"--event", "New"},
			want: "TIME  TYPE   EVENT  USER\nnew   track  New    u2\n",
		},
		{
			name: "compact properties",
			args: []string{"--view", "compact"},
			want: "TIME  TYPE   EVENT  USER  PROPERTIES\nnew   track  New    u2    {\"sku\":\"old\",\"sku\":\"new\"}\n",
		},
		{
			name: "selected field",
			args: []string{"--fields", "properties.sku"},
			want: "PROPERTIES.SKU\nnew\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := newRecorder(t, http.StatusOK, streamHeader, body)

			stdout, stderr, err := execute(append([]string{"sources", "events", "--url", rec.srv.URL}, tc.args...)...)

			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, "1 events after cursor 0\n"+tc.want, stdout)
		})
	}
}
