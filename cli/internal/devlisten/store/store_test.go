package store

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

func capture(writeKey, target, body string) *ingest.Capture {
	r := httptest.NewRequest(http.MethodPost, target, nil)
	r.Header.Set("Content-Type", "application/json")
	return &ingest.Capture{
		ReceivedAt:   time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
		Kind:         "ingestion",
		Route:        r.URL.Path,
		Transport:    "http",
		Request:      r,
		WriteKey:     writeKey,
		Body:         []byte(body),
		Decoded:      []byte(body),
		BodyComplete: true,
		Events:       []ingest.Event{{Message: []byte(body)}},
		StatusCode:   http.StatusOK,
		Header:       http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
		ResponseBody: []byte("ok"),
	}
}

func seqs(records []*Record) []uint64 {
	var out []uint64
	for _, r := range records {
		out = append(out, r.Seq)
	}
	return out
}

func TestStoreEvictsWholeOldestRequests(t *testing.T) {
	t.Parallel()
	size := newRecord(capture("dev", "/v1/track", `{"userId":"u1"}`)).size

	for _, tc := range []struct {
		name               string
		maxRequests        int
		maxBytes           int
		wantSeqs           []uint64
		wantEvictedThrough uint64
	}{
		{name: "within both budgets", maxRequests: 5, maxBytes: 5 * size, wantSeqs: []uint64{1, 2, 3, 4, 5}},
		{name: "request budget", maxRequests: 3, maxBytes: 5 * size, wantSeqs: []uint64{3, 4, 5}, wantEvictedThrough: 2},
		{name: "byte budget", maxRequests: 5, maxBytes: 2*size + size/2, wantSeqs: []uint64{4, 5}, wantEvictedThrough: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Store{maxRequests: tc.maxRequests, maxBytes: tc.maxBytes}

			for range 5 {
				s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
			}

			records, evictedThrough := s.Since(0)
			require.Equal(t, tc.wantSeqs, seqs(records))
			require.Equal(t, tc.wantEvictedThrough, evictedThrough)
			require.Equal(t, len(tc.wantSeqs)*size, s.bytes)
		})
	}
}

func TestStoreSinceReturnsTheRecordsAfterASeq(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 4, maxBytes: maxBytes}
	for range 6 {
		s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
	}

	for since, want := range map[uint64][]uint64{
		0:              {3, 4, 5, 6},
		2:              {3, 4, 5, 6},
		4:              {5, 6},
		6:              nil,
		9:              nil,
		math.MaxUint64: nil,
	} {
		records, evictedThrough := s.Since(since)
		require.Equal(t, want, seqs(records), "since %d", since)
		require.Equal(t, uint64(2), evictedThrough)
	}
}

func TestStoreGetFindsAStoredRecordOnly(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 4, maxBytes: maxBytes}
	for range 6 {
		s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
	}

	for seq, found := range map[uint64]bool{0: false, 2: false, 3: true, 6: true, 7: false, math.MaxUint64: false} {
		r := s.Get(seq)
		require.Equal(t, found, r != nil, "seq %d", seq)
		if found {
			require.Equal(t, seq, r.Seq)
		}
	}
}

func TestRecordKeepsTheFirstValueOfKnownHeaders(t *testing.T) {
	t.Parallel()
	c := capture("dev", "/v1/track", `{"userId":"u1"}`)
	c.Request.Header = http.Header{
		"User-Agent":       {"sdk/1", "sdk/2"},
		"Content-Type":     {"application/json"},
		"Content-Encoding": {"gzip"},
		"Origin":           {strings.Repeat("o", 2000)},
		"Anonymousid":      {"a1"},
		"X-Forwarded-For":  {"203.0.113.7"},
		"Authorization":    {"Basic ZGV2Og=="},
		"Cookie":           {"session=secret"},
		"X":                make([]string, 100_000),
	}
	c.Header = http.Header{
		"Content-Type":                {"text/plain; charset=utf-8"},
		"Access-Control-Allow-Origin": make([]string, 100_000),
	}

	rec := newRecord(c)

	require.Equal(t, map[string]string{
		"User-Agent":       "sdk/1",
		"Content-Type":     "application/json",
		"Content-Encoding": "gzip",
		"Origin":           strings.Repeat("o", maxHeaderValue),
		"Anonymousid":      "a1",
		"X-Forwarded-For":  "203.0.113.7",
	}, rec.Request.Headers)
	require.Equal(t, 1+1+1+100_000, rec.Request.DroppedHeaders)
	require.Equal(t, map[string]string{"Content-Type": "text/plain; charset=utf-8", "Access-Control-Allow-Origin": ""}, rec.Response.Headers)
	require.Less(t, rec.size, 8<<10)
}

