package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameClash(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		spec      string
		names     []string
		want      string
		wantClash bool
	}{
		{
			name:  "unique",
			spec:  "Orders",
			names: []string{"Orders", "Users"},
		},
		{
			name:      "exact duplicate",
			spec:      "Orders",
			names:     []string{"Orders", "Orders"},
			want:      "duplicate display_name 'Orders'",
			wantClash: true,
		},
		{
			name:      "case variant",
			spec:      "orders",
			names:     []string{"orders", "Orders"},
			want:      "duplicate display_name 'orders' (case-insensitive match with 'Orders')",
			wantClash: true,
		},
		{
			name:      "several case variants, listed once each in order",
			spec:      "orders",
			names:     []string{"orders", "Orders", "ORDERS", "Orders"},
			want:      "duplicate display_name 'orders' (case-insensitive match with 'ORDERS', 'Orders')",
			wantClash: true,
		},
		{
			// An exact duplicate is the plainer message, whatever else matches.
			name:      "exact duplicate and case variant",
			spec:      "Orders",
			names:     []string{"Orders", "orders", "Orders"},
			want:      "duplicate display_name 'Orders'",
			wantClash: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := NameClash("display_name", tc.spec, tc.names)
			assert.Equal(t, tc.wantClash, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
