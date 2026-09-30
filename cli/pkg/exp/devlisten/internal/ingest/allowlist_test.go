package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// With an allowlist, a key outside it is refused at the auth stage the way
// the oracle refuses an unknown write key, and the refusal is captured.
func TestAllowlistRejectsAnUnlistedKey(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	g.AllowWriteKeys([]string{"frontendKey12345"})

	got := send(t, g, post("/v1/track", `{"event":"A","userId":"u1"}`))

	require.Equal(t, http.StatusUnauthorized, got.status)
	require.Equal(t, "invalid write key\n", got.body)
	rec := onlyRecord(t, st)
	require.Equal(t, "auth", rec.Rejection.Stage)
	require.True(t, rec.Failed)
}

func TestAllowlistRejectsAnUnlistedKeyOnSourceConfig(t *testing.T) {
	t.Parallel()
	g, st := newTestGateway()
	g.AllowWriteKeys([]string{"frontendKey12345"})
	r := httptest.NewRequest(http.MethodGet, "/sourceConfig?p=npm", nil)
	r.SetBasicAuth("otherKey12345678", "")

	got := send(t, g, r)

	require.Equal(t, http.StatusUnauthorized, got.status)
	require.Equal(t, `{"message":"Invalid write key"}`, got.body)
	require.True(t, onlyRecord(t, st).Failed)
}
