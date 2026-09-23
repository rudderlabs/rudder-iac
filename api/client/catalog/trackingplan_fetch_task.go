package catalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

type TrackingPlanIdentifiersFetcher interface {
	GetTrackingPlanWithIdentifiers(ctx context.Context, id string, rebuildSchemas bool) (*TrackingPlanWithIdentifiers, error)
}

type TrackingPlanWithIdentifiersFetchTask struct {
	trackingPlanID string
	rebuildSchemas bool
}

func NewTrackingPlanWithIdentifiersFetchTask(trackingPlanID string, rebuildSchemas bool) *TrackingPlanWithIdentifiersFetchTask {
	return &TrackingPlanWithIdentifiersFetchTask{
		trackingPlanID: trackingPlanID,
		rebuildSchemas: rebuildSchemas,
	}
}

func (t *TrackingPlanWithIdentifiersFetchTask) Id() string {
	return t.trackingPlanID
}

func (t *TrackingPlanWithIdentifiersFetchTask) Dependencies() []string {
	return nil
}

func RunTrackingPlanWithIdentifiersFetchTask(
	ctx context.Context,
	fetcher TrackingPlanIdentifiersFetcher,
	results *tasker.Results[*TrackingPlanWithIdentifiers],
) func(task tasker.Task) error {
	return func(task tasker.Task) error {
		fetchTask, ok := task.(*TrackingPlanWithIdentifiersFetchTask)
		if !ok {
			return fmt.Errorf("task is not a TrackingPlanWithIdentifiersFetchTask")
		}

		tp, err := fetcher.GetTrackingPlanWithIdentifiers(ctx, fetchTask.trackingPlanID, fetchTask.rebuildSchemas)
		if err != nil {
			return fmt.Errorf("fetching tracking plan %s: %w", fetchTask.trackingPlanID, err)
		}

		results.Store(fetchTask.Id(), tp)
		return nil
	}
}
