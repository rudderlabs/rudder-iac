package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzBody sends any body, plain or declared gzip, to a single-event and a
// batch route. Every answer must be an answer RudderStack gives, and an accepted
// request must carry its decoded events.
func FuzzBody(f *testing.F) {
	f.Add([]byte(`{"userId":"u1"}`), false, false)
	f.Add([]byte(`{"batch":[{"userId":"u1"},{"anonymousId":"a1"}]}`), false, true)
	f.Add(gz(f, `{"batch":[{"userId":"u1"}]}`), true, true)
	f.Add(gz(f, `{"userId":"u1"}`)[:20], true, false)
	f.Add([]byte(`{"batch":[{"userId":"u1"},1]}`), false, true)
	f.Add([]byte(""), false, false)

	answers := map[string]bool{"ok": true}
	for _, e := range []gwError{
		errUncompress, errRequestBodyTooLarge, errRequestBodyNil, errRequestBodyReadFailed, errInvalidJSON,
		errNotRudderEvent, errEmptyBatch, errNonIdentifiable,
	} {
		answers[e.message+"\n"] = true
	}

	f.Fuzz(func(t *testing.T, body []byte, gzipped, batch bool) {
		path := "/v1/track"
		if batch {
			path = "/v1/batch"
		}
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		r.SetBasicAuth("dev", "")
		if gzipped {
			r.Header.Set("Content-Encoding", "gzip")
		}
		h, sink := newTestHandler()

		rec := serve(h, r)

		require.True(t, answers[rec.Body.String()], "unexpected answer %d %q", rec.Code, rec.Body.String())
		c := sink.only(t)
		require.LessOrEqual(t, len(c.Decoded), maxReqSize)
		if rec.Code != http.StatusOK {
			require.NotNil(t, c.Rejection)
			return
		}
		require.NotEmpty(t, c.Events)
		for _, ev := range c.Events {
			require.True(t, json.Valid(ev.Message), "event %q", ev.Message)
		}
	})
}

// FuzzQueryValue checks that the write-key scan reads the same value as
// url.ParseQuery.
func FuzzQueryValue(f *testing.F) {
	for _, q := range []string{
		"writeKey=dev", "a=1&writeKey=dev&writeKey=other", "writeKey=%zz&writeKey=ok",
		"%77riteKey=a+b%21", "writeKey=a;b&writeKey=c", "writeKey", "&&writeKey=&x=1", "writekey=dev",
	} {
		f.Add(q)
	}
	f.Fuzz(func(t *testing.T, rawQuery string) {
		want, _ := url.ParseQuery(rawQuery)
		require.Equal(t, want.Get("writeKey"), queryValue(rawQuery, "writeKey"))
	})
}
