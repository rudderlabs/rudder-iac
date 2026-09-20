package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventsFiltersToPackage(t *testing.T) {
	in := strings.Join([]string{
		`{"Time":"2026-09-20T10:00:00Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestProjectApply"}`,
		`{"Time":"2026-09-20T10:00:01Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests/helpers","Test":"TestComparator"}`,
		`{"Time":"2026-09-20T10:00:02Z","Action":"pass","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestProjectApply"}`,
	}, "\n")

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)

	assert.Equal(t, []Event{
		{Time: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), Action: "run", Package: e2ePackage, Test: "TestProjectApply"},
		{Time: time.Date(2026, 9, 20, 10, 0, 2, 0, time.UTC), Action: "pass", Package: e2ePackage, Test: "TestProjectApply"},
	}, got)
}

func TestParseEventsRejectsParallelSubtest(t *testing.T) {
	in := strings.Join([]string{
		`{"Time":"2026-09-20T10:00:00Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestA"}`,
		`{"Time":"2026-09-20T10:00:01Z","Action":"pause","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestA"}`,
	}, "\n")

	_, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.ErrorIs(t, err, ErrParallelSubtest)
	assert.Contains(t, err.Error(), "TestA")
}

func TestParseEventsIgnoresParallelismOutsidePackage(t *testing.T) {
	// cli/tests/helpers is parallel throughout and drives no CLI, so its pause
	// events must not disqualify a run.
	in := `{"Time":"2026-09-20T10:00:00Z","Action":"pause","Package":"github.com/rudderlabs/rudder-iac/cli/tests/helpers","Test":"TestFileManager"}`

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestParseEventsSkipsPackageLevelEvents(t *testing.T) {
	in := `{"Time":"2026-09-20T10:00:00Z","Action":"output","Package":"github.com/rudderlabs/rudder-iac/cli/tests"}`

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)
	assert.Empty(t, got, "events with no Test name belong to the package, not a step")
}
