package store

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

func one(*Record) int { return 1 }

func TestStoreStats(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 3, maxBytes: maxBytes}
	require.Equal(t, Stats{MaxRequests: 3, MaxBytes: maxBytes}, s.Stats())
	require.Zero(t, s.Cursor())

	control := capture("dev", "/sourceConfig", "")
	control.Kind, control.Events = "control", nil
	batch := capture("dev", "/v1/batch", `{"batch":[{"userId":"u1"},{"userId":"u2"}]}`)
	batch.Events = append(batch.Events, batch.Events[0])
	for _, c := range []*ingest.Capture{
		capture("dev", "/v1/track", `{"userId":"u1"}`), control, batch, capture("dev", "/v1/track", `{"userId":"u1"}`),
	} {
		s.Capture(c)
	}

	records, _ := s.Since(0)
	bytes := 0
	for _, r := range records {
		bytes += r.size
	}
	require.Equal(t, Stats{
		Requests: 2, Events: 3, Control: 1, Bytes: bytes,
		Evicted: 1, EvictedThrough: 1, MaxRequests: 3, MaxBytes: maxBytes,
	}, s.Stats())
	require.Equal(t, uint64(4), s.Cursor())
}

func TestStoreWait(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		stored   int
		since    uint64
		min      int
		appended int
		want     int
	}{
		{name: "enough stored", stored: 3, min: 2, want: 3},
		{name: "counts after since only", stored: 3, since: 2, min: 1, want: 1},
		{name: "wakes on append", stored: 1, min: 3, appended: 2, want: 3},
		{name: "wakes on each append", stored: 0, min: 2, appended: 2, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Store{maxRequests: 100, maxBytes: maxBytes}
			for range tc.stored {
				s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
			}
			// Append only once the waiter blocks, so the test takes the wake path.
			waiting := make(chan struct{}, tc.appended+1)
			s.onWait = func() { waiting <- struct{}{} }
			go func() {
				for range tc.appended {
					<-waiting
					s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
				}
			}()

			found, err := s.Wait(context.Background(), tc.since, tc.min, one)

			require.NoError(t, err)
			require.Equal(t, tc.want, found)
		})
	}
}

func TestStoreWaitStopsWithTheContext(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 100, maxBytes: maxBytes}
	s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
	ctx, cancel := context.WithCancel(context.Background())
	s.onWait = cancel

	found, err := s.Wait(ctx, 0, 2, one)

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, found)
}

// A match that eviction removed during the wait no longer counts, so the
// summary that follows the wait never shows fewer matches than the wait saw.
func TestStoreWaitCountsOnlyStoredMatches(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 1, maxBytes: maxBytes}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parked := 0
	s.onWait = func() {
		parked++
		switch parked {
		case 1, 2:
			go s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
		default:
			cancel()
		}
	}

	found, err := s.Wait(ctx, 0, 2, one)

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, found)
	require.Equal(t, 3, parked)
}

// A waiter keeps its scan position, so a wake costs the new records only.
// The one recount when the sum is reached adds a second pass.
func TestStoreWaitScansEachRecordOnce(t *testing.T) {
	t.Parallel()
	s := &Store{maxRequests: 100, maxBytes: maxBytes}
	for range 10 {
		s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
	}
	waiting := make(chan struct{}, 1)
	s.onWait = func() { waiting <- struct{}{} }
	scanned := map[uint64]int{}
	go func() {
		for range 3 {
			<-waiting
			s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
		}
	}()

	_, err := s.Wait(context.Background(), 0, 13, func(r *Record) int {
		scanned[r.Seq]++
		return 1
	})

	require.NoError(t, err)
	require.Len(t, scanned, 13)
	for seq, n := range scanned {
		require.Equal(t, 2, n, "seq %d", seq)
	}
}

// appendLatency stores records, parks waiters on since=0, then returns the
// mean time from an append until every waiter has scanned it and blocks
// again, and the number of records the waiters counted.
func appendLatency(tb testing.TB, records, waiters, appends int) (time.Duration, int64) {
	tb.Helper()
	s := &Store{maxRequests: records + appends, maxBytes: 1 << 40}
	for range records {
		s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
	}
	var parked, counted atomic.Int64
	s.onWait = func() { parked.Add(1) }
	count := func(*Record) int {
		counted.Add(1)
		return 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for range waiters {
		wg.Go(func() { _, _ = s.Wait(ctx, 0, records+appends+1, count) })
	}
	settle := func(want int64) {
		for parked.Load() < want {
			time.Sleep(10 * time.Microsecond)
		}
	}
	settle(int64(waiters))

	start := time.Now()
	for i := range appends {
		s.Capture(capture("dev", "/v1/track", `{"userId":"u1"}`))
		settle(int64(waiters * (i + 2)))
	}
	elapsed := time.Since(start)

	cancel()
	wg.Wait()
	return elapsed / time.Duration(appends), counted.Load()
}

// An append costs each waiter the new record only. The test counts the scan
// calls rather than timing them, because a loaded CI runner slows the clock
// but not the count; BenchmarkStoreWaitAppend keeps the timing.
func TestStoreWaitAppendScansOnlyTheNewRecord(t *testing.T) {
	t.Parallel()
	const records, waiters, appends = 10_000, 20, 50
	_, counted := appendLatency(t, records, waiters, appends)
	require.LessOrEqual(t, counted, int64(waiters*(records+2*appends)))
}

func BenchmarkStoreWaitAppend(b *testing.B) {
	latency, _ := appendLatency(b, 10_000, 20, max(b.N, 1))
	b.ReportMetric(float64(latency.Microseconds()), "µs/append")
}
