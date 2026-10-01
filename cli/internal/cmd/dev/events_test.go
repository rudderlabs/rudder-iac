package dev

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

const summaryBody = `{"apiVersion":"v1","serverId":"9f3ac1d2b7e4c601","since":41,"cursor":47,"evictedThrough":0,` +
	`"timedOut":false,"waitedMs":0,"summary":{"requests":{"total":3,"failed":1,"byRoute":{"/v1/track":3},` +
	`"byStatusCode":{"200":2,"400":1},"byOutcome":{"accepted":2,"rejected":1},"byStage":{"identity":1}},` +
	`"events":{"total":2,"byType":{"track":2}},"byEvent":{"Order Completed":0,"chatOpened":1,"chatSuggestionClicked":1},` +
	`"rejected":{"events":1,"byEvent":{"Order Completed":1}},"byWriteKey":{"web":{"requests":3,"events":2}},` +
	`"control":{"total":1,"sourceConfig":1,"sourceConfigFailed":0,"preflight":0,"pluginPath":0,"other":0},` +
	`"bySource":{"byChannel":{},"bySdk":{}},"diagnosis":[` +
	`{"code":"body_rejected","count":1,"message":"1 request was rejected while the body was read.","next":"curl -fsS 'http://127.0.0.1:1/_dev/v1/requests?since=41&failed=true&view=compact'"},` +
	`{"code":"filtered_empty","count":3,"message":"Requests arrived, but the filters match none.","next":"rudder-cli dev events --since 41 --json"}]},` +
	`"next":"rudder-cli dev events --since 47 --event 'Order Completed' --json"}` + "\n"

