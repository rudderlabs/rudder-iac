package tasker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTask struct {
	id           string
	dependencies []string
}

func (t *mockTask) Id() string {
	return t.id
}

func (t *mockTask) Dependencies() []string {
	return t.dependencies
}

type safeQueue struct {
	sync.Mutex
	items []string
}

func (q *safeQueue) Push(item string) {
	q.Lock()
	defer q.Unlock()
	q.items = append(q.items, item)
}

func (q *safeQueue) Items() []string {
	q.Lock()
	defer q.Unlock()
	return append([]string{}, q.items...)
}

func (q *safeQueue) Clear() {
	q.Lock()
	defer q.Unlock()
	q.items = q.items[:0]
}

func TestRunTasks_WithoutDependencies(t *testing.T) {
	t.Parallel()

	numTasks := 100
	concurrency := 10
	queue := &safeQueue{}

	tasks := make([]Task, numTasks)
	for i := range numTasks {
		tasks[i] = &mockTask{id: fmt.Sprintf("task-%d", i)}
	}

	errs := RunTasks(context.Background(), tasks, concurrency, false, func(task Task) error {
		queue.Push(task.Id())
		return nil
	})
	require.Empty(t, errs, "Expected no errors")

	items := queue.Items()
	require.Len(t, items, numTasks, "Expected all tasks to complete")

	// Verify all task IDs are present
	taskMap := make(map[string]bool)
	for _, item := range items {
		taskMap[item] = true
	}
	for i := range numTasks {
		expectedID := fmt.Sprintf("task-%d", i)
		assert.True(t, taskMap[expectedID], "Task %s should be in queue", expectedID)
	}
}

func TestRunTasks_Dependencies(t *testing.T) {
	t.Parallel()

	concurrency := 10
	queue := &safeQueue{}

	// task-A has no deps, task-B depends on task-A, task-C depends on task-B
	tasks := []Task{
		&mockTask{id: "task-c", dependencies: []string{"task-b"}},
		&mockTask{id: "task-b", dependencies: []string{"task-a"}},
		&mockTask{id: "task-a", dependencies: []string{}},
		&mockTask{id: "task-d", dependencies: []string{}}, // Extra task with no dependencies
		&mockTask{id: "task-e", dependencies: []string{}},
	}

	errs := RunTasks(context.Background(), tasks, concurrency, false, func(task Task) error {
		queue.Push(task.Id())
		return nil
	})
	require.Empty(t, errs, "Expected no errors")

	items := queue.Items()
	require.Len(t, items, 5, "Expected all 3 tasks to complete")

	// Verify all tasks are present
	taskMap := make(map[string]bool)
	for _, item := range items {
		taskMap[item] = true
	}

	assert.True(t, taskMap["task-a"], "task-a should be in queue")
	assert.True(t, taskMap["task-b"], "task-b should be in queue")
	assert.True(t, taskMap["task-c"], "task-c should be in queue")
	assert.True(t, taskMap["task-d"], "task-d should be in queue")
	assert.True(t, taskMap["task-e"], "task-e should be in queue")

	// Verify dependency ordering: A appears before B, B appears before C
	indexA := -1
	indexB := -1
	indexC := -1
	for i, item := range items {
		switch item {
		case "task-a":
			indexA = i
		case "task-b":
			indexB = i
		case "task-c":
			indexC = i
		}
	}

	assert.NotEqual(t, -1, indexA, "task-a should be in queue")
	assert.NotEqual(t, -1, indexB, "task-b should be in queue")
	assert.NotEqual(t, -1, indexC, "task-c should be in queue")
	assert.True(t, indexA < indexB, "task-a should appear before task-b")
	assert.True(t, indexB < indexC, "task-b should appear before task-c")
}

