package tasker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

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
	once        sync.Once
	failed      atomic.Bool
	tasks       []Task
	taskResults taskResultMap // To track whether a task has completed.
	semaphore   semaphore     // To control number of tasks running concurrently
}

// RunTasks creates a job and executes it. This is the main entry point for task execution.
func RunTasks(ctx context.Context, tasks []Task, concurrency int, continueOnFail bool, command func(task Task) error) []error {
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
	return job.run(ctx, continueOnFail, command)
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

func (job *job) run(ctx context.Context, continueOnFail bool, command func(task Task) error) []error {
	var errors []error
	var errorsMutex sync.Mutex

	var wg sync.WaitGroup
	for _, task := range job.tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			if err := job.runTask(ctx, t, continueOnFail, command); err != nil {
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

func (job *job) runTask(ctx context.Context, task Task, continueOnFail bool, command func(task Task) error) (err error) {
	semaphoreAcquired := false

	defer func() {
		r := job.taskResults[task.Id()]
		if err != nil {
			// set the error on the task result
			// so that the unblocked goroutines
			// can check if the task has failed
			r.err = err

			if !continueOnFail {

				// If we can't continue on fail, the first time we fail
				// we should simply set the flag about failed to true
				job.once.Do(func() {
					job.failed.Store(true)
				})

			}
		}

		close(r.done)

		if semaphoreAcquired {
			select {
			case job.semaphore <- struct{}{}:
			default:
			}
		}

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

	// Once you have acquired the semaphore, you need to check:
	// 1. if the overall job has failed due to continueOnFail flag being false and failure happening.
	// 2. if the caller context is done due to cancellation.
	if job.failed.Load() {
		return fmt.Errorf("returning early because overall job has failed")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return command(task)
}
