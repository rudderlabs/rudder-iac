package retl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// SyncType selects how much of the source a run reads. The API defaults to
// full when the field is absent; the CLI sends it explicitly so the recorded
// intent matches what ran.
type SyncType string

const (
	SyncTypeFull        SyncType = "full"
	SyncTypeIncremental SyncType = "incremental"
)

// SyncStatus is the API's verdict on the run itself, which is not the same as a
// verdict on the data: a run that finished with failed rows still reports
// SyncSucceeded, with the count in Metrics.Failed.Total. Use SyncOutcome to
// render one to a reader.
type SyncStatus string

const (
	SyncRunning   SyncStatus = "running"
	SyncSucceeded SyncStatus = "succeeded"
	SyncFailed    SyncStatus = "failed"
)

// SyncRowMetrics counts rows in one category of a run.
type SyncRowMetrics struct {
	Total int `json:"total"`
}

// SyncMetrics is the row accounting for a run. Failed is the field that decides
// whether a "succeeded" run actually delivered everything.
type SyncMetrics struct {
	Total     int            `json:"total"`
	Succeeded SyncRowMetrics `json:"succeeded"`
	Failed    SyncRowMetrics `json:"failed"`
	Changed   SyncRowMetrics `json:"changed"`
}

// Sync is one run of a connection.
type Sync struct {
	ID         string      `json:"id"`
	Status     SyncStatus  `json:"status"`
	StartedAt  time.Time   `json:"startedAt"`
	FinishedAt time.Time   `json:"finishedAt"`
	Error      string      `json:"error,omitempty"`
	Metrics    SyncMetrics `json:"metrics"`
}

// SyncsPage is the response from GET /v2/retl-connections/:id/syncs. The
// envelope key is "syncs", not "data": this endpoint predates the paginated
// shape the CRUD routes use, and it pages with per_page rather than pageSize.
type SyncsPage struct {
	Syncs  []Sync        `json:"syncs"`
	Paging client.Paging `json:"paging"`
}

// StartSyncResponse carries the id of the run that was just queued.
type StartSyncResponse struct {
	SyncID string `json:"syncId"`
}

// ListSyncsRequest filters a connection's run history. Zero values are omitted,
// so the empty request asks for the first page of everything.
type ListSyncsRequest struct {
	Page          int
	PerPage       int
	Status        SyncStatus
	StartedAfter  time.Time
	StartedBefore time.Time
}

// StartSync queues a run and returns its id. The data plane registers a newly
// created source asynchronously, so a source the CLI has just applied can
// answer "source not found" for a short while — that is a retry, not a failure.
func (r *RudderRETLStore) StartSync(ctx context.Context, connectionID string, syncType SyncType) (*StartSyncResponse, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("connection ID cannot be empty")
	}
	if syncType == "" {
		syncType = SyncTypeFull
	}

	data, err := json.Marshal(map[string]string{"syncType": string(syncType)})
	if err != nil {
		return nil, fmt.Errorf("marshalling sync request: %w", err)
	}

	path := fmt.Sprintf("%s/%s/start", retlConnectionsBasePath, connectionID)
	resp, err := r.client.Do(ctx, "POST", path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("starting sync for connection %q: %w", connectionID, err)
	}

	var result StartSyncResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return &result, nil
}

// StopSync cancels the run in progress. Without it a caller can start a sync it
// cannot cancel.
func (r *RudderRETLStore) StopSync(ctx context.Context, connectionID string) error {
	if connectionID == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/stop", retlConnectionsBasePath, connectionID)
	if _, err := r.client.Do(ctx, "POST", path, bytes.NewReader([]byte("{}"))); err != nil {
		return fmt.Errorf("stopping sync for connection %q: %w", connectionID, err)
	}
	return nil
}

// ListSyncs returns one page of a connection's run history, newest first.
func (r *RudderRETLStore) ListSyncs(ctx context.Context, connectionID string, req ListSyncsRequest) (*SyncsPage, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("connection ID cannot be empty")
	}

	query := url.Values{}
	if req.Page > 0 {
		query.Set("page", strconv.Itoa(req.Page))
	}
	if req.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(req.PerPage))
	}
	if req.Status != "" {
		query.Set("status", string(req.Status))
	}
	if !req.StartedAfter.IsZero() {
		query.Set("started_after", req.StartedAfter.UTC().Format(time.RFC3339))
	}
	if !req.StartedBefore.IsZero() {
		query.Set("started_before", req.StartedBefore.UTC().Format(time.RFC3339))
	}

	path := fmt.Sprintf("%s/%s/syncs", retlConnectionsBasePath, connectionID)
	if encoded := query.Encode(); encoded != "" {
		path = path + "?" + encoded
	}

	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing syncs for connection %q: %w", connectionID, err)
	}

	var page SyncsPage
	if err := json.Unmarshal(resp, &page); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return &page, nil
}

// GetSync returns one run. The API resolves it by filtering the history, and
// answers 404 rather than an empty page when the id is unknown.
func (r *RudderRETLStore) GetSync(ctx context.Context, connectionID, syncID string) (*Sync, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("connection ID cannot be empty")
	}
	if syncID == "" {
		return nil, fmt.Errorf("sync ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s/syncs/%s", retlConnectionsBasePath, connectionID, syncID)
	resp, err := r.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting sync %q for connection %q: %w", syncID, connectionID, err)
	}

	var sync Sync
	if err := json.Unmarshal(resp, &sync); err != nil {
		return nil, fmt.Errorf("unmarshalling response: %w", err)
	}
	return &sync, nil
}

// SyncOutcome is what a run means to a reader, which is not what Status says.
// The webapp derives the same four values in constructSyncData and never shows
// Status raw; the CLI has to agree with it or the two will describe the same
// run differently.
type SyncOutcome string

const (
	// OutcomeSyncing is a run still in progress.
	OutcomeSyncing SyncOutcome = "syncing"
	// OutcomeSucceeded is a finished run that delivered every row.
	OutcomeSucceeded SyncOutcome = "succeeded"
	// OutcomeFailed is a finished run that delivered some rows and dropped
	// others. The API calls this "succeeded".
	OutcomeFailed SyncOutcome = "failed"
	// OutcomeAborted is a run that died. The API calls this "failed".
	OutcomeAborted SyncOutcome = "aborted"
)

// Outcome maps the API's run status and the row counts onto what actually
// happened. The two names invert: an API "failed" is an aborted run, while an
// API "succeeded" carrying failed rows is a failed sync. Reporting Status
// directly tells an author their sync succeeded while rows were dropped.
func (s Sync) Outcome() SyncOutcome {
	switch s.Status {
	case SyncRunning:
		return OutcomeSyncing
	case SyncFailed:
		return OutcomeAborted
	case SyncSucceeded:
		if s.Metrics.Failed.Total > 0 {
			return OutcomeFailed
		}
		return OutcomeSucceeded
	default:
		// An unrecognised status is reported as-is rather than guessed at; a
		// new upstream state must not silently read as success.
		return SyncOutcome(s.Status)
	}
}

// Delivered reports whether the run finished having delivered every row, which
// is the question a script exiting non-zero actually wants answered.
func (s Sync) Delivered() bool {
	return s.Outcome() == OutcomeSucceeded
}

// Duration is how long the run took, measuring an unfinished run against now.
// FinishedAt is not meaningful until a run ends.
func (s Sync) Duration(now time.Time) time.Duration {
	if s.Status == SyncRunning || s.FinishedAt.IsZero() || s.FinishedAt.Before(s.StartedAt) {
		return now.Sub(s.StartedAt)
	}
	return s.FinishedAt.Sub(s.StartedAt)
}
