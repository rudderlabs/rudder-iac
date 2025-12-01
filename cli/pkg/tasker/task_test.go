package tasker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

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

	expectedErr := errors.New("task-a failed")

	t.Run("continueOnFail=false", func(t *testing.T) {
		t.Parallel()

		queue := &safeQueue{}

		// task-A fails, task-B depends on task-A
		tasks := []Task{
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

		errs := RunTasks(context.Background(), tasks, 1, false, command)
		require.NotEmpty(t, errs, "Expected error from task-a")
		assert.Contains(t, errs, expectedErr, "Error slice should contain expected error")

		items := queue.Items()
		assert.Empty(t, items, "Queue should be empty as task-b should not execute")
	})

	t.Run("continueOnFail=true", func(t *testing.T) {
		t.Parallel()

		queue := &safeQueue{}

		// task-A fails, task-B depends on task-A
		tasks := []Task{
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

		errs := RunTasks(context.Background(), tasks, 1, true, command)

		require.NotEmpty(t, errs, "Expected error from task-a")
		assert.Contains(t, errs, expectedErr, "Error slice should contain expected error")

		items := queue.Items()
		require.Len(t, items, 1, "Queue should contain only task-b")
		assert.Equal(t, "task-b", items[0], "Queue should contain task-b")
	})
}
