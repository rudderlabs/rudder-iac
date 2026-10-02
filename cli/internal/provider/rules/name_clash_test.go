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
			// Folded the way Postgres LOWER() folds, which is what the server
			// check uses: LOWER('İstanbul') drops the combining dot to exactly
			// 'istanbul', so the server rejects this pair and so do we.
			// strings.EqualFold would not report it, and validate would then
			// pass something apply fails.
			name:  "dotted capital I clashes with plain i, as LOWER() has it",
			spec:  "istanbul",
			names: []string{"istanbul", "İstanbul"},
			want:  "duplicate display_name 'istanbul' (case-insensitive match with 'İstanbul')",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, NameClashMessage("display_name", tc.spec, tc.names))
		})
	}
}