func TestRunTasks_ErrorWithDependentTask(t *testing.T) {
	t.Parallel()

	var (
		expectedErr = errors.New("task-a failed")
		concurrency = 2
	)

	t.Run("tasks dependent on a failed task should always fail", func(t *testing.T) {
		t.Parallel()

		queue := &safeQueue{}

		tasks := []Task{
			&mockTask{id: "task-c", dependencies: []string{"task-b"}}, // task-c fails as it's dependent on task-b which failed
			&mockTask{id: "task-b", dependencies: []string{"task-a"}}, // task-b fails as it's dependent on task-a which failed
			&mockTask{id: "task-a", dependencies: []string{}},
		}

		command := func(task Task) error {
			if task.Id() == "task-a" {
				return expectedErr
			}
			queue.Push(task.Id())
			return nil
		}

		errs := RunTasks(context.Background(), tasks, concurrency, false, command)
		require.NotEmpty(t, errs, "Expected error from task-a")
		assert.Contains(t, errs, expectedErr, "Error slice should contain expected error")

		items := queue.Items()
		assert.Empty(t, items, "Queue should be empty as task-b and task-c should not execute")
	})

	t.Run("tasks not dependent on a failed task should execute successfully when continueOnFail=true", func(t *testing.T) {
		t.Parallel()

		queue := &safeQueue{}
		tasks := []Task{
			&mockTask{id: "task-c", dependencies: []string{}}, // task-c is not dependent on any task
			&mockTask{id: "task-b", dependencies: []string{"task-a"}},
			&mockTask{id: "task-a", dependencies: []string{}},
		}

		command := func(task Task) error {
			if task.Id() == "task-a" {
				return expectedErr
			}
			queue.Push(task.Id())
			return nil
		}

		errs := RunTasks(context.Background(), tasks, concurrency, true, command)
		require.NotEmpty(t, errs, "Expected error from task-a")

		items := queue.Items()
		assert.Len(t, items, 1, "Queue should contain only task-c")
		assert.Equal(t, "task-c", items[0], "Queue should contain task-c")
	})

	t.Run("tasks should fail when we have a failed task and when continueOnFail=false", func(t *testing.T) {
		t.Parallel()

		queue := &safeQueue{}
		tasks := []Task{
			&mockTask{id: "task-c", dependencies: []string{}}, // task-c is not dependent on any task
			&mockTask{id: "task-b", dependencies: []string{"task-a"}},
			&mockTask{id: "task-a", dependencies: []string{}},
		}

		command := func(task Task) error {
			if true {
				// always fail the task
				return expectedErr
			}
			queue.Push(task.Id())
			return nil
		}

		errs := RunTasks(context.Background(), tasks, concurrency, false, command)
		require.NotEmpty(t, errs, "Expected error from task-a")
		require.Len(t, errs, 3, "Expected 3 errors from task-a, task-b, and task-c")

		items := queue.Items()
		assert.Empty(t, items, "Queue should be empty as all tasks should fail")
	})

}

func TestRunTasks_WithDuplicateTasks(t *testing.T) {
	t.Parallel()

	t.Run("duplicate tasks added to the task list should return an error", func(t *testing.T) {
		t.Parallel()

		tasks := []Task{
			&mockTask{id: "task-a", dependencies: []string{}},
			&mockTask{id: "task-a", dependencies: []string{}},
		}

		errs := RunTasks(context.Background(), tasks, 1, false, func(task Task) error {
			return nil
		})

		require.NotEmpty(t, errs, "Expected error from duplicate task")
		assert.EqualError(t, errs[0], "duplicate tasks found: [task-a]")
	})
}

func TestRunTasks_ContextCancel(t *testing.T) {
	t.Parallel()

	t.Run("context cancellation should return error", func(t *testing.T) {
		t.Parallel()
		queue := &safeQueue{}

		ctx, cancel := context.WithCancel(context.Background())
		cancelCtxChan := make(chan bool)

		tasks := []Task{
			&mockTask{id: "task-a", dependencies: []string{}},
			&mockTask{id: "task-b", dependencies: []string{"task-a"}},
			&mockTask{id: "task-c", dependencies: []string{}},
		}

		command := func(task Task) error {
			cancelCtxChan <- true
			// Simulate some work in which the context is cancelled
			time.Sleep(100 * time.Millisecond)
			queue.Push(task.Id())

			return nil
		}

		go func() {
			<-cancelCtxChan
			cancel()
		}()

		errs := RunTasks(ctx, tasks, 1, false, command)
		require.NotEmpty(t, errs, "Expected error from context cancellation")
		require.Len(t, errs, 2, "Expected 2 errors from context cancellation")
		assert.ErrorIs(t, errs[0], context.Canceled, "Error slice should contain context canceled error")

		items := queue.Items()
		assert.Len(t, items, 1, "Queue should contain only 1 task whichever ran first and cancelled the context")
	})
}
