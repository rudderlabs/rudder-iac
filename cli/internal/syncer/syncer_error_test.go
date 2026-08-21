package syncer

import (
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
	"github.com/stretchr/testify/require"
)

func TestFirstActionableErrorSkipsOverallJobFailedCancellation(t *testing.T) {
	providerErr := errors.New("simulated failure for event2")

	err := firstActionableError([]error{
		&tasker.ErrTaskCancelled{
			TaskID: "property:property1",
			Err:    errors.New("overall job failed"),
		},
		&tasker.ErrTaskFailed{
			TaskID: "event:event2",
			Err:    &OperationError{Err: providerErr},
		},
	})

	require.ErrorIs(t, err, providerErr)
}

func TestFirstActionableErrorReturnsCancellationWhenOnlyError(t *testing.T) {
	cancelledErr := &tasker.ErrTaskCancelled{
		TaskID: "property:property1",
		Err:    errors.New("overall job failed"),
	}

	err := firstActionableError([]error{cancelledErr})

	require.Same(t, cancelledErr, err)
}
