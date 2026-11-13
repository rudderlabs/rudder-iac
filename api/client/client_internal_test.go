package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryAfterDelay(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value string
		retry int
		want  time.Duration
	}{
		{name: "seconds", value: "2", want: 2 * time.Second},
		{name: "zero", value: "0", want: 0},
		{name: "minute-long window", value: "60", want: time.Minute},
		{name: "seconds above cap", value: "3600", want: maxRateLimitWait},
		{name: "seconds that overflow a duration", value: "9223372036854775807", want: maxRateLimitWait},
		{name: "negative", value: "-1", want: baseRateLimitWait},
		{name: "missing", value: "", want: baseRateLimitWait},
		{name: "invalid", value: "soon", want: baseRateLimitWait},
		{name: "missing on a later retry grows", value: "", retry: 4, want: 32 * time.Second},
		{name: "invalid on a later retry grows", value: "soon", retry: 2, want: 8 * time.Second},
		{name: "missing growth is capped", value: "", retry: 10, want: maxRateLimitWait},
		{name: "missing growth stays capped at the last retry", value: "", retry: maxRateLimitRetries, want: maxRateLimitWait},
		{name: "header ignores retry", value: "2", retry: 4, want: 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, retryAfterDelay(tc.value, tc.retry))
		})
	}
}
