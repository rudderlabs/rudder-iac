package retlconnection

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// renderSync prints one run the way the webapp's sync page reads it: the
// outcome rather than the raw status, a human duration, and the row counts that
// decide the difference between the two.
func renderSync(sync retlClient.Sync, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sync %s  %s\n", sync.ID, sync.Outcome())
	fmt.Fprintf(&b, "  started   %s\n", sync.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "  duration  %s\n", compactDuration(sync.Duration(now)))
	fmt.Fprintf(&b, "  rows      %d changed, %d delivered, %d failed\n",
		sync.Metrics.Changed.Total, sync.Metrics.Succeeded.Total, sync.Metrics.Failed.Total)
	if sync.Error != "" {
		// The reason is the point of showing a failed run at all, so it is
		// never truncated or folded into the status line.
		fmt.Fprintf(&b, "  error     %s\n", sync.Error)
	}
	return b.String()
}

// compactDuration renders a duration the way a run log wants it — "1m 30s",
// not "1m30.000000123s".
func compactDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// renderSyncTable prints a run history the way the webapp's past-syncs tab
// reads it: the outcome rather than the raw status, when it started in relative
// terms, how long it took, and the two row counts that separate a clean run
// from one that dropped records.
func renderSyncTable(syncs []retlClient.Sync, now time.Time) string {
	const format = "%-24s  %-10s  %-12s  %-10s  %8s  %8s\n"

	var b strings.Builder
	fmt.Fprintf(&b, format, "RUN ID", "OUTCOME", "STARTED", "DURATION", "CHANGED", "FAILED")
	for _, sync := range syncs {
		fmt.Fprintf(&b, format,
			sync.ID,
			sync.Outcome(),
			relativeTime(sync.StartedAt, now),
			compactDuration(sync.Duration(now)),
			strconv.Itoa(sync.Metrics.Changed.Total),
			strconv.Itoa(sync.Metrics.Failed.Total),
		)
	}
	return b.String()
}

// relativeTime is how long ago something happened. A run history is read to
// answer "when did this last go wrong", and an absolute timestamp makes the
// reader do the subtraction. The JSON form keeps the absolute time.
func relativeTime(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}

	d := now.Sub(t)
	if d < 0 {
		return "just now"
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
