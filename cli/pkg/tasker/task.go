package tasker

import (
	"context"
	"sync"
)

type (
	taskStatusMap = map[string]chan struct{}
	semaphore     = chan struct{}
)

type Task interface {
	Id() string
	Dependencies() []string
}

type job struct {
	tasks          []Task
	taskStatus     taskStatusMap // To track whether a task has completed.
	semaphore      semaphore     // To control number of tasks running concurrently
	continueOnFail bool
}

// RunTasks creates a job and executes it. This is the main entry point for task execution.
func RunTasks(ctx context.Context, tasks []Task, concurrency int, continueOnFail bool, command func(task Task) error) []error {
	job := newJob(tasks, concurrency, continueOnFail)
	return job.run(ctx, command)
}

func newJob(tasks []Task, concurrency int, continueOnFail bool) *job {
	semaphore := make(chan struct{}, concurrency)
	for range concurrency {
		semaphore <- struct{}{}
	}

	job := &job{
		semaphore:      semaphore,
		tasks:          tasks,
		taskStatus:     make(taskStatusMap),
		continueOnFail: continueOnFail,
	}

	for _, task := range job.tasks {
		job.taskStatus[task.Id()] = make(chan struct{})
	}
	return job
}

func (job *job) run(ctx context.Context, command func(task Task) error) []error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errors []error
	var errorsMutex sync.Mutex

	var wg sync.WaitGroup
	for _, task := range job.tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			if err := job.runTask(ctx, cancel, t, command); err != nil {
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

func (job *job) runTask(ctx context.Context, cancelCtx context.CancelFunc, task Task, command func(task Task) error) (err error) {
	semaphoreAcquired := false

	defer func() {
		if err != nil && !job.continueOnFail {
			// cancellation should happen before releasing the semaphore
			// otherwise some other goroutine might start running a task
			cancelCtx()
		}
		if semaphoreAcquired {
			select {
			case job.semaphore <- struct{}{}:
			default:
			}
		}

		// Notify dependent tasks that this task has completed
		close(job.taskStatus[task.Id()])
	}()

	dependencies := task.Dependencies()
	// Wait for all dependencies to be completed
	for _, depTaskId := range dependencies {
		_, ok := job.taskStatus[depTaskId]
		// No need to return error if dependency not found in task list
		// This can happen if there are no operations for that dependency
		if ok {
			select {
			case <-job.taskStatus[depTaskId]:
			case <-ctx.Done():
				return ctx.Err()
			}

			// This double check is done because we can have race condition between
			// close of channel vs context cancellation. The consequence is that the waiting task
			// might run even though the continue on fail false setting is used.
			// The below check doubly checks if the context is done ( in case of race condition again )
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
	}

	select {
	case <-job.semaphore:
		semaphoreAcquired = true
	case <-ctx.Done():
		return ctx.Err()
	}
	return command(task)
}