var summaryHeader = http.Header{
	"Content-Type":    {"application/json"},
	"X-Dev-Server-Id": {"9f3ac1d2b7e4c601"},
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
		{args: []string{"events"}, want: "/_dev/v1/events?view=counts"},
		{
			args: []string{
				"events", "--since", "5", "--event", "Order Completed", "--event", "chat*", "--type", "track",
				"--user-id", "u1", "--anonymous-id", "a1", "--write-key", "web", "--write-key", "",
				"--wait", "30s", "--min", "2", "--server-id", "9f3ac1d2b7e4c601", "--timeout", "40s", "-j",
			},
			want: "/_dev/v1/events?anonymousId=a1&event=Order+Completed&event=chat%2A&min=2&serverId=9f3ac1d2b7e4c601" +
				"&since=5&type=track&userId=u1&view=counts&wait=30s&writeKey=web&writeKey=",
		},
		{args: []string{"events", "--since", "5m"}, want: "/_dev/v1/events?since=5m&view=counts"},
		{args: []string{"events", "--since", "2026-09-30T12:00:00Z"}, want: "/_dev/v1/events?since=2026-09-30T12%3A00%3A00Z&view=counts"},
		{args: []string{"events", "--user-id", ""}, want: "/_dev/v1/events?userId=&view=counts"},
		{args: []string{"events", "list"}, want: "/_dev/v1/events"},
		{args: []string{"events", "list", "--view", "list"}, want: "/_dev/v1/events"},
		{args: []string{"events", "list", "--view", "full", "--json"}, want: "/_dev/v1/events"},
		{args: []string{"events", "list", "--view", "compact"}, want: "/_dev/v1/events?view=compact"},
		{
			args: []string{"events", "list", "--fields", "properties", "--fields", "context.page.path", "--limit", "5"},
			want: "/_dev/v1/events?fields=properties&fields=context.page.path&limit=5",
		},
		{
			args: []string{"events", "list", "--since", "3", "--event", "e", "--type", "page", "--write-key", "k", "--server-id", "s"},
			want: "/_dev/v1/events?event=e&serverId=s&since=3&type=page&writeKey=k",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()
			rec := newRecorder(t, http.StatusOK, streamHeader, "")
			if tc.args[len(tc.args)-1] != "list" && !contains(tc.args, "list") {
				rec.header, rec.body = summaryHeader, summaryBody
			}

			_, stderr, err := execute(append([]string{"dev"}, append(tc.args, "--url", rec.srv.URL)...)...)

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
			args: []string{"events", "list", "--view", "counts"}, message: "--view counts is the summary, which dev events prints",
			next: "rudder-cli dev events --url URL --json",
		},
		{
			args: []string{"events", "list", "--view", "table"}, message: `--view must be list, compact or full, got "table"`,
			next: "rudder-cli dev events list --url URL --help",
		},
		{
			args:    []string{"events", "list", "--view", "compact", "--fields", "properties"},
			message: "--fields replaces --view; give one of them", next: "rudder-cli dev events list --url URL --fields properties",
		},
		{
			args:    []string{"events", "list", "--fields", "properties,event"},
			message: "--fields: repeat the flag for each path, as --fields properties --fields event",
			next:    "rudder-cli dev events list --url URL --fields properties --fields event",
		},
		{
			args: []string{"events", "list", "--limit", "0"}, message: "--limit must be 1 to 1000, got 0",
			next: "rudder-cli dev events list --url URL --help",
		},
		{
			args: []string{"events", "list", "--limit", "1001"}, message: "--limit must be 1 to 1000, got 1001",
			next: "rudder-cli dev events list --url URL --help",
		},
		{
			args: []string{"events", "--min", "0"}, message: "--min must be 1 or more, got 0",
			next: "rudder-cli dev events --url URL --help",
		},
		{
			args: []string{"events", "--wait", "5m"}, message: "--wait: 5m exceeds the maximum of 110s",
			next: "rudder-cli dev events --url URL --help",
		},
		{
			args:    []string{"events", "--since", "yesterday"},
			message: `--since: "yesterday" is not a cursor, a duration such as 5m or an RFC 3339 time`,
			next:    "rudder-cli dev events --url URL --help",
		},
		{
			args: []string{"events", "--url", "127.0.0.1:4321"}, noURL: true,
			message: `--url must be an http URL such as http://127.0.0.1:4321, got "127.0.0.1:4321"`,
			next:    "rudder-cli dev events --url " + placeholder,
		},
		{
			args: []string{"events", "list"}, noURL: true,
			message: "no listener URL: pass --url or set RUDDERSTACK_DEV_URL to the url of the ready line",
			next:    "rudder-cli dev events list --url " + placeholder,
		},
		{
			args: []string{"events", "list", "--wait", "30s"}, message: "--wait and --min belong to dev events: dev events list reads what is there",
			next: "rudder-cli dev events --url " + placeholder + " --wait 30s --json",
		},
		{
			args: []string{"events", "list", "--min", "1"}, message: "--wait and --min belong to dev events: dev events list reads what is there",
			next: "rudder-cli dev events --url " + placeholder + " --wait 30s --json",
		},
		{
			args: []string{"events", "--limit", "5"}, message: "unknown flag: --limit",
			next: "rudder-cli dev events --help",
		},
		{
			args: []string{"events", "nope"}, message: `unknown command "nope" for "rudder-cli dev events"`,
			next: "rudder-cli dev events --help",
		},
		{
			args: []string{"events", "list", "nope"}, message: `unknown argument "nope" for "rudder-cli dev events list"`,
			next: "rudder-cli dev events list --help",
		},
		{
			args: []string{"requests"}, noURL: true, message: `unknown command "requests" for "rudder-cli dev"`,
			next: "rudder-cli dev --help",
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			rec := newRecorder(t, http.StatusOK, streamHeader, "")
			args := append([]string{"dev"}, tc.args...)
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
		`"param":null,"details":{"serverId":"aaaa","startedAt":"2026-09-30T12:00:00Z"},"next":"rudder-cli dev events --json"}}` + "\n"
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
				`"next":"rudder-cli dev events --url ` + rec.srv.URL + ` --json"}}`,
		},
		{
			args: []string{"events", "list", "-j", "--url", rec.srv.URL, "--limit", "0"},
			want: `{"error":{"status":null,"code":"usage","message":"--limit must be 1 to 1000, got 0","param":null,` +
				`"details":null,"next":"rudder-cli dev events list --url ` + rec.srv.URL + ` --help"}}`,
		},
		{
			args: []string{"events", "--json", "--url", closed},
			want: `{"error":{"status":null,"code":"server_unreachable","message":"no listener answers at ` + closed +
				`","param":null,"details":null,"next":"rudder-cli dev listen --help"}}`,
		},
	} {
		stdout, stderr, err := execute(append([]string{"dev"}, tc.args...)...)

		var silent *cmderrors.SilentError
		require.ErrorAs(t, err, &silent)
		require.Empty(t, stdout)
		require.Equal(t, tc.want+"\n", stderr, tc.args)
	}

	_, stderr, _ := execute("dev", "events", "--url", rec.srv.URL)
	require.Equal(t, "Error: Another listener answers at this URL.\nNext: rudder-cli dev events --url "+rec.srv.URL+" --json\n", stderr)
}