func TestRecordMasksTheWriteKey(t *testing.T) {
	t.Parallel()
	const long = "fake-write-key-for-tests"
	masked := WriteKey{Key: "fake...ests", Sha256: "42e968f7a0534a276deb5e166babe865d2de755b52506527a924362d0aaedfc4"}

	for _, tc := range []struct {
		name, key, target string
		want              WriteKey
		wantTarget        string
		wantEchoKey       string
	}{
		{
			name: "short key in clear", key: "web", target: "/beacon/v1/batch?writeKey=web",
			want: WriteKey{Key: "web"}, wantTarget: "/beacon/v1/batch?writeKey=web", wantEchoKey: "web",
		},
		{
			name: "eight characters in clear", key: "abcdefgh", target: "/beacon/v1/batch?writeKey=abcdefgh",
			want: WriteKey{Key: "abcdefgh"}, wantTarget: "/beacon/v1/batch?writeKey=abcdefgh", wantEchoKey: "abcdefgh",
		},
		{
			name: "nine characters show no suffix", key: "abcdefghi", target: "/beacon/v1/batch?writeKey=abcdefghi",
			want:       WriteKey{Key: "abcd...", Sha256: "19cc02f26df43cc571bc9ed7b0c4d29224a3ec229529221725ef76d021c8326f"},
			wantTarget: "/beacon/v1/batch?writeKey=abcd...", wantEchoKey: "abcd...",
		},
		{
			name: "long key", key: long, target: "/beacon/v1/batch?writeKey=" + long,
			want: masked, wantTarget: "/beacon/v1/batch?writeKey=fake...ests", wantEchoKey: "fake...ests",
		},
		{
			name: "long key query-escaped", key: "abcd efgh+ijkl", target: "/beacon/v1/batch?writeKey=" + url.QueryEscape("abcd efgh+ijkl"),
			want:       WriteKey{Key: "abcd...ijkl", Sha256: "219befb6a14294555dda70af8bd6334918c775b416bb617a5cb05c7631b8f34c"},
			wantTarget: "/beacon/v1/batch?writeKey=abcd...ijkl", wantEchoKey: "abcd...ijkl",
		},
		{
			name: "key only in the query of a preflight", target: "/sourceConfig/?p=npm&writeKey=" + long + "&v=3",
			wantTarget: "/sourceConfig/?p=npm&writeKey=fake...ests&v=3",
		},
		{
			name: "key before a malformed escape", target: "/beacon/v1/batch?writeKey=2realWriteKeyABCDEFGH%ZZ",
			wantTarget: "/beacon/v1/batch?writeKey=2rea...H%25ZZ",
		},
		{
			name: "key escaped differently from the sent one", key: long, target: "/sourceConfig/?writeKey=%66ake-write-key-for-tests&writeKey=" + long,
			want: masked, wantTarget: "/sourceConfig/?writeKey=fake...ests&writeKey=fake...ests", wantEchoKey: "fake...ests",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := `{"writeKey":"` + tc.key + `"}`
			c := capture(tc.key, tc.target, body)
			c.ResponseBody = []byte(`{"source":{"writeKey":"` + tc.key + `"}}`)

			rec := newRecord(c)

			require.Equal(t, tc.want, rec.WriteKey)
			require.Equal(t, tc.wantTarget, rec.Request.Target)
			require.Equal(t, `{"source":{"writeKey":"`+tc.wantEchoKey+`"}}`, rec.Response.Body)
			require.Equal(t, body, string(rec.Request.Body), "the body keeps the sent bytes")
			require.Equal(t, body, string(rec.Events[0].Message), "events keep the sent bytes")
		})
	}
}

// The /sourceConfig answer is json.Marshal output, which escapes some
// characters of the key, so the raw key is not in it.
func TestRecordMasksAJSONEscapedKeyInTheResponse(t *testing.T) {
	t.Parallel()
	const key = "2abc<secretsecret>wxyz"
	c := capture(key, "/sourceConfig", "")
	answer, err := json.Marshal(map[string]any{"source": map[string]string{"writeKey": key}})
	require.NoError(t, err)
	require.NotContains(t, string(answer), key)
	c.ResponseBody = answer

	rec := newRecord(c)

	require.Equal(t, `{"source":{"writeKey":"2abc...wxyz"}}`, rec.Response.Body)
}

func TestRecordKeepsNoSliceOfTheCredentials(t *testing.T) {
	t.Parallel()
	credentials := "dev:" + strings.Repeat("p", 512<<10)
	c := capture(credentials[:3], "/v1/track", `{"userId":"u1"}`)

	rec := newRecord(c)

	require.Equal(t, "dev", rec.WriteKey.Key)
	require.NotSame(t, unsafe.StringData(credentials), unsafe.StringData(rec.WriteKey.Key))
}

