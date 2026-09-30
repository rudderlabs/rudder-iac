package store_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

func TestAppendAssignsSeqFromOne(t *testing.T) {
	t.Parallel()

	s := store.New("abc")

	first := s.Append(store.Record{Kind: "ingestion"})
	second := s.Append(store.Record{Kind: "control"})

	require.Equal(t, store.Record{RecordVersion: 1, ServerID: "abc", Seq: 1, Kind: "ingestion"}, first)
	require.Equal(t, uint64(2), second.Seq)
	require.Equal(t, uint64(2), s.Cursor())
}

func TestSinceReturnsRecordsAfterCursor(t *testing.T) {
	t.Parallel()

	s := store.New("abc")
	for range 3 {
		s.Append(store.Record{Kind: "ingestion"})
	}

	view := s.Since(1)

	require.Equal(t, uint64(3), view.Cursor)
	require.Equal(t, []uint64{2, 3}, seqs(view.Records))
	require.Empty(t, s.Since(3).Records)
}

func seqs(records []store.Record) []uint64 {
	out := make([]uint64, 0, len(records))
	for _, r := range records {
		out = append(out, r.Seq)
	}
	return out
}

func TestSinceChangedClosesOnAppend(t *testing.T) {
	t.Parallel()

	s := store.New("abc")
	view := s.Since(0)
	require.NotNil(t, view.Changed)

	select {
	case <-view.Changed:
		t.Fatal("changed closed before any append")
	default:
	}

	s.Append(store.Record{Kind: "ingestion"})

	select {
	case <-view.Changed:
	case <-time.After(time.Second):
		t.Fatal("changed not closed after append")
	}
}

func TestCloseSignalsDone(t *testing.T) {
	t.Parallel()

	s := store.New("abc")
	done := s.Done()

	s.Close()
	s.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("done not closed after Close")
	}
}

func TestGetAndReset(t *testing.T) {
	t.Parallel()
	s := store.New("id")
	s.Append(store.Record{Kind: "control"})
	s.Append(store.Record{Kind: "ingestion"})

	got, ok := s.Get(2)
	require.True(t, ok)
	require.Equal(t, "ingestion", got.Kind)
	_, ok = s.Get(3)
	require.False(t, ok)

	require.Len(t, s.Reset(), 2)
	_, ok = s.Get(1)
	require.False(t, ok)
	require.Equal(t, uint64(2), s.Cursor())
	require.Equal(t, uint64(3), s.Append(store.Record{}).Seq, "seq keeps counting after reset")
	require.Empty(t, s.Since(0).Records[:0])
}

func TestRecordCapEvictsTheOldestWholeRequests(t *testing.T) {
	t.Parallel()
	s := store.New("id", store.WithLimits(2, 1<<20))

	for range 3 {
		s.Append(store.Record{Kind: "ingestion"})
	}

	require.Equal(t, []uint64{2, 3}, seqs(s.Since(0).Records))
	require.Equal(t, store.Stats{Requests: 2, Evicted: 1, EvictedThrough: 1, Bytes: s.Stats().Bytes}, s.Stats())
}

func TestByteCapEvictsUntilTheTotalFits(t *testing.T) {
	t.Parallel()
	s := store.New("id", store.WithLimits(100, 250))
	body := strings.Repeat("x", 100)

	for range 4 {
		s.Append(store.Record{Kind: "ingestion", Request: store.Request{Body: body}})
	}

	require.Equal(t, []uint64{3, 4}, seqs(s.Since(0).Records))
	require.Equal(t, store.Stats{Requests: 2, Bytes: 200, Evicted: 2, EvictedThrough: 2}, s.Stats())
	_, ok := s.Get(2)
	require.False(t, ok)
}

func TestStatsCountBodiesMessagesAndHeaders(t *testing.T) {
	t.Parallel()
	s := store.New("id")

	s.Append(store.Record{
		Request: store.Request{Target: "/v1/t", Body: "abc", BodyBase64: []byte("zz"),
			Headers: http.Header{"Ab": {"cd"}}},
		Response: store.Response{Body: "ok", Headers: http.Header{"X": {"y"}}},
		Events:   []store.Event{{Message: json.RawMessage(`{}`), EnrichedMessage: json.RawMessage(`{"a":1}`)}},
	})

	// 5 target + 3 body + 2 base64 + 2 response + 2+7 messages + 4+2 headers
	require.Equal(t, 27, s.Stats().Bytes)
	require.Equal(t, store.Stats{Requests: 1, Bytes: 27}, s.Stats())
	maxRecords, maxBytes := s.Limits()
	require.Equal(t, store.DefaultMaxRecords, maxRecords)
	require.Equal(t, store.DefaultMaxBytes, maxBytes)
}
