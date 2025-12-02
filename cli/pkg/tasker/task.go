package tasker

import (
	"context"
	"fmt"
	"sync"

	"github.com/samber/lo"
)

type (
	taskResultMap = map[string]*taskResult
	semaphore     = chan struct{}
)

type taskResult struct {
	err  error
	done chan bool
}

type Task interface {
	Id() string
	Dependencies() []string
}

type job struct {
	tasks       []Task
	taskResults taskResultMap // To track whether a task has completed.
	semaphore   semaphore     // To control number of tasks running concurrently
}

// RunTasks creates a job and executes it. This is the main entry point for task execution.
func RunTasks(ctx context.Context, tasks []Task, concurrency int, _ bool, command func(task Task) error) []error {
	// Find out duplicate tasks by their Id
	// as we depend on the task id to be unique
	duplicates := lo.FindDuplicatesBy(tasks, func(item Task) string {
		return item.Id()
	})

	if len(duplicates) > 0 {
		return []error{fmt.Errorf("duplicate tasks found: %v", lo.Map(duplicates, func(item Task, _ int) string {
			return item.Id()
		}))}
	}

	job := newJob(tasks, concurrency)
	return job.run(ctx, command)
}

func newJob(tasks []Task, concurrency int) *job {
	semaphore := make(chan struct{}, concurrency)
	for range concurrency {
		semaphore <- struct{}{}
	}

	job := &job{
		semaphore:   semaphore,
		tasks:       tasks,
		taskResults: make(taskResultMap),
	}

	for _, task := range job.tasks {
		job.taskResults[task.Id()] = &taskResult{done: make(chan bool, 1)}
	}
	return job
}

func (job *job) run(ctx context.Context, command func(task Task) error) []error {
	var errors []error
	var errorsMutex sync.Mutex

	var wg sync.WaitGroup
	for _, task := range job.tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			if err := job.runTask(ctx, t, command); err != nil {
				errorsMutex.Lock()
				errors = append(errors, err)
				errorsMutex.Unlock()
			}
		}(task)
	}
	wg.Wait()

	errorsMutex.Lock()
	defer errorsMutex.Unlock()
	return errors
}

func (job *job) runTask(ctx context.Context, task Task, command func(task Task) error) (err error) {
	semaphoreAcquired := false

	defer func() {
		r := job.taskResults[task.Id()]
		if err != nil {
			// set the error on the task result
			// so that the unblocked goroutines
			// can check if the task has failed
			r.err = err
		}

		if semaphoreAcquired {
			select {
			case job.semaphore <- struct{}{}:
			default:
			}
		}

		close(r.done)
	}()

	// Wait for all dependencies to report their results
	dependencies := task.Dependencies()
	for _, depTaskId := range dependencies {
		result, ok := job.taskResults[depTaskId]
		// No need to return error if dependency not found in task list
		// This can happen if there are no operations for that dependency
		if ok {
			<-result.done
			if result.err != nil {
				// If any one of my dependencies fail, I should fail.
				return fmt.Errorf("dependency %s failed: %w", depTaskId, result.err)
			}
		}
	}

	<-job.semaphore
	semaphoreAcquired = true

	// Once you have acquired the semaphore, you need to check if
	// caller context is done due to cancellation.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return command(task)
}