// Events are slices of the body, so the record charges the body's backing
// array once and each event built from a query on its own.
func TestRecordChargesTheBytesItKeeps(t *testing.T) {
	t.Parallel()
	key := strings.Repeat("k", 64)
	body := []byte(`{"userId":"u","p":"` + strings.Repeat(key, 1000) + `"}`)
	decoded := make([]byte, len(body), 2*len(body))
	copy(decoded, body)

	c := capture(key, "/v1/track", string(body))
	c.Decoded = decoded
	c.Events = []ingest.Event{{Message: decoded}}
	require.GreaterOrEqual(t, newRecord(c).size, cap(decoded))

	pixel := capture("dev", "/pixel/v1/track?writeKey=dev", "")
	pixel.Decoded = nil
	pixel.Events = []ingest.Event{{Message: body, FromQuery: true}}
	require.GreaterOrEqual(t, newRecord(pixel).size, len(body))
}

func TestRecordKeepsLittleOfARefusedBody(t *testing.T) {
	t.Parallel()
	huge := strings.Repeat("a", 2_100_000)

	refused := capture("dev", "/v1/track", huge)
	refused.StatusCode = http.StatusRequestEntityTooLarge
	refused.BodyComplete = false
	refused.Decoded = nil
	refused.Events = nil
	rec := newRecord(refused)

	require.Equal(t, 2_100_000, rec.Request.BodyBytes)
	require.Len(t, rec.Request.Body, maxRefusedBody)
	require.Equal(t, maxRefusedBody, cap(rec.Request.Body), "the copy lets the 2 MB array go")
	require.False(t, rec.Request.BodyComplete)
	require.Less(t, rec.size, 8<<10)

	// A body refused at the route or auth step never reached the parser, even
	// though it was read in full.
	for _, stage := range []string{"route", "auth"} {
		early := capture("dev", "/x", huge)
		early.StatusCode = http.StatusNotFound
		early.Events = nil
		early.Rejection = &ingest.Rejection{Stage: stage}
		rec := newRecord(early)
		require.Len(t, rec.Request.Body, maxRefusedBody, stage)
		require.False(t, rec.Request.BodyComplete, stage)
	}

	// A refusal at a parsing step keeps the body, because the events point
	// into it and the sender needs to see where it broke.
	parsed := capture("dev", "/v1/batch", huge)
	parsed.StatusCode = http.StatusBadRequest
	parsed.Rejection = &ingest.Rejection{Stage: "identity"}
	require.Len(t, newRecord(parsed).Request.Body, len(huge))

	// A body that was read in full but failed the parser keeps its bytes, so
	// the sender can see where the JSON broke.
	broken := capture("dev", "/v1/track", huge)
	broken.StatusCode = http.StatusBadRequest
	broken.Events = nil
	broken.Rejection = &ingest.Rejection{Stage: "parse"}
	require.Len(t, newRecord(broken).Request.Body, len(huge))

	// An accepted request is captured whole.
	require.Len(t, newRecord(capture("dev", "/v1/track", huge)).Request.Body, len(huge))
}

func TestRecordCapsTheTarget(t *testing.T) {
	t.Parallel()
	long := "/pixel/v1/track?x=" + strings.Repeat("a", 512<<10)

	// An accepted pixel request keeps its whole query, the only raw copy of
	// what the SDK sent.
	require.Len(t, newRecord(capture("dev", long, "")).Request.Target, len(long))

	refused := capture("dev", long, "")
	refused.Rejection = &ingest.Rejection{Stage: "identity"}
	rec := newRecord(refused)
	require.Len(t, rec.Request.Target, maxTarget)
	require.True(t, strings.HasPrefix(rec.Request.Target, "/pixel/v1/track?x="))
}

func TestStoreIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()
	const writers, perWriter = 8, 250
	s := &Store{maxRequests: 1000, maxBytes: maxBytes}

	var (
		readers, writersWG sync.WaitGroup
		done               = make(chan struct{})
		gap                atomic.Bool
	)
	readers.Go(func() {
		for {
			select {
			case <-done:
				return
			default:
				records, _ := s.Since(0)
				for i := 1; i < len(records); i++ {
					gap.CompareAndSwap(false, records[i].Seq != records[i-1].Seq+1)
				}
			}
		}
	})
	for range writers {
		writersWG.Go(func() {
			for range perWriter {
				s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
			}
		})
	}
	writersWG.Wait()
	close(done)
	readers.Wait()

	require.False(t, gap.Load(), "a reader saw seqs out of order")

	records, evictedThrough := s.Since(0)
	require.Len(t, records, 1000)
	require.Equal(t, uint64(writers*perWriter), records[len(records)-1].Seq)
	require.Equal(t, uint64(writers*perWriter-1000), evictedThrough)
	total := 0
	for _, r := range records {
		total += r.size
	}
	require.Equal(t, total, s.bytes)
}
