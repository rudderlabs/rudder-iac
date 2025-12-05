package tasker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestErrTaskFailed(t *testing.T) {

	t.Run("should return the correct error message", func(t *testing.T) {
		t.Parallel()

		err := &ErrTaskFailed{
			TaskID: "task-1",
			Err:    errors.New("test error"),
		}

		require.EqualError(t, err, "task: task-1 failed: test error")
	})

	t.Run("should handle unwrapping the error correctly", func(t *testing.T) {
		t.Parallel()

		err := &ErrTaskFailed{
			TaskID: "task-1",
			Err:    io.EOF, // assume we have an EOF error from task which we wrapped
		}

		// ErrorIs validates that Unwrap() correctly exposes the wrapped error
		require.ErrorIs(t, err, io.EOF)
	})
}

func TestErrTaskCancelled(t *testing.T) {

	t.Run("should return the correct error message", func(t *testing.T) {
		t.Parallel()

		// With dependency
		err := &ErrTaskCancelled{
			TaskID:     "task-1",
			Dependency: lo.ToPtr("task-2"),
			Err:        errors.New("test error"),
		}
		require.EqualError(t, err, "task: task-1 cancelled: dependency: task-2 failed: test error")

		// Without dependency
		err = &ErrTaskCancelled{
			TaskID: "task-2",
			Err:    errors.New("test error"),
		}
		require.EqualError(t, err, "task: task-2 cancelled: error: test error")
	})

	t.Run("should handle unwrapping the error correctly", func(t *testing.T) {
		t.Parallel()

		err := &ErrTaskCancelled{
			TaskID: "task-1",
			Err:    fmt.Errorf("parent context cancelled: %w", context.Canceled),
		}

		// ErrorIs validates that Unwrap() correctly exposes the wrapped error chain
		require.ErrorIs(t, err, context.Canceled)
	})

}