// A server with an HTML fallback answers 200 on any path. Without the
// listener's header that answer is not output: a pipe such as wc -l would
// count its lines as events.
func TestReadsRefuseA200FromAnotherServer(t *testing.T) {
	t.Parallel()
	page := "<!doctype html>\n<html><body>app</body></html>\n"
	rec := newRecorder(t, http.StatusOK, http.Header{"Content-Type": {"text/html; charset=utf-8"}}, page)
	message := rec.srv.URL + " answered HTTP 200, not as dev listen answers; check the url of the ready line"

	for _, args := range [][]string{
		{"events"},
		{"events", "--json"},
		{"events", "list"},
		{"events", "list", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()
			stdout, stderr, err := execute(append(append([]string{"dev"}, args...), "--url", rec.srv.URL)...)

			var silent *cmderrors.SilentError
			require.ErrorAs(t, err, &silent)
			require.Empty(t, stdout)
			if contains(args, "--json") {
				require.Equal(t, `{"error":{"status":null,"code":"server_unreachable","message":"`+message+
					`","param":null,"details":null,"next":"rudder-cli dev listen --help"}}`+"\n", stderr)
				return
			}
			require.Equal(t, "Error: "+message+"\nNext: rudder-cli dev listen --help\n", stderr)
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

// The summary is the server's object; the CLI adds the --url it used to
// every rudder-cli next, so the command runs as printed.
func TestEventsPrintsTheSummary(t *testing.T) {
	t.Parallel()
	rec := newRecorder(t, http.StatusOK, summaryHeader, summaryBody)
	u := rec.srv.URL

	stdout, stderr, err := execute("dev", "events", "--url", u, "--json")

	require.NoError(t, err)
	require.Empty(t, stderr)
	want := strings.NewReplacer(
		`"next":"rudder-cli dev events --since 41`, `"next":"rudder-cli dev events --url `+u+` --since 41`,
		`"next":"rudder-cli dev events --since 47`, `"next":"rudder-cli dev events --url `+u+` --since 47`,
	).Replace(summaryBody)
	require.Equal(t, want, stdout)

	stdout, stderr, err = execute("dev", "events", "--url", u)

	require.NoError(t, err)
	require.Empty(t, stderr)
	require.Equal(t, `Window after cursor 41; cursor now 47.

ACCEPTED EVENTS  2
  chatOpened             1
  chatSuggestionClicked  1
  Order Completed        0
REJECTED EVENTS  1
  Order Completed  1
REQUESTS  3 (1 failed)
CONTROL   1

body_rejected (1): 1 request was rejected while the body was read.
  Next: curl -fsS 'http://127.0.0.1:1/_dev/v1/requests?since=41&failed=true&view=compact'
filtered_empty (3): Requests arrived, but the filters match none.
  Next: rudder-cli dev events --url `+u+` --since 41 --json

Next: rudder-cli dev events --url `+u+` --since 47 --event 'Order Completed' --json
`, stdout)
}

func TestEventsNotesEvictionAndTimeout(t *testing.T) {
	t.Parallel()
	body := strings.Replace(strings.Replace(summaryBody, `"evictedThrough":0`, `"evictedThrough":44`, 1),
		`"timedOut":false,"waitedMs":0`, `"timedOut":true,"waitedMs":30001`, 1)
	rec := newRecorder(t, http.StatusOK, summaryHeader, body)

	stdout, stderr, err := execute("dev", "events", "--url", rec.srv.URL, "--since", "41", "--wait", "30s")

	require.NoError(t, err, "a wait that runs out is a successful read")
	require.Equal(t, "note: requests up to seq 44 were evicted; the window starts after them\n", stderr)
	require.True(t, strings.HasPrefix(stdout, "Window after cursor 41; cursor now 47. The wait ran out after 30.0s.\n"), stdout)
}

const streamBody = `{"type":"track","event":"Order Completed","userId":"u1","originalTimestamp":"2026-09-30T12:00:00.000Z","properties":{"total":42,"note":"a` + "\\u001b[31m" + `b"}}
{"type":"page","name":"Home","anonymousId":"a1","sentAt":"2026-09-30T12:00:01.000Z"}
{"userId":"u2","event":"Signed Up"}
`

func TestEventsListPrintsTheStream(t *testing.T) {
	t.Parallel()
	header := streamHeader.Clone()
	header.Set("X-Dev-Has-More", "true")
	header.Set("X-Dev-Cursor", "61")
	rec := newRecorder(t, http.StatusOK, header, streamBody)

	stdout, stderr, err := execute("dev", "events", "list", "--url", rec.srv.URL, "--json")

	require.NoError(t, err)
	require.Equal(t, streamBody, stdout, "the server's bytes as they are")
	require.Equal(t, "more events: continue with --since 61\n", stderr)
}

func TestEventsListTables(t *testing.T) {
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

			stdout, stderr, err := execute(append([]string{"dev", "events", "list", "--url", rec.srv.URL}, tc.args...)...)

			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Equal(t, tc.want, stdout)
		})
	}
}

