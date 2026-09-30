package devlistentest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

func TestStartServesUntilCleanup(t *testing.T) {
	t.Parallel()
	var client interface{ URL() string }

	t.Run("inner", func(t *testing.T) {
		s := devlistentest.Start(t)
		info, err := s.Client().Info(context.Background())
		require.NoError(t, err)
		require.Equal(t, s.Ready().ServerID, info.ServerID)
		client = s.Client()
	})

	_, err := devlisten.NewClient(client.URL()).Info(context.Background())
	require.Error(t, err, "the cleanup closed the server")
}
