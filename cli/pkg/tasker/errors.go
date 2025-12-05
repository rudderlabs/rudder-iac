package tasker

import "fmt"

type ErrTaskFailed struct {
	TaskID string
	Err    error
}

func (e *ErrTaskFailed) Error() string {
	return fmt.Sprintf("task: %s failed: %s", e.TaskID, e.Err.Error())
}

func (e *ErrTaskFailed) Unwrap() error {
	return e.Err
}

type ErrTaskCancelled struct {
	TaskID     string
	Dependency *string
	Err        error
}

func (e *ErrTaskCancelled) Error() string {
	if e.Dependency != nil {
		return fmt.Sprintf(
			"task: %s cancelled: dependency: %s failed: %s",
			e.TaskID,
			*e.Dependency,
			e.Err.Error(),
		)
	}

	return fmt.Sprintf("task: %s cancelled: error: %s", e.TaskID, e.Err.Error())
}

func (e *ErrTaskCancelled) Unwrap() error {
	return e.Err
}