func TestEventsListTableEdges(t *testing.T) {
	t.Parallel()
	rec := newRecorder(t, http.StatusOK, streamHeader, "")

	stdout, _, err := execute("dev", "events", "list", "--url", rec.srv.URL, "--since", "57")

	require.NoError(t, err)
	require.Equal(t, "0 events after cursor 57; run rudder-cli dev events --url "+rec.srv.URL+" --since 57 for the diagnosis\n", stdout)

	// The diagnosis reads the same window, so a time since goes along too.
	stdout, _, err = execute("dev", "events", "list", "--url", rec.srv.URL, "--since", "2026-09-30T12:00:00Z")

	require.NoError(t, err)
	require.Equal(t, "0 events since 2026-09-30T12:00:00Z; run rudder-cli dev events --url "+rec.srv.URL+
		" --since 2026-09-30T12:00:00Z for the diagnosis\n", stdout)

	header := streamHeader.Clone()
	header.Set("X-Dev-Has-More", "true")
	header.Set("X-Dev-Cursor", "61")
	long := `{"type":"track","event":"` + strings.Repeat("x", 100) + `","userId":"u"}` + "\n"
	rec = newRecorder(t, http.StatusOK, header, long)

	stdout, stderr, err := execute("dev", "events", "list", "--url", rec.srv.URL)

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
func TestEventsListEscapesControlCharacters(t *testing.T) {
	t.Parallel()
	rec := newRecorder(t, http.StatusOK, streamHeader, `{"event":"a\u001b[2Jb\u009bc","userId":"u"}`+"\n")

	stdout, _, err := execute("dev", "events", "list", "--url", rec.srv.URL)

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

	_, _, err := execute("dev", "events", "list")
	require.NoError(t, err)
	_, _, err = execute("dev", "events", "list", "--url", flag.srv.URL)
	require.NoError(t, err)

	require.Equal(t, []string{"/_dev/v1/events"}, env.sent())
	require.Equal(t, []string{"/_dev/v1/events"}, flag.sent())
}
