package testutils

import (
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// MockReporter is a mock implementation of SyncReporter for testing
type MockReporter struct {
	mu sync.Mutex

	// Configuration
	ConfirmResponse           bool
	ConfirmError              error
	ConfirmNameMatchesResult  []differ.NameMatchCandidate // What to return from ConfirmNameMatches

	// Call tracking
	ReportPlanCalls           []*planner.Plan
	AskConfirmationCalls      int
	ConfirmNameMatchesCalls   [][]differ.NameMatchCandidate
	SyncStartedCalls          []int
	SyncCompletedCalls        int
	TaskStartedCalls          []TaskCall
	TaskCompletedCalls        []TaskCompletionCall
}

type TaskCall struct {
	TaskID      string
	Description string
}

type TaskCompletionCall struct {
	TaskID      string
	Description string
	Err         error
}

// NewMockReporter creates a new MockReporter with default confirmation response of true
func NewMockReporter() *MockReporter {
	return &MockReporter{
		ConfirmResponse:         true,
		ReportPlanCalls:         make([]*planner.Plan, 0),
		ConfirmNameMatchesCalls: make([][]differ.NameMatchCandidate, 0),
		SyncStartedCalls:        make([]int, 0),
		TaskStartedCalls:        make([]TaskCall, 0),
		TaskCompletedCalls:      make([]TaskCompletionCall, 0),
	}
}

func (m *MockReporter) ReportPlan(plan *planner.Plan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReportPlanCalls = append(m.ReportPlanCalls, plan)
}

func (m *MockReporter) AskConfirmation() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AskConfirmationCalls++
	return m.ConfirmResponse, m.ConfirmError
}

func (m *MockReporter) ConfirmNameMatches(matches []differ.NameMatchCandidate) []differ.NameMatchCandidate {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConfirmNameMatchesCalls = append(m.ConfirmNameMatchesCalls, matches)
	if m.ConfirmNameMatchesResult != nil {
		return m.ConfirmNameMatchesResult
	}
	// Default: return all matches (auto-confirm)
	return matches
}

func (m *MockReporter) SyncStarted(totalTasks int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SyncStartedCalls = append(m.SyncStartedCalls, totalTasks)
}

func (m *MockReporter) SyncCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SyncCompletedCalls++
}

func (m *MockReporter) TaskStarted(taskId string, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TaskStartedCalls = append(m.TaskStartedCalls, TaskCall{
		TaskID:      taskId,
		Description: description,
	})
}

func (m *MockReporter) TaskCompleted(taskId string, description string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TaskCompletedCalls = append(m.TaskCompletedCalls, TaskCompletionCall{
		TaskID:      taskId,
		Description: description,
		Err:         err,
	})
}

// Reset clears all recorded calls
func (m *MockReporter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReportPlanCalls = make([]*planner.Plan, 0)
	m.AskConfirmationCalls = 0
	m.ConfirmNameMatchesCalls = make([][]differ.NameMatchCandidate, 0)
	m.SyncStartedCalls = make([]int, 0)
	m.SyncCompletedCalls = 0
	m.TaskStartedCalls = make([]TaskCall, 0)
	m.TaskCompletedCalls = make([]TaskCompletionCall, 0)
}
