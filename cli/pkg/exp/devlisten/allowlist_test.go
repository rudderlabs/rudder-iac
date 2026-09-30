package devlisten_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

func TestWriteKeyAllowlistShowsInReadyAndInfo(t *testing.T) {
	t.Parallel()
	s := startServer(t, devlisten.WithWriteKeys("frontendKey12345", "backendKey123456"))

	ready := s.Ready()
	info, err := s.Client().Info(context.Background())

	require.NoError(t, err)
	require.Equal(t, "allowlist", ready.WriteKeyPolicy)
	require.Equal(t, "fron...2345", ready.WriteKey, "the ready line never prints a real key")
	require.Equal(t, []string{"fron...2345", "back...3456"}, info.WriteKeys)
}

func TestWriteKeyAllowlistRejectsTheDevKey(t *testing.T) {
	t.Parallel()
	s := startServer(t, devlisten.WithWriteKeys("frontendKey12345"))

	req, err := http.NewRequest(http.MethodPost, s.URL()+"/v1/track", strings.NewReader(`{"event":"A","userId":"u1"}`))
	require.NoError(t, err)
	req.SetBasicAuth("dev", "")
	resp, err := testClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
