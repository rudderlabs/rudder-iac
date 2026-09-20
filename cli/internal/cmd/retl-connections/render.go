package retlconnection

import (
	"fmt"
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
