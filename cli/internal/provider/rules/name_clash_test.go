package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameClashMessage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		spec  string
		names []string
		want  string
	}{
		{
			name:  "unique",
			spec:  "Orders",
			names: []string{"Orders", "Users"},
		},
		{
			name:  "exact duplicate",
			spec:  "Orders",
			names: []string{"Orders", "Orders"},
			want:  "duplicate display_name 'Orders'",
		},
		{
			name:  "case variant",
			spec:  "orders",
			names: []string{"orders", "Orders"},
			want:  "duplicate display_name 'orders' (case-insensitive match with 'Orders')",
		},
		{
			// One variant is enough to point at the clash, and the smallest is
			// picked so the message does not depend on the graph walk's order.
			name:  "several case variants, smallest named",
			spec:  "orders",
			names: []string{"orders", "Orders", "ORDERS", "Orders"},
			want:  "duplicate display_name 'orders' (case-insensitive match with 'ORDERS')",
		},
		{
			// An exact duplicate is the plainer message, whatever else matches.
			name:  "exact duplicate and case variant",
			spec:  "Orders",
			names: []string{"Orders", "orders", "Orders"},
			want:  "duplicate display_name 'Orders'",
		},
		{
			// Full case mapping, as the control plane's toLowerCase does it:
			// "İstanbul" is not a clash with "istanbul". Go's strings.ToLower
			// folds it to exactly "istanbul" and would report one.
			name:  "dotted capital I is not a clash with plain i",
			spec:  "istanbul",
			names: []string{"istanbul", "İstanbul"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, NameClashMessage("display_name", tc.spec, tc.names))
		})
	}
}
