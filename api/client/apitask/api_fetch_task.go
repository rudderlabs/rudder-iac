package apitask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

type APIFetchTask[T any] struct {
	client *client.Client
	path   string
}

func NewAPIFetchTask[T any](client *client.Client, path string) *APIFetchTask[T] {
	return &APIFetchTask[T]{
		client: client,
		path:   path,
	}
}

func (t *APIFetchTask[T]) Id() string {
	return t.path
}

func (t *APIFetchTask[T]) Dependencies() []string {
	return nil
}

func (t *APIFetchTask[T]) Execute(ctx context.Context) (T, error) {
	var result T

	data, err := t.client.Do(ctx, "GET", t.path, nil)
	if err != nil {
		return *new(T), fmt.Errorf("failed to fetch data from API: %w", err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return *new(T), fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return result, nil
}

func RunAPIFetchTask[T any](ctx context.Context, results *tasker.Results[T]) func(task tasker.Task) error {
	return func(task tasker.Task) error {
		apiFetchTask, ok := task.(*APIFetchTask[T])
		if !ok {
			return fmt.Errorf("task is not an API fetch task")
		}

		resp, err := apiFetchTask.Execute(ctx)
		if err != nil {
			return fmt.Errorf("executing API fetch task: %w", err)
		}

		results.Store(apiFetchTask.Id(), resp)
		return nil
	}
}
