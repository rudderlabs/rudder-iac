package store_test

import (
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
