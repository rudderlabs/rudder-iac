package catalog

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTrackingPlanIdentifiersFetcher struct {
	calls []fetchCall
}

type fetchCall struct {
	id             string
	rebuildSchemas bool
}

func (m *mockTrackingPlanIdentifiersFetcher) GetTrackingPlanWithIdentifiers(
	_ context.Context,
	id string,
	rebuildSchemas bool,
) (*TrackingPlanWithIdentifiers, error) {
	m.calls = append(m.calls, fetchCall{id: id, rebuildSchemas: rebuildSchemas})
	return &TrackingPlanWithIdentifiers{
		TrackingPlan: TrackingPlan{ID: id},
	}, nil
}

func TestTrackingPlanWithIdentifiersFetchTask(t *testing.T) {
	t.Run("satisfies tasker.Task interface with nil dependencies", func(t *testing.T) {
		task := NewTrackingPlanWithIdentifiersFetchTask("tp-1", true)

		assert.Equal(t, "tp-1", task.Id())
		assert.Nil(t, task.Dependencies())
	})

	t.Run("calls GetTrackingPlanWithIdentifiers for each tracking plan with correct params", func(t *testing.T) {
		var (
			ctx     = context.Background()
			fetcher = &mockTrackingPlanIdentifiersFetcher{}
		)

		tasks := []tasker.Task{
			NewTrackingPlanWithIdentifiersFetchTask("tp-1", true),
			NewTrackingPlanWithIdentifiersFetchTask("tp-2", true),
		}

		results := tasker.NewResults[*TrackingPlanWithIdentifiers]()
		errs := tasker.RunTasks(
			ctx,
			tasks,
			1,
			false,
			RunTrackingPlanWithIdentifiersFetchTask(ctx, fetcher, results),
		)

		require.Empty(t, errs)
		require.Len(t, fetcher.calls, 2)

		expectedCalls := []fetchCall{
			{id: "tp-1", rebuildSchemas: true},
			{id: "tp-2", rebuildSchemas: true},
		}
		assert.ElementsMatch(t, expectedCalls, fetcher.calls)

		tp1, ok := results.Get("tp-1")
		require.True(t, ok)
		assert.Equal(t, "tp-1", tp1.ID)

		tp2, ok := results.Get("tp-2")
		require.True(t, ok)
		assert.Equal(t, "tp-2", tp2.ID)
	})

	t.Run("passes rebuildSchemas=false correctly", func(t *testing.T) {
		var (
			ctx     = context.Background()
			fetcher = &mockTrackingPlanIdentifiersFetcher{}
		)

		tasks := []tasker.Task{
			NewTrackingPlanWithIdentifiersFetchTask("tp-1", false),
		}

		results := tasker.NewResults[*TrackingPlanWithIdentifiers]()
		errs := tasker.RunTasks(
			ctx,
			tasks,
			1,
			false,
			RunTrackingPlanWithIdentifiersFetchTask(ctx, fetcher, results),
		)

		require.Empty(t, errs)
		require.Len(t, fetcher.calls, 1)
		assert.Equal(t, fetchCall{id: "tp-1", rebuildSchemas: false}, fetcher.calls[0])
	})
}
